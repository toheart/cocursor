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

// ImpactService 影响分析服务
type ImpactService struct {
	logger           *slog.Logger
	projectService   *ProjectService
	callGraphManager *infra.CallGraphManager
	diffAnalyzer     *infra.DiffAnalyzer
	impactAnalyzer   *infra.ImpactAnalyzer
}

// NewImpactService 创建影响分析服务
func NewImpactService(
	projectService *ProjectService,
	callGraphManager *infra.CallGraphManager,
	diffAnalyzer *infra.DiffAnalyzer,
	impactAnalyzer *infra.ImpactAnalyzer,
) *ImpactService {
	return &ImpactService{
		logger:           log.NewModuleLogger("codeanalysis", "impact_service"),
		projectService:   projectService,
		callGraphManager: callGraphManager,
		diffAnalyzer:     diffAnalyzer,
		impactAnalyzer:   impactAnalyzer,
	}
}

// AnalyzeDiffRequest 分析 diff 请求
type AnalyzeDiffRequest struct {
	ProjectPath string `json:"project_path"`
	CommitRange string `json:"commit_range"`
}

// AnalyzeDiff 分析 Git diff
func (s *ImpactService) AnalyzeDiff(ctx context.Context, req *AnalyzeDiffRequest) (*codeanalysis.DiffAnalysisResult, error) {
	absPath, err := filepath.Abs(req.ProjectPath)
	if err != nil {
		return nil, err
	}

	commitRange := req.CommitRange
	if commitRange == "" {
		commitRange = "HEAD~1..HEAD"
	}

	s.logger.Info("analyzing diff",
		"project", absPath,
		"commit_range", commitRange,
	)

	result, err := s.diffAnalyzer.AnalyzeDiff(ctx, absPath, commitRange)
	if err != nil {
		return nil, fmt.Errorf("diff analysis failed: %w", err)
	}

	return result, nil
}

// QueryImpactRequest 查询影响面请求
type QueryImpactRequest struct {
	ProjectPath string   `json:"project_path"`
	Functions   []string `json:"functions"`
	Depth       int      `json:"depth"`
	Commit      string   `json:"commit"`
}

// QueryImpact 查询影响面
func (s *ImpactService) QueryImpact(ctx context.Context, req *QueryImpactRequest) (*codeanalysis.ImpactAnalysisResult, error) {
	absPath, err := filepath.Abs(req.ProjectPath)
	if err != nil {
		return nil, err
	}

	// 获取项目配置
	project, err := s.projectService.GetProject(ctx, absPath)
	if err != nil {
		return nil, fmt.Errorf("project not registered: %w", err)
	}

	// 获取调用图
	var dbPath string
	if req.Commit != "" && req.Commit != "HEAD" {
		dbPath = s.callGraphManager.GetCommitDBPath(project.ID, req.Commit)
	} else {
		// 使用最新的调用图
		latest, err := s.callGraphManager.GetLatest(ctx, project.ID)
		if err != nil {
			return nil, fmt.Errorf("no call graph available: %w", err)
		}
		dbPath = latest.DBPath
	}

	depth := req.Depth
	if depth <= 0 {
		depth = 3
	}

	s.logger.Info("querying impact",
		"project", project.Name,
		"functions", len(req.Functions),
		"depth", depth,
	)

	result, err := s.impactAnalyzer.AnalyzeImpact(ctx, dbPath, req.Functions, depth)
	if err != nil {
		return nil, fmt.Errorf("impact analysis failed: %w", err)
	}

	return result, nil
}

// FullAnalysisRequest 完整分析请求
type FullAnalysisRequest struct {
	ProjectPath string `json:"project_path"`
	CommitRange string `json:"commit_range"`
	Depth       int    `json:"depth"`
}

// FullAnalysisResponse 完整分析响应
type FullAnalysisResponse struct {
	DiffResult   *codeanalysis.DiffAnalysisResult   `json:"diff_result"`
	ImpactResult *codeanalysis.ImpactAnalysisResult `json:"impact_result"`
}

// FullAnalysis 完整分析（diff + impact）
func (s *ImpactService) FullAnalysis(ctx context.Context, req *FullAnalysisRequest) (*FullAnalysisResponse, error) {
	// 1. 分析 diff
	diffResult, err := s.AnalyzeDiff(ctx, &AnalyzeDiffRequest{
		ProjectPath: req.ProjectPath,
		CommitRange: req.CommitRange,
	})
	if err != nil {
		return nil, fmt.Errorf("diff analysis failed: %w", err)
	}

	// 2. 如果没有变更函数，直接返回
	if len(diffResult.ChangedFunctions) == 0 {
		return &FullAnalysisResponse{
			DiffResult: diffResult,
			ImpactResult: &codeanalysis.ImpactAnalysisResult{
				Impacts: []codeanalysis.FunctionImpact{},
				Summary: codeanalysis.ImpactSummary{
					FunctionsAnalyzed: 0,
					TotalAffected:     0,
					AffectedFiles:     []string{},
				},
			},
		}, nil
	}

	// 3. 提取函数完整名称
	functions := make([]string, len(diffResult.ChangedFunctions))
	for i, fn := range diffResult.ChangedFunctions {
		functions[i] = fn.FullName
	}

	// 4. 查询影响面
	impactResult, err := s.QueryImpact(ctx, &QueryImpactRequest{
		ProjectPath: req.ProjectPath,
		Functions:   functions,
		Depth:       req.Depth,
	})
	if err != nil {
		return nil, fmt.Errorf("impact analysis failed: %w", err)
	}

	return &FullAnalysisResponse{
		DiffResult:   diffResult,
		ImpactResult: impactResult,
	}, nil
}
