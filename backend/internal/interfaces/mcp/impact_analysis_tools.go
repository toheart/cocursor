package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	appAnalysis "github.com/cocursor/backend/internal/application/codeanalysis"
	"github.com/cocursor/backend/internal/domain/codeanalysis"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ============================================================
// search_function 工具
// ============================================================

// SearchFunctionInput 搜索函数输入
type SearchFunctionInput struct {
	ProjectPath string `json:"project_path" jsonschema:"required,Absolute path to the project"`
	FilePath    string `json:"file_path,omitempty" jsonschema:"File path relative to project root"`
	Line        int    `json:"line,omitempty" jsonschema:"Line number, used with file_path for precise location"`
	FullName    string `json:"full_name,omitempty" jsonschema:"Full function name (canonical format)"`
	Package     string `json:"package,omitempty" jsonschema:"Package path"`
	FuncName    string `json:"func_name,omitempty" jsonschema:"Short function name (supports fuzzy matching)"`
	Limit       int    `json:"limit,omitempty" jsonschema:"Max results (default 20)"`
}

// SearchFunctionOutput 搜索函数输出
type SearchFunctionOutput struct {
	Text string `json:"text"`
}

func (s *MCPServer) searchFunctionTool(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input SearchFunctionInput,
) (*mcp.CallToolResult, SearchFunctionOutput, error) {
	// 获取项目配置以获取调用图路径
	project, err := s.projectService.GetProject(ctx, input.ProjectPath)
	if err != nil {
		return nil, SearchFunctionOutput{Text: fmt.Sprintf("❌ 项目未注册或无法识别: %v\n请先生成调用图。", err)}, nil
	}

	// 获取最新调用图
	latest, err := s.callGraphManager.GetLatest(ctx, project.ID)
	if err != nil {
		return nil, SearchFunctionOutput{Text: fmt.Sprintf("❌ 未找到调用图: %v\n请先生成调用图。", err)}, nil
	}

	// 搜索函数
	nodes, err := s.callGraphRepo.SearchFunctions(
		ctx, latest.DBPath,
		input.FilePath, input.Line,
		input.FullName, input.Package, input.FuncName,
		input.Limit,
	)
	if err != nil {
		return nil, SearchFunctionOutput{Text: fmt.Sprintf("❌ 搜索失败: %v", err)}, nil
	}

	if len(nodes) == 0 {
		return nil, SearchFunctionOutput{Text: "未找到匹配的函数。"}, nil
	}

	// 格式化输出
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 函数搜索结果（共 %d 个）\n\n", len(nodes)))
	sb.WriteString(fmt.Sprintf("调用图版本: commit %s\n\n", latest.Commit))

	for i, node := range nodes {
		sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, node.FuncName))
		sb.WriteString(fmt.Sprintf("- 规范名称: `%s`\n", node.CanonicalName))
		if node.FullName != node.CanonicalName {
			sb.WriteString(fmt.Sprintf("- SSA 全名: `%s`\n", node.FullName))
		}
		sb.WriteString(fmt.Sprintf("- 包: `%s`\n", node.Package))
		if node.FilePath != "" {
			sb.WriteString(fmt.Sprintf("- 文件: %s:%d-%d\n", node.FilePath, node.LineStart, node.LineEnd))
		}
		exported := "否"
		if node.IsExported {
			exported = "是"
		}
		sb.WriteString(fmt.Sprintf("- 导出: %s\n\n", exported))
	}

	return nil, SearchFunctionOutput{Text: sb.String()}, nil
}

// ============================================================
// query_impact 工具
// ============================================================

// QueryImpactInput 查询影响面输入
type QueryImpactInput struct {
	ProjectPath string   `json:"project_path" jsonschema:"required,Absolute path to the project"`
	Functions   []string `json:"functions" jsonschema:"required,Function names to analyze"`
	Depth       int      `json:"depth,omitempty" jsonschema:"Max call chain depth (default 3, max 10)"`
	Commit      string   `json:"commit,omitempty" jsonschema:"Specific call graph commit version"`
}

// QueryImpactOutput 查询影响面输出
type QueryImpactOutput struct {
	Text string `json:"text"`
}

func (s *MCPServer) queryImpactTool(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input QueryImpactInput,
) (*mcp.CallToolResult, QueryImpactOutput, error) {
	if len(input.Functions) == 0 {
		return nil, QueryImpactOutput{Text: "❌ 请提供至少一个函数名。"}, nil
	}

	depth := input.Depth
	if depth <= 0 {
		depth = 3
	}
	if depth > 10 {
		depth = 10
	}

	// 调用影响面查询
	result, err := s.impactService.QueryImpact(ctx, &appAnalysis.QueryImpactRequest{
		ProjectPath: input.ProjectPath,
		Functions:   input.Functions,
		Depth:       depth,
		Commit:      input.Commit,
	})
	if err != nil {
		return nil, QueryImpactOutput{Text: fmt.Sprintf("❌ 影响面查询失败: %v", err)}, nil
	}

	// 获取时效性信息
	freshness := s.getFreshnessInfo(ctx, input.ProjectPath, result.AnalysisCommit)

	// 格式化输出
	text := formatImpactResult(result, freshness, depth)
	return nil, QueryImpactOutput{Text: text}, nil
}

// ============================================================
// analyze_diff_impact 工具
// ============================================================

// AnalyzeDiffImpactInput 分析 diff 影响面输入
type AnalyzeDiffImpactInput struct {
	ProjectPath string `json:"project_path" jsonschema:"required,Absolute path to the project"`
	CommitRange string `json:"commit_range,omitempty" jsonschema:"Git commit range (default: HEAD~1..HEAD)"`
	Depth       int    `json:"depth,omitempty" jsonschema:"Max call chain depth (default 3)"`
}

// AnalyzeDiffImpactOutput 分析 diff 影响面输出
type AnalyzeDiffImpactOutput struct {
	Text string `json:"text"`
}

func (s *MCPServer) analyzeDiffImpactTool(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input AnalyzeDiffImpactInput,
) (*mcp.CallToolResult, AnalyzeDiffImpactOutput, error) {
	depth := input.Depth
	if depth <= 0 {
		depth = 3
	}
	if depth > 10 {
		depth = 10
	}

	// 调用完整分析
	resp, err := s.impactService.FullAnalysis(ctx, &appAnalysis.FullAnalysisRequest{
		ProjectPath: input.ProjectPath,
		CommitRange: input.CommitRange,
		Depth:       depth,
	})
	if err != nil {
		return nil, AnalyzeDiffImpactOutput{Text: fmt.Sprintf("❌ 影响面分析失败: %v", err)}, nil
	}

	// 获取时效性信息
	analysisCommit := ""
	if resp.ImpactResult != nil {
		analysisCommit = resp.ImpactResult.AnalysisCommit
	}
	freshness := s.getFreshnessInfo(ctx, input.ProjectPath, analysisCommit)

	// 格式化输出
	text := formatDiffImpactResult(resp, freshness, depth)
	return nil, AnalyzeDiffImpactOutput{Text: text}, nil
}

// ============================================================
// 辅助函数
// ============================================================

// freshnessInfo 数据时效性信息
type freshnessInfo struct {
	commit    string
	createdAt string
	level     string // "fresh", "stale", "outdated"
	message   string
}

// getFreshnessInfo 获取调用图的时效性信息
func (s *MCPServer) getFreshnessInfo(ctx context.Context, projectPath string, graphCommit string) freshnessInfo {
	info := freshnessInfo{
		commit: graphCommit,
		level:  "fresh",
	}

	if graphCommit == "" {
		info.level = "outdated"
		info.message = "❌ 未找到调用图版本信息"
		return info
	}

	// 获取项目并查调用图创建时间
	project, err := s.projectService.GetProject(ctx, projectPath)
	if err != nil {
		return info
	}

	latest, err := s.callGraphManager.GetLatest(ctx, project.ID)
	if err != nil {
		return info
	}

	info.createdAt = latest.CreatedAt.Format("2006-01-02 15:04:05")

	// 检查当前 HEAD
	diffAnalyzer := s.impactService.GetDiffAnalyzer()
	if diffAnalyzer == nil {
		return info
	}

	currentHead, err := diffAnalyzer.GetCurrentCommit(ctx, projectPath)
	if err != nil {
		return info
	}

	if currentHead != "" && graphCommit != "" {
		// 比较 commit（短 hash 前缀匹配）
		if strings.HasPrefix(currentHead, graphCommit) || strings.HasPrefix(graphCommit, currentHead) {
			info.level = "fresh"
			info.message = "✅ 调用图与当前 HEAD 一致"
		} else {
			// 计算落后多少 commit
			behind, err := diffAnalyzer.GetCommitsBetween(ctx, projectPath, graphCommit, currentHead)
			if err == nil && behind <= 5 {
				info.level = "stale"
				info.message = fmt.Sprintf("⚠️ 调用图落后 %d 个 commit，影响面结果可能不完整，建议重新生成调用图", behind)
			} else {
				// 检查时间
				if time.Since(latest.CreatedAt) > 7*24*time.Hour {
					info.level = "outdated"
					info.message = "❌ 调用图已过时（超过 7 天），请先重新生成"
				} else {
					info.level = "stale"
					info.message = "⚠️ HEAD 已变更，影响面结果可能不完整，建议重新生成调用图"
				}
			}
		}
	}

	return info
}

// formatImpactResult 格式化影响面分析结果
func formatImpactResult(result *codeanalysis.ImpactAnalysisResult, freshness freshnessInfo, maxDepth int) string {
	var sb strings.Builder

	sb.WriteString("## 影响面分析结果\n\n")

	// 数据版本信息
	if freshness.commit != "" {
		sb.WriteString(fmt.Sprintf("数据版本: commit %s", freshness.commit))
		if freshness.createdAt != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", freshness.createdAt))
		}
		sb.WriteString("\n")
	}
	if freshness.message != "" {
		sb.WriteString(freshness.message + "\n")
	}
	sb.WriteString("\n")

	if len(result.Impacts) == 0 {
		sb.WriteString("未找到匹配的函数或无调用关系。\n")
		return sb.String()
	}

	for _, impact := range result.Impacts {
		sb.WriteString(fmt.Sprintf("### 变更函数: %s\n", impact.DisplayName))
		if impact.File != "" {
			sb.WriteString(fmt.Sprintf("文件: %s\n\n", impact.File))
		}

		if len(impact.Callers) == 0 {
			sb.WriteString("无上游调用者（可能是入口函数或未被调用的函数）\n\n")
			continue
		}

		sb.WriteString(fmt.Sprintf("#### 上游调用链（最大深度: %d）\n", maxDepth))
		// 按深度分组展示
		for depth := 1; depth <= impact.MaxDepthReached; depth++ {
			for _, caller := range impact.Callers {
				if caller.Depth != depth {
					continue
				}
				indent := strings.Repeat("  ", depth-1)
				location := ""
				if caller.File != "" {
					location = fmt.Sprintf(" → %s:%d", caller.File, caller.Line)
				}
				sb.WriteString(fmt.Sprintf("%s├── [深度%d] %s%s\n", indent, depth, caller.DisplayName, location))
			}
		}
		sb.WriteString("\n")

		// 汇总
		sb.WriteString("#### 汇总\n")
		sb.WriteString(fmt.Sprintf("- 调用者总数: %d 个\n", impact.TotalCallers))
		sb.WriteString(fmt.Sprintf("- 最大深度: %d\n\n", impact.MaxDepthReached))
	}

	// 全局汇总
	sb.WriteString("### 全局汇总\n")
	sb.WriteString(fmt.Sprintf("- 分析函数数: %d\n", result.Summary.FunctionsAnalyzed))
	sb.WriteString(fmt.Sprintf("- 受影响函数: %d 个\n", result.Summary.TotalAffected))
	sb.WriteString(fmt.Sprintf("- 受影响文件: %d 个\n", len(result.Summary.AffectedFiles)))

	return sb.String()
}

// formatDiffImpactResult 格式化 diff + 影响面分析结果
func formatDiffImpactResult(resp *appAnalysis.FullAnalysisResponse, freshness freshnessInfo, maxDepth int) string {
	var sb strings.Builder

	sb.WriteString("## Diff 影响面分析报告\n\n")

	// 数据版本信息
	if freshness.commit != "" {
		sb.WriteString(fmt.Sprintf("调用图版本: commit %s", freshness.commit))
		if freshness.createdAt != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", freshness.createdAt))
		}
		sb.WriteString("\n")
	}
	if freshness.message != "" {
		sb.WriteString(freshness.message + "\n")
	}
	sb.WriteString("\n")

	// Diff 结果
	if resp.DiffResult != nil {
		sb.WriteString(fmt.Sprintf("### 变更概览（%s）\n", resp.DiffResult.CommitRange))
		sb.WriteString(fmt.Sprintf("- 变更文件: %d 个\n", len(resp.DiffResult.ChangedFiles)))
		sb.WriteString(fmt.Sprintf("- 变更函数: %d 个\n\n", len(resp.DiffResult.ChangedFunctions)))

		if len(resp.DiffResult.ChangedFunctions) > 0 {
			sb.WriteString("#### 变更函数列表\n")
			for _, fn := range resp.DiffResult.ChangedFunctions {
				changeIcon := "📝"
				switch fn.ChangeType {
				case "added":
					changeIcon = "➕"
				case "deleted":
					changeIcon = "➖"
				}
				sb.WriteString(fmt.Sprintf("- %s `%s` — %s:%d-%d (+%d/-%d)\n",
					changeIcon, fn.Name, fn.File, fn.LineStart, fn.LineEnd, fn.LinesAdded, fn.LinesRemoved))
			}
			sb.WriteString("\n")
		}
	}

	// 影响面结果
	if resp.ImpactResult != nil && len(resp.ImpactResult.Impacts) > 0 {
		sb.WriteString("### 影响面分析\n\n")

		for _, impact := range resp.ImpactResult.Impacts {
			sb.WriteString(fmt.Sprintf("#### %s\n", impact.DisplayName))
			if impact.File != "" {
				sb.WriteString(fmt.Sprintf("文件: %s\n\n", impact.File))
			}

			if len(impact.Callers) == 0 {
				sb.WriteString("无上游调用者\n\n")
				continue
			}

			sb.WriteString(fmt.Sprintf("上游调用链（最大深度: %d）:\n", maxDepth))
			for depth := 1; depth <= impact.MaxDepthReached; depth++ {
				for _, caller := range impact.Callers {
					if caller.Depth != depth {
						continue
					}
					indent := strings.Repeat("  ", depth-1)
					location := ""
					if caller.File != "" {
						location = fmt.Sprintf(" → %s:%d", caller.File, caller.Line)
					}
					sb.WriteString(fmt.Sprintf("%s├── [深度%d] %s%s\n", indent, depth, caller.DisplayName, location))
				}
			}
			sb.WriteString("\n")
		}

		// 全局汇总
		sb.WriteString("### 全局汇总\n")
		sb.WriteString(fmt.Sprintf("- 变更函数: %d 个\n", resp.ImpactResult.Summary.FunctionsAnalyzed))
		sb.WriteString(fmt.Sprintf("- 受影响函数: %d 个\n", resp.ImpactResult.Summary.TotalAffected))
		sb.WriteString(fmt.Sprintf("- 受影响文件: %d 个\n", len(resp.ImpactResult.Summary.AffectedFiles)))
	} else if resp.DiffResult != nil && len(resp.DiffResult.ChangedFunctions) == 0 {
		sb.WriteString("### 影响面分析\n\n无 Go 函数变更，跳过影响面分析。\n")
	} else {
		sb.WriteString("### 影响面分析\n\n未找到变更函数在调用图中的匹配。可能原因：\n")
		sb.WriteString("1. 调用图未包含这些函数（检查入口函数配置）\n")
		sb.WriteString("2. 调用图版本过旧\n")
	}

	return sb.String()
}
