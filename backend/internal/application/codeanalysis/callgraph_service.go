package codeanalysis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cocursor/backend/internal/domain/codeanalysis"
	infra "github.com/cocursor/backend/internal/infrastructure/codeanalysis"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/google/uuid"
)

// CallGraphService 调用图服务
type CallGraphService struct {
	logger            *slog.Logger
	projectService    *ProjectService
	ssaAnalyzer       *infra.SSAAnalyzer
	callGraphRepo     *infra.CallGraphRepository
	callGraphManager  *infra.CallGraphManager
	entryPointScanner *infra.EntryPointScanner
	worktreeManager   *infra.WorktreeManager

	// 异步任务管理
	tasksMu sync.RWMutex
	tasks   map[string]*codeanalysis.GenerationTask
}

// NewCallGraphService 创建调用图服务
func NewCallGraphService(
	projectService *ProjectService,
	ssaAnalyzer *infra.SSAAnalyzer,
	callGraphRepo *infra.CallGraphRepository,
	callGraphManager *infra.CallGraphManager,
	entryPointScanner *infra.EntryPointScanner,
) *CallGraphService {
	return &CallGraphService{
		logger:            log.NewModuleLogger("codeanalysis", "callgraph_service"),
		projectService:    projectService,
		ssaAnalyzer:       ssaAnalyzer,
		callGraphRepo:     callGraphRepo,
		callGraphManager:  callGraphManager,
		entryPointScanner: entryPointScanner,
		worktreeManager:   infra.NewWorktreeManager(),
		tasks:             make(map[string]*codeanalysis.GenerationTask),
	}
}

// CheckStatusRequest 检查状态请求
type CheckStatusRequest struct {
	ProjectPath string `json:"project_path"`
	Commit      string `json:"commit"`
}

// CheckStatus 检查调用图状态
func (s *CallGraphService) CheckStatus(ctx context.Context, req *CheckStatusRequest) (*codeanalysis.CallGraphStatus, error) {
	absPath, err := filepath.Abs(req.ProjectPath)
	if err != nil {
		return nil, err
	}

	// 首先验证 Go 模块（支持全栈项目，go.mod 可能在子目录中）
	validation := s.entryPointScanner.ValidateGoModule(ctx, absPath)
	if !validation.Valid {
		// 不是有效的 Go 模块，返回验证错误状态
		return &codeanalysis.CallGraphStatus{
			Exists:            false,
			ProjectRegistered: false,
			ValidGoModule:     false,
			GoModuleError:     validation.Error,
		}, nil
	}

	// 获取项目配置（使用原始路径和 go.mod 所在路径均尝试查找）
	project, err := s.projectService.GetProject(ctx, absPath)
	if err != nil && validation.GoModDir != "" && validation.GoModDir != absPath {
		// 原始路径未找到，尝试使用 go.mod 所在目录查找
		project, err = s.projectService.GetProject(ctx, validation.GoModDir)
	}
	if err != nil {
		// 项目未注册，但是有效的 Go 模块
		return &codeanalysis.CallGraphStatus{
			Exists:            false,
			ProjectRegistered: false,
			ValidGoModule:     true,
		}, nil
	}

	// 获取调用图状态
	status, err := s.callGraphManager.GetCallGraphStatus(ctx, project.ID, absPath, req.Commit)
	if err != nil {
		return nil, err
	}

	// 设置 Go 模块验证状态
	status.ValidGoModule = true

	return status, nil
}

// GenerateRequest 生成调用图请求
type GenerateRequest struct {
	ProjectPath string `json:"project_path"`
	Commit      string `json:"commit"`
}

// GenerateWithConfigRequest 生成调用图请求（包含配置）
type GenerateWithConfigRequest struct {
	ProjectPath        string                     `json:"project_path"`
	EntryPoints        []string                   `json:"entry_points"`
	Exclude            []string                   `json:"exclude"`
	Algorithm          codeanalysis.AlgorithmType `json:"algorithm"`
	Commit             string                     `json:"commit"`
	IntegrationTestDir string                     `json:"integration_test_dir"`
	IntegrationTestTag string                     `json:"integration_test_tag"`
}

// GenerateResponse 生成调用图响应
type GenerateResponse struct {
	Commit           string `json:"commit"`
	FuncCount        int    `json:"func_count"`
	EdgeCount        int    `json:"edge_count"`
	GenerationTimeMs int64  `json:"generation_time_ms"`
	DBPath           string `json:"db_path"`
	// ActualAlgorithm 实际使用的算法（可能因降级而与配置的算法不同）
	ActualAlgorithm string `json:"actual_algorithm,omitempty"`
	// Fallback 是否发生了算法降级
	Fallback bool `json:"fallback,omitempty"`
	// FallbackReason 降级原因
	FallbackReason string `json:"fallback_reason,omitempty"`
}

// Generate 生成调用图（同步）
func (s *CallGraphService) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	startTime := time.Now()

	absPath, err := filepath.Abs(req.ProjectPath)
	if err != nil {
		return nil, err
	}

	// 获取项目配置
	project, err := s.projectService.GetProject(ctx, absPath)
	if err != nil {
		return nil, fmt.Errorf("project not registered: %w", err)
	}

	// 获取当前 commit
	commit := req.Commit
	if commit == "" || commit == "HEAD" {
		diffAnalyzer := infra.NewDiffAnalyzer()
		commit, err = diffAnalyzer.GetCurrentCommit(ctx, absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get current commit: %w", err)
		}
	}

	// 获取分支名
	branch := s.getCurrentBranch(ctx, absPath)

	s.logger.Info("generating call graph",
		"project", project.Name,
		"commit", commit,
		"algorithm", project.Algorithm,
	)

	// 执行 SSA 分析
	result, err := s.ssaAnalyzer.Analyze(ctx, absPath, project.EntryPoints, project.Algorithm)
	if err != nil {
		return nil, fmt.Errorf("SSA analysis failed: %w", err)
	}

	// 获取数据库路径并初始化
	dbPath := s.callGraphManager.GetCommitDBPath(project.ID, commit)
	if err := s.callGraphRepo.Init(ctx, dbPath); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	// 保存函数节点
	if err := s.callGraphRepo.SaveFuncNodes(ctx, dbPath, result.FuncNodes); err != nil {
		return nil, fmt.Errorf("failed to save func nodes: %w", err)
	}

	// 保存调用边
	if err := s.callGraphRepo.SaveFuncEdges(ctx, dbPath, result.FuncEdges); err != nil {
		return nil, fmt.Errorf("failed to save func edges: %w", err)
	}

	// 保存元数据
	s.callGraphRepo.SaveMetadata(ctx, dbPath, "commit", commit)
	s.callGraphRepo.SaveMetadata(ctx, dbPath, "algorithm", string(project.Algorithm))
	s.callGraphRepo.SaveMetadata(ctx, dbPath, "module_path", result.ModulePath)
	s.callGraphRepo.SaveMetadata(ctx, dbPath, "created_at", time.Now().Format(time.RFC3339))

	generationTime := time.Since(startTime).Milliseconds()

	// 保存调用图元信息
	cg := &codeanalysis.CallGraph{
		Commit:           commit,
		Branch:           branch,
		Algorithm:        project.Algorithm,
		FuncCount:        len(result.FuncNodes),
		EdgeCount:        len(result.FuncEdges),
		DBPath:           dbPath,
		CreatedAt:        time.Now(),
		GenerationTimeMs: generationTime,
	}

	// 获取远程 URL
	remoteURL, _ := s.entryPointScanner.GetRemoteURL(ctx, absPath)

	if err := s.callGraphManager.SaveCallGraph(ctx, project.ID, project.Name, remoteURL, cg); err != nil {
		return nil, fmt.Errorf("failed to save call graph metadata: %w", err)
	}

	s.logger.Info("call graph generated",
		"project", project.Name,
		"commit", commit,
		"func_count", cg.FuncCount,
		"edge_count", cg.EdgeCount,
		"generation_time_ms", generationTime,
		"fallback", result.Fallback,
	)

	return &GenerateResponse{
		Commit:           commit,
		FuncCount:        cg.FuncCount,
		EdgeCount:        cg.EdgeCount,
		GenerationTimeMs: generationTime,
		DBPath:           dbPath,
		ActualAlgorithm:  string(result.ActualAlgorithm),
		Fallback:         result.Fallback,
		FallbackReason:   result.FallbackReason,
	}, nil
}

// GenerateAsyncResponse 异步生成响应
type GenerateAsyncResponse struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

// GenerateAsync 生成调用图（异步）
func (s *CallGraphService) GenerateAsync(ctx context.Context, req *GenerateRequest) (*GenerateAsyncResponse, error) {
	absPath, err := filepath.Abs(req.ProjectPath)
	if err != nil {
		return nil, err
	}

	// 获取项目配置
	project, err := s.projectService.GetProject(ctx, absPath)
	if err != nil {
		return nil, fmt.Errorf("project not registered: %w", err)
	}

	// 创建任务
	taskID := uuid.New().String()
	now := time.Now()

	task := &codeanalysis.GenerationTask{
		TaskID:      taskID,
		ProjectID:   project.ID,
		ProjectPath: absPath,
		Commit:      req.Commit,
		Status:      "pending",
		Progress:    0,
		Message:     "Task created",
		StartedAt:   &now,
	}

	s.tasksMu.Lock()
	s.tasks[taskID] = task
	s.tasksMu.Unlock()

	// 在后台执行
	go s.runGenerationTask(context.Background(), task, project)

	return &GenerateAsyncResponse{
		TaskID: taskID,
		Status: "pending",
	}, nil
}

// GenerateWithConfigAsync 生成调用图（异步，包含配置）
func (s *CallGraphService) GenerateWithConfigAsync(ctx context.Context, req *GenerateWithConfigRequest) (*GenerateAsyncResponse, error) {
	absPath, err := filepath.Abs(req.ProjectPath)
	if err != nil {
		return nil, err
	}

	if len(req.EntryPoints) == 0 {
		return nil, fmt.Errorf("entry_points is required")
	}

	// 验证 Go 模块（支持全栈项目，go.mod 可能在子目录中）
	validation := s.entryPointScanner.ValidateGoModule(ctx, absPath)
	if !validation.Valid {
		return nil, fmt.Errorf("invalid Go module: %s", validation.Error)
	}

	// 确定实际的 Go 模块根目录（用于 SSA 分析）
	goModDir := absPath
	if validation.GoModDir != "" && validation.GoModDir != absPath {
		goModDir = validation.GoModDir
		s.logger.Info("using go.mod subdirectory for SSA analysis",
			"original_path", absPath,
			"go_mod_dir", goModDir,
		)
	}

	// 如果是全栈项目，需要将入口函数路径调整为相对于 Go 模块根目录
	adjustedReq := req
	if goModDir != absPath {
		relPrefix, _ := filepath.Rel(absPath, goModDir)
		relPrefix = filepath.ToSlash(relPrefix)
		adjustedEntryPoints := make([]string, len(req.EntryPoints))
		for i, ep := range req.EntryPoints {
			// 入口函数格式: file:func，如 backend/cmd/main.go:main
			// 需要去掉 backend/ 前缀，变为 cmd/main.go:main
			if strings.HasPrefix(ep, relPrefix+"/") {
				adjustedEntryPoints[i] = strings.TrimPrefix(ep, relPrefix+"/")
			} else {
				adjustedEntryPoints[i] = ep
			}
		}
		adjustedReq = &GenerateWithConfigRequest{
			ProjectPath:        req.ProjectPath,
			EntryPoints:        adjustedEntryPoints,
			Exclude:            req.Exclude,
			Algorithm:          req.Algorithm,
			Commit:             req.Commit,
			IntegrationTestDir: req.IntegrationTestDir,
			IntegrationTestTag: req.IntegrationTestTag,
		}
	}

	project, err := s.buildProjectFromConfig(ctx, absPath, adjustedReq)
	if err != nil {
		return nil, err
	}

	// 创建任务（使用 Go 模块目录作为分析路径）
	taskID := uuid.New().String()
	now := time.Now()

	task := &codeanalysis.GenerationTask{
		TaskID:      taskID,
		ProjectID:   project.ID,
		ProjectPath: goModDir,
		Commit:      req.Commit,
		Status:      "pending",
		Progress:    0,
		Message:     "Task created",
		StartedAt:   &now,
	}

	s.tasksMu.Lock()
	s.tasks[taskID] = task
	s.tasksMu.Unlock()

	// 在后台执行
	go s.runGenerationTaskWithConfig(context.Background(), task, project, req)

	return &GenerateAsyncResponse{
		TaskID: taskID,
		Status: "pending",
	}, nil
}

// runGenerationTask 执行生成任务
func (s *CallGraphService) runGenerationTask(ctx context.Context, task *codeanalysis.GenerationTask, project *codeanalysis.Project) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("generation task panicked",
				"task_id", task.TaskID,
				"panic", fmt.Sprintf("%v", r),
			)
			s.updateTask(task.TaskID, func(t *codeanalysis.GenerationTask) {
				t.Status = "failed"
				t.Error = fmt.Sprintf("panic: %v", r)
				now := time.Now()
				t.CompletedAt = &now
			})
		}
	}()

	// 更新状态为运行中
	s.updateTask(task.TaskID, func(t *codeanalysis.GenerationTask) {
		t.Status = "running"
		t.Progress = 5
		t.Message = "Loading packages..."
	})

	// 使用带进度回调的生成方法
	resp, err := s.generateWithProgress(ctx, &GenerateRequest{
		ProjectPath: task.ProjectPath,
		Commit:      task.Commit,
	}, func(progress int, message string) {
		s.updateTask(task.TaskID, func(t *codeanalysis.GenerationTask) {
			t.Progress = progress
			t.Message = message
		})
	})

	now := time.Now()

	if err != nil {
		s.updateTask(task.TaskID, func(t *codeanalysis.GenerationTask) {
			t.Status = "failed"
			t.Progress = 100
			s.applyTaskError(t, err)
			t.CompletedAt = &now
		})
		return
	}

	// 更新为完成
	s.updateTask(task.TaskID, func(t *codeanalysis.GenerationTask) {
		t.Status = "completed"
		t.Progress = 100
		if resp.Fallback {
			t.Message = "Generation completed (algorithm fallback occurred)"
		} else {
			t.Message = "Generation completed"
		}
		t.CompletedAt = &now
		t.Result = &codeanalysis.CallGraph{
			Commit:           resp.Commit,
			FuncCount:        resp.FuncCount,
			EdgeCount:        resp.EdgeCount,
			DBPath:           resp.DBPath,
			GenerationTimeMs: resp.GenerationTimeMs,
			ActualAlgorithm:  codeanalysis.AlgorithmType(resp.ActualAlgorithm),
			Fallback:         resp.Fallback,
			FallbackReason:   resp.FallbackReason,
		}
	})
}

// runGenerationTaskWithConfig 执行生成任务（包含配置）
func (s *CallGraphService) runGenerationTaskWithConfig(ctx context.Context, task *codeanalysis.GenerationTask, project *codeanalysis.Project, req *GenerateWithConfigRequest) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("generation task panicked",
				"task_id", task.TaskID,
				"panic", fmt.Sprintf("%v", r),
			)
			s.updateTask(task.TaskID, func(t *codeanalysis.GenerationTask) {
				t.Status = "failed"
				t.Error = fmt.Sprintf("panic: %v", r)
				now := time.Now()
				t.CompletedAt = &now
			})
		}
	}()

	// 更新状态为运行中
	s.updateTask(task.TaskID, func(t *codeanalysis.GenerationTask) {
		t.Status = "running"
		t.Progress = 5
		t.Message = "Loading packages..."
	})

	// 使用带进度回调的生成方法
	resp, err := s.generateWithProgressByProject(ctx, &GenerateRequest{
		ProjectPath: task.ProjectPath,
		Commit:      task.Commit,
	}, project, func(progress int, message string) {
		s.updateTask(task.TaskID, func(t *codeanalysis.GenerationTask) {
			t.Progress = progress
			t.Message = message
		})
	})

	now := time.Now()

	if err != nil {
		s.logger.Error("generation task failed",
			"task_id", task.TaskID,
			"error", err,
		)
		s.updateTask(task.TaskID, func(t *codeanalysis.GenerationTask) {
			t.Status = "failed"
			t.Progress = 100
			s.applyTaskError(t, err)
			t.CompletedAt = &now
		})
		return
	}

	// 生成成功后保存配置
	_, registerErr := s.projectService.RegisterProject(ctx, &RegisterProjectRequest{
		ProjectPath:        task.ProjectPath,
		EntryPoints:        req.EntryPoints,
		Exclude:            req.Exclude,
		Algorithm:          req.Algorithm,
		IntegrationTestDir: req.IntegrationTestDir,
		IntegrationTestTag: req.IntegrationTestTag,
	})
	if registerErr != nil {
		s.logger.Warn("failed to save project config after generation", "error", registerErr)
	}

	// 更新为完成
	s.updateTask(task.TaskID, func(t *codeanalysis.GenerationTask) {
		t.Status = "completed"
		t.Progress = 100
		t.Message = "Generation completed"
		t.CompletedAt = &now
		t.Result = &codeanalysis.CallGraph{
			Commit:           resp.Commit,
			FuncCount:        resp.FuncCount,
			EdgeCount:        resp.EdgeCount,
			DBPath:           resp.DBPath,
			GenerationTimeMs: resp.GenerationTimeMs,
			ActualAlgorithm:  codeanalysis.AlgorithmType(resp.ActualAlgorithm),
			Fallback:         resp.Fallback,
			FallbackReason:   resp.FallbackReason,
		}
	})
}

// ProgressCallback 进度回调函数类型
type ProgressCallback func(progress int, message string)

// generateWithProgress 带进度回调的生成方法
func (s *CallGraphService) generateWithProgress(ctx context.Context, req *GenerateRequest, onProgress ProgressCallback) (*GenerateResponse, error) {
	absPath, err := filepath.Abs(req.ProjectPath)
	if err != nil {
		return nil, err
	}

	// 获取项目配置
	onProgress(10, "Loading project configuration...")
	project, err := s.projectService.GetProject(ctx, absPath)
	if err != nil {
		return nil, fmt.Errorf("project not registered: %w", err)
	}

	return s.generateWithProgressByProject(ctx, req, project, onProgress)
}

func (s *CallGraphService) generateWithProgressByProject(ctx context.Context, req *GenerateRequest, project *codeanalysis.Project, onProgress ProgressCallback) (*GenerateResponse, error) {
	startTime := time.Now()
	absPath, err := filepath.Abs(req.ProjectPath)
	if err != nil {
		return nil, err
	}

	// 判断是否需要 worktree
	isHead, err := s.worktreeManager.IsHeadCommit(ctx, absPath, req.Commit)
	if err != nil {
		return nil, fmt.Errorf("failed to check commit: %w", err)
	}

	// SSA 分析的目标路径（可能是 worktree 路径）
	analysisPath := absPath
	commit := req.Commit

	if !isHead {
		// 非 HEAD commit，需要创建 worktree
		onProgress(10, "Creating worktree for base commit...")
		wtResult, err := s.worktreeManager.CreateWorktree(ctx, absPath, req.Commit)
		if err != nil {
			return nil, fmt.Errorf("failed to create worktree: %w", err)
		}
		analysisPath = wtResult.WorktreePath
		commit = wtResult.ResolvedCommit
		// 确保分析完成后清理 worktree
		defer func() {
			onProgress(98, "Cleaning up worktree...")
			if rmErr := s.worktreeManager.RemoveWorktree(ctx, absPath, wtResult.WorktreePath); rmErr != nil {
				s.logger.Warn("failed to remove worktree", "error", rmErr)
			}
		}()
		onProgress(15, "Worktree ready, resolving commit...")
	} else {
		// HEAD commit，直接获取完整 hash
		onProgress(15, "Getting current commit...")
		if commit == "" || commit == "HEAD" {
			diffAnalyzer := infra.NewDiffAnalyzer()
			commit, err = diffAnalyzer.GetCurrentCommit(ctx, absPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get current commit: %w", err)
			}
		}
	}

	// 获取分支名
	branch := s.getCurrentBranch(ctx, absPath)

	s.logger.Info("generating call graph with progress",
		"project", project.Name,
		"commit", commit,
		"algorithm", project.Algorithm,
		"analysis_path", analysisPath,
		"using_worktree", !isHead,
	)

	// 执行 SSA 分析（这是最耗时的步骤，占 20%-80%）
	// 将 SSA 内部进度（0-100%）映射到整体进度（20%-80%）
	onProgress(20, "Loading and building SSA...")
	result, err := s.ssaAnalyzer.AnalyzeWithProgress(ctx, analysisPath, project.EntryPoints, project.Algorithm,
		func(ssaProgress int, ssaMessage string) {
			// SSA 0-100% → 整体 20%-80%
			overallProgress := 20 + ssaProgress*60/100
			onProgress(overallProgress, ssaMessage)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("SSA analysis failed: %w", err)
	}

	// 初始化数据库
	onProgress(80, "Initializing database...")
	dbPath := s.callGraphManager.GetCommitDBPath(project.ID, commit)
	if err := s.callGraphRepo.Init(ctx, dbPath); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	// 保存函数节点
	onProgress(85, fmt.Sprintf("Saving %d functions...", len(result.FuncNodes)))
	if err := s.callGraphRepo.SaveFuncNodes(ctx, dbPath, result.FuncNodes); err != nil {
		return nil, fmt.Errorf("failed to save func nodes: %w", err)
	}

	// 保存调用边
	onProgress(90, fmt.Sprintf("Saving %d call edges...", len(result.FuncEdges)))
	if err := s.callGraphRepo.SaveFuncEdges(ctx, dbPath, result.FuncEdges); err != nil {
		return nil, fmt.Errorf("failed to save func edges: %w", err)
	}

	// 保存元数据
	onProgress(95, "Saving metadata...")
	s.callGraphRepo.SaveMetadata(ctx, dbPath, "commit", commit)
	s.callGraphRepo.SaveMetadata(ctx, dbPath, "algorithm", string(project.Algorithm))
	s.callGraphRepo.SaveMetadata(ctx, dbPath, "module_path", result.ModulePath)
	s.callGraphRepo.SaveMetadata(ctx, dbPath, "created_at", time.Now().Format(time.RFC3339))

	generationTime := time.Since(startTime).Milliseconds()

	// 保存调用图元信息
	cg := &codeanalysis.CallGraph{
		Commit:           commit,
		Branch:           branch,
		Algorithm:        project.Algorithm,
		FuncCount:        len(result.FuncNodes),
		EdgeCount:        len(result.FuncEdges),
		DBPath:           dbPath,
		CreatedAt:        time.Now(),
		GenerationTimeMs: generationTime,
	}

	// 获取远程 URL
	remoteURL, _ := s.entryPointScanner.GetRemoteURL(ctx, absPath)

	if err := s.callGraphManager.SaveCallGraph(ctx, project.ID, project.Name, remoteURL, cg); err != nil {
		return nil, fmt.Errorf("failed to save call graph metadata: %w", err)
	}

	s.logger.Info("call graph generated with progress",
		"project", project.Name,
		"commit", commit,
		"func_count", cg.FuncCount,
		"edge_count", cg.EdgeCount,
		"generation_time_ms", generationTime,
		"fallback", result.Fallback,
	)

	return &GenerateResponse{
		Commit:           commit,
		FuncCount:        cg.FuncCount,
		EdgeCount:        cg.EdgeCount,
		GenerationTimeMs: generationTime,
		DBPath:           dbPath,
		ActualAlgorithm:  string(result.ActualAlgorithm),
		Fallback:         result.Fallback,
		FallbackReason:   result.FallbackReason,
	}, nil
}

// buildProjectFromConfig 构建生成所需的项目配置（不落盘）
func (s *CallGraphService) buildProjectFromConfig(ctx context.Context, absPath string, req *GenerateWithConfigRequest) (*codeanalysis.Project, error) {
	remoteURL, err := s.entryPointScanner.GetRemoteURL(ctx, absPath)
	if err != nil {
		s.logger.Warn("failed to get remote URL, using path as identifier", "error", err)
		remoteURL = absPath
	}

	projectName := filepath.Base(absPath)
	projectID := infra.GetProjectID(remoteURL)

	// 尝试读取已有配置
	existing, _ := s.projectService.GetProject(ctx, absPath)
	localPaths := []string{absPath}
	createdAt := time.Time{}
	if existing != nil {
		localPaths = existing.LocalPaths
		createdAt = existing.CreatedAt
		// 合并本地路径
		pathSet := make(map[string]bool)
		for _, p := range existing.LocalPaths {
			pathSet[p] = true
		}
		pathSet[absPath] = true
		localPaths = make([]string, 0, len(pathSet))
		for p := range pathSet {
			localPaths = append(localPaths, p)
		}
	}

	// 集成测试配置：优先使用请求中的值，否则保留已有配置
	integrationTestDir := req.IntegrationTestDir
	integrationTestTag := req.IntegrationTestTag
	if existing != nil {
		if integrationTestDir == "" {
			integrationTestDir = existing.IntegrationTestDir
		}
		if integrationTestTag == "" {
			integrationTestTag = existing.IntegrationTestTag
		}
	}

	project := &codeanalysis.Project{
		ID:                 projectID,
		Name:               projectName,
		RemoteURL:          remoteURL,
		LocalPaths:         localPaths,
		EntryPoints:        req.EntryPoints,
		Exclude:            req.Exclude,
		Algorithm:          req.Algorithm,
		IntegrationTestDir: integrationTestDir,
		IntegrationTestTag: integrationTestTag,
		CreatedAt:          createdAt,
	}

	// 设置默认值
	if len(project.Exclude) == 0 {
		project.Exclude = []string{"vendor/", "*_test.go"}
	}
	if project.Algorithm == "" {
		project.Algorithm = codeanalysis.AlgorithmRTA
	}

	return project, nil
}

// applyTaskError 填充任务错误信息
func (s *CallGraphService) applyTaskError(task *codeanalysis.GenerationTask, err error) {
	if task == nil || err == nil {
		return
	}

	var algoErr *codeanalysis.AlgorithmFailedError
	if errors.As(err, &algoErr) {
		task.Error = algoErr.Error()
		task.ErrorCode = codeanalysis.ErrorCodeAlgorithmFailed
		task.Suggestion = algoErr.Suggestion
		task.Details = algoErr.Details
		return
	}

	task.Error = err.Error()
}

// updateTask 更新任务
func (s *CallGraphService) updateTask(taskID string, update func(*codeanalysis.GenerationTask)) {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()

	if task, ok := s.tasks[taskID]; ok {
		update(task)
	}
}

// GetTaskProgress 获取任务进度
func (s *CallGraphService) GetTaskProgress(taskID string) (*codeanalysis.GenerationTask, error) {
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// 返回副本
	result := *task
	return &result, nil
}

// getCurrentBranch 获取当前分支
func (s *CallGraphService) getCurrentBranch(ctx context.Context, projectPath string) string {
	diffAnalyzer := infra.NewDiffAnalyzer()
	// 使用 git symbolic-ref 获取分支名
	// 这里简化处理，返回空字符串
	_, _ = diffAnalyzer.GetCurrentCommit(ctx, projectPath)
	return ""
}
