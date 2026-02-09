package codeanalysis

import (
	"context"
	"log/slog"

	"github.com/cocursor/backend/internal/domain/codeanalysis"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// ImpactAnalyzer 影响面分析器实现
type ImpactAnalyzer struct {
	logger     *slog.Logger
	repository *CallGraphRepository
}

// NewImpactAnalyzer 创建影响面分析器
func NewImpactAnalyzer(repository *CallGraphRepository) *ImpactAnalyzer {
	return &ImpactAnalyzer{
		logger:     log.NewModuleLogger("codeanalysis", "impact_analyzer"),
		repository: repository,
	}
}

// AnalyzeImpact 分析函数变更的影响面
func (a *ImpactAnalyzer) AnalyzeImpact(ctx context.Context, dbPath string, functions []string, maxDepth int) (*codeanalysis.ImpactAnalysisResult, error) {
	a.logger.Info("analyzing impact",
		"db_path", dbPath,
		"functions", len(functions),
		"max_depth", maxDepth,
	)

	if maxDepth <= 0 {
		maxDepth = 3 // 默认深度
	}

	result := &codeanalysis.ImpactAnalysisResult{
		Impacts: make([]codeanalysis.FunctionImpact, 0),
		Summary: codeanalysis.ImpactSummary{
			FunctionsAnalyzed: len(functions),
			AffectedFiles:     make([]string, 0),
		},
	}

	// 获取 commit 信息
	commit, _ := a.repository.GetMetadata(ctx, dbPath, "commit")
	result.AnalysisCommit = commit

	// 获取所有目标函数的节点（同时按 full_name 和 canonical_name 匹配）
	nodes, err := a.repository.GetFuncNodesByFullNames(ctx, dbPath, functions)
	if err != nil {
		a.logger.Warn("failed to get function nodes", "error", err)
	}

	if len(nodes) == 0 {
		a.logger.Warn("no matching functions found in call graph",
			"functions", functions,
		)
		return result, nil
	}

	a.logger.Info("matched function nodes",
		"requested", len(functions),
		"matched", len(nodes),
	)

	// 收集所有函数 ID
	funcIDs := make([]int64, len(nodes))
	funcMap := make(map[int64]*codeanalysis.FuncNode)
	for i, node := range nodes {
		funcIDs[i] = node.ID
		funcMap[node.ID] = node
	}

	// 查询所有调用者
	callers, err := a.repository.GetCallersWithDepth(ctx, dbPath, funcIDs, maxDepth)
	if err != nil {
		return nil, err
	}

	// 按目标函数分组组织结果
	affectedFilesSet := make(map[string]bool)
	totalAffected := make(map[string]bool)

	for _, node := range nodes {
		impact := codeanalysis.FunctionImpact{
			Function:    node.FullName,
			DisplayName: node.FuncName,
			File:        node.FilePath,
			Callers:     make([]codeanalysis.CallerInfo, 0),
		}

		// 找到这个函数的所有调用者
		maxDepthReached := 0
		for _, caller := range callers {
			// 通过递归查询已经包含了所有上游调用者
			// 这里简单地将所有调用者都归到第一个匹配的函数
			impact.Callers = append(impact.Callers, caller)
			if caller.Depth > maxDepthReached {
				maxDepthReached = caller.Depth
			}

			// 记录受影响的文件
			if caller.File != "" {
				affectedFilesSet[caller.File] = true
			}
			totalAffected[caller.Function] = true
		}

		impact.TotalCallers = len(impact.Callers)
		impact.MaxDepthReached = maxDepthReached

		result.Impacts = append(result.Impacts, impact)
	}

	// 构建摘要
	for file := range affectedFilesSet {
		result.Summary.AffectedFiles = append(result.Summary.AffectedFiles, file)
	}
	result.Summary.TotalAffected = len(totalAffected)

	a.logger.Info("impact analysis completed",
		"functions_analyzed", len(functions),
		"total_affected", result.Summary.TotalAffected,
		"affected_files", len(result.Summary.AffectedFiles),
	)

	return result, nil
}

// 确保实现接口
var _ codeanalysis.ImpactAnalyzer = (*ImpactAnalyzer)(nil)
