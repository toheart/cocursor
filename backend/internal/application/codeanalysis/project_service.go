package codeanalysis

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/cocursor/backend/internal/domain/codeanalysis"
	infra "github.com/cocursor/backend/internal/infrastructure/codeanalysis"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// ProjectService 项目管理服务
type ProjectService struct {
	logger            *slog.Logger
	projectStore      *infra.ProjectStore
	entryPointScanner *infra.EntryPointScanner
}

// NewProjectService 创建项目管理服务
func NewProjectService(
	projectStore *infra.ProjectStore,
	entryPointScanner *infra.EntryPointScanner,
) *ProjectService {
	return &ProjectService{
		logger:            log.NewModuleLogger("codeanalysis", "project_service"),
		projectStore:      projectStore,
		entryPointScanner: entryPointScanner,
	}
}

// ScanEntryPointsRequest 扫描入口函数请求
type ScanEntryPointsRequest struct {
	ProjectPath string `json:"project_path"`
}

// ScanEntryPointsResponse 扫描入口函数响应
type ScanEntryPointsResponse struct {
	ProjectName    string                             `json:"project_name"`
	RemoteURL      string                             `json:"remote_url"`
	Candidates     []codeanalysis.EntryPointCandidate `json:"candidates"`
	DefaultExclude []string                           `json:"default_exclude"`
}

// ScanEntryPoints 扫描项目中的入口函数
// 支持全栈项目：当 go.mod 不在根目录而在子目录（如 backend/）时，自动使用子目录进行扫描
func (s *ProjectService) ScanEntryPoints(ctx context.Context, req *ScanEntryPointsRequest) (*ScanEntryPointsResponse, error) {
	absPath, err := filepath.Abs(req.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("invalid project path: %w", err)
	}

	// 验证 Go 模块
	validation := s.entryPointScanner.ValidateGoModule(ctx, absPath)
	if !validation.Valid {
		s.logger.Warn("Go module validation failed during scan",
			"path", absPath,
			"error", validation.Error,
		)
		return nil, fmt.Errorf("invalid Go module: %s", validation.Error)
	}

	// 确定实际的扫描路径（可能是子目录，如全栈项目的 backend/）
	scanPath := absPath
	if validation.GoModDir != "" && validation.GoModDir != absPath {
		s.logger.Info("go.mod found in subdirectory, using it as scan root",
			"original_path", absPath,
			"go_mod_dir", validation.GoModDir,
		)
		scanPath = validation.GoModDir
	}

	// 扫描入口函数
	candidates, err := s.entryPointScanner.ScanEntryPoints(ctx, scanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan entry points: %w", err)
	}

	// 如果扫描路径与原始路径不同（全栈项目），调整候选项的文件路径为相对于原始项目根目录
	if scanPath != absPath {
		relPrefix, _ := filepath.Rel(absPath, scanPath)
		relPrefix = filepath.ToSlash(relPrefix)
		for i := range candidates {
			if candidates[i].File != "*" {
				candidates[i].File = relPrefix + "/" + candidates[i].File
			}
		}
	}

	// 获取项目名称和远程 URL（使用原始路径，因为 .git 在项目根目录）
	projectName := filepath.Base(absPath)
	remoteURL, _ := s.entryPointScanner.GetRemoteURL(ctx, absPath)

	return &ScanEntryPointsResponse{
		ProjectName:    projectName,
		RemoteURL:      remoteURL,
		Candidates:     candidates,
		DefaultExclude: []string{"vendor/", "*_test.go"},
	}, nil
}

// RegisterProjectRequest 注册项目请求
type RegisterProjectRequest struct {
	ProjectPath        string                     `json:"project_path"`
	EntryPoints        []string                   `json:"entry_points"`
	Exclude            []string                   `json:"exclude"`
	Algorithm          codeanalysis.AlgorithmType `json:"algorithm"`
	IntegrationTestDir string                     `json:"integration_test_dir"`
	IntegrationTestTag string                     `json:"integration_test_tag"`
}

// RegisterProjectResponse 注册项目响应
type RegisterProjectResponse struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	IsNew       bool   `json:"is_new"`
}

// RegisterProject 注册或更新项目
func (s *ProjectService) RegisterProject(ctx context.Context, req *RegisterProjectRequest) (*RegisterProjectResponse, error) {
	absPath, err := filepath.Abs(req.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("invalid project path: %w", err)
	}

	// 验证 Go 模块
	validation := s.entryPointScanner.ValidateGoModule(ctx, absPath)
	if !validation.Valid {
		s.logger.Warn("Go module validation failed",
			"path", absPath,
			"error", validation.Error,
		)
		return nil, fmt.Errorf("invalid Go module: %s", validation.Error)
	}

	s.logger.Info("Go module validated",
		"path", absPath,
		"module", validation.ModulePath,
	)

	// 获取远程 URL
	remoteURL, err := s.entryPointScanner.GetRemoteURL(ctx, absPath)
	if err != nil {
		s.logger.Warn("failed to get remote URL, using path as identifier", "error", err)
		remoteURL = absPath
	}

	projectName := filepath.Base(absPath)
	projectID := infra.GetProjectID(remoteURL)

	// 检查是否已存在
	existing, err := s.projectStore.GetByID(ctx, projectID)
	isNew := err != nil || existing == nil

	// 创建或更新项目配置
	project := &codeanalysis.Project{
		ID:                 projectID,
		Name:               projectName,
		RemoteURL:          remoteURL,
		LocalPaths:         []string{absPath},
		EntryPoints:        req.EntryPoints,
		Exclude:            req.Exclude,
		Algorithm:          req.Algorithm,
		IntegrationTestDir: req.IntegrationTestDir,
		IntegrationTestTag: req.IntegrationTestTag,
	}

	// 如果是更新，保留现有的本地路径
	if !isNew {
		// 合并本地路径
		pathSet := make(map[string]bool)
		for _, p := range existing.LocalPaths {
			pathSet[p] = true
		}
		pathSet[absPath] = true

		project.LocalPaths = make([]string, 0, len(pathSet))
		for p := range pathSet {
			project.LocalPaths = append(project.LocalPaths, p)
		}
		project.CreatedAt = existing.CreatedAt
	}

	// 设置默认值
	if len(project.Exclude) == 0 {
		project.Exclude = []string{"vendor/", "*_test.go"}
	}
	if project.Algorithm == "" {
		project.Algorithm = codeanalysis.AlgorithmRTA
	}

	if err := s.projectStore.Save(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	s.logger.Info("project registered",
		"project_id", projectID,
		"project_name", projectName,
		"is_new", isNew,
	)

	return &RegisterProjectResponse{
		ProjectID:   projectID,
		ProjectName: projectName,
		IsNew:       isNew,
	}, nil
}

// GetProject 获取项目配置
func (s *ProjectService) GetProject(ctx context.Context, projectPath string) (*codeanalysis.Project, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}

	// 优先通过路径查找
	project, err := s.projectStore.GetByPath(ctx, absPath)
	if err == nil {
		return project, nil
	}

	// 尝试通过 remote URL 查找
	remoteURL, err := s.entryPointScanner.GetRemoteURL(ctx, absPath)
	if err != nil {
		return nil, fmt.Errorf("project not found and failed to get remote URL: %w", err)
	}

	project, err = s.projectStore.GetByRemoteURL(ctx, remoteURL)
	if err != nil {
		return nil, fmt.Errorf("project not registered: %s", absPath)
	}

	return project, nil
}

// ListProjects 获取所有项目
func (s *ProjectService) ListProjects(ctx context.Context) ([]*codeanalysis.Project, error) {
	return s.projectStore.List(ctx)
}

// DeleteProject 删除项目
func (s *ProjectService) DeleteProject(ctx context.Context, projectID string) error {
	return s.projectStore.Delete(ctx, projectID)
}
