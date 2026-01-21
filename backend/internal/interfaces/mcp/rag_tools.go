package mcp

import (
	"context"
	"fmt"
	"time"

	appRAG "github.com/cocursor/backend/internal/application/rag"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SearchHistoryInput RAG 历史搜索工具输入
type SearchHistoryInput struct {
	Query       string `json:"query" jsonschema:"Search query - describe what you're looking for in natural language (required)"`
	ProjectPath string `json:"project_path" jsonschema:"Project path to search in (required), e.g., /Users/xxx/code/myproject"`
	Limit       int    `json:"limit,omitempty" jsonschema:"Maximum number of results to return, defaults to 3, max 10"`
}

// SearchHistoryOutput RAG 历史搜索工具输出
type SearchHistoryOutput struct {
	Results     []*HistorySearchResult `json:"results" jsonschema:"List of relevant historical conversations"`
	TotalCount  int                    `json:"total_count" jsonschema:"Total number of results found"`
	ProjectName string                 `json:"project_name" jsonschema:"Project name searched"`
}

// HistorySearchResult 历史搜索结果（精简版，只包含对 AI 有用的信息）
type HistorySearchResult struct {
	Topic      string   `json:"topic" jsonschema:"Main topic of the conversation"`
	Summary    string   `json:"summary" jsonschema:"Brief summary of what was discussed and solved"`
	Relevance  string   `json:"relevance" jsonschema:"Relevance level: high/medium/low"`
	Tags       []string `json:"tags,omitempty" jsonschema:"Related tags (max 5)"`
	TimeAgo    string   `json:"time_ago" jsonschema:"When this conversation happened (e.g. '2 days ago')"`
	FilesCount int      `json:"files_count,omitempty" jsonschema:"Number of files modified"`
}

// searchHistoryTool RAG 历史搜索工具实现
func (s *MCPServer) searchHistoryTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SearchHistoryInput,
) (*mcp.CallToolResult, SearchHistoryOutput, error) {
	output := SearchHistoryOutput{
		Results: []*HistorySearchResult{},
	}

	// 验证输入
	if input.Query == "" {
		return nil, output, fmt.Errorf("query is required")
	}
	if input.ProjectPath == "" {
		return nil, output, fmt.Errorf("project_path is required - please provide the current project path")
	}

	// 根据项目路径获取项目 ID（workspace ID）
	pathResolver := infraCursor.NewPathResolver()
	workspaceID, err := pathResolver.GetWorkspaceIDByPath(input.ProjectPath)
	if err != nil {
		return nil, output, fmt.Errorf("cannot find workspace for path %s: %w", input.ProjectPath, err)
	}

	// 获取项目名称
	projectName := shortenProjectName(input.ProjectPath)
	output.ProjectName = projectName

	// 设置默认值（默认 3 个，最多 10 个，避免上下文过载）
	limit := input.Limit
	if limit <= 0 {
		limit = 3
	}
	if limit > 10 {
		limit = 10
	}

	// 获取 RAG 搜索服务
	if s.ragInitializer == nil {
		return nil, output, fmt.Errorf("RAG service not initialized. Please configure embedding service first.")
	}

	searchService := s.ragInitializer.GetSearchService()
	if searchService == nil {
		return nil, output, fmt.Errorf("RAG search service not available. Please ensure Qdrant is running and embedding is configured.")
	}

	// 执行搜索（使用 workspace ID 作为 project ID 过滤）
	results, err := searchService.SearchChunks(ctx, &appRAG.SearchRequest{
		Query:      input.Query,
		ProjectIDs: []string{workspaceID}, // 只搜索当前项目
		Limit:      limit,
	})
	if err != nil {
		return nil, output, fmt.Errorf("search failed: %w", err)
	}

	// 转换结果（精简数据）
	output.Results = make([]*HistorySearchResult, 0, len(results))
	for _, r := range results {
		result := &HistorySearchResult{
			Topic:     r.MainTopic,
			Summary:   truncateSummary(r.Summary, 200), // 限制摘要长度
			Relevance: scoreToRelevance(r.Score),
			Tags:      limitTags(r.Tags, 5), // 最多 5 个标签
			TimeAgo:   formatTimeAgo(r.Timestamp),
		}

		// 只有当有修改文件时才返回数量
		if len(r.FilesModified) > 0 {
			result.FilesCount = len(r.FilesModified)
		}

		output.Results = append(output.Results, result)
	}
	output.TotalCount = len(output.Results)

	// 返回 nil，SDK 会自动序列化 output
	return nil, output, nil
}

// truncateSummary 截断摘要到指定长度
func truncateSummary(summary string, maxLen int) string {
	if len(summary) <= maxLen {
		return summary
	}
	// 找到最后一个空格，避免截断单词
	truncated := summary[:maxLen]
	for i := len(truncated) - 1; i >= maxLen-20; i-- {
		if truncated[i] == ' ' {
			return truncated[:i] + "..."
		}
	}
	return truncated + "..."
}

// scoreToRelevance 将分数转换为相关性等级
func scoreToRelevance(score float32) string {
	if score >= 0.7 {
		return "high"
	}
	if score >= 0.4 {
		return "medium"
	}
	return "low"
}

// shortenProjectName 缩短项目名称
func shortenProjectName(name string) string {
	// 移除常见前缀如 "Users-xxx-code-"
	if len(name) > 30 {
		// 只保留最后一部分
		for i := len(name) - 1; i >= 0; i-- {
			if name[i] == '-' || name[i] == '/' {
				return name[i+1:]
			}
		}
	}
	return name
}

// limitTags 限制标签数量
func limitTags(tags []string, max int) []string {
	if len(tags) <= max {
		return tags
	}
	return tags[:max]
}

// formatTimeAgo 格式化时间为相对时间
func formatTimeAgo(timestamp int64) string {
	if timestamp == 0 {
		return "unknown"
	}

	t := time.UnixMilli(timestamp)
	duration := time.Since(t)

	switch {
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins <= 1 {
			return "just now"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	case duration < 30*24*time.Hour:
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}
