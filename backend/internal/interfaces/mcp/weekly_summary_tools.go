package mcp

import (
	"context"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetDailySummariesRangeInput 批量查询日报工具输入
type GetDailySummariesRangeInput struct {
	StartDate string `json:"start_date" jsonschema:"Start date in YYYY-MM-DD format"`
	EndDate   string `json:"end_date" jsonschema:"End date in YYYY-MM-DD format"`
}

// GetDailySummariesRangeOutput 批量查询日报工具输出
type GetDailySummariesRangeOutput struct {
	Summaries []*domainCursor.DailySummary `json:"summaries" jsonschema:"Array of daily summaries"`
	Count     int                          `json:"count" jsonschema:"Number of summaries found"`
}

// getDailySummariesRangeTool 批量查询日报工具
func (s *MCPServer) getDailySummariesRangeTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetDailySummariesRangeInput,
) (*mcp.CallToolResult, GetDailySummariesRangeOutput, error) {
	summaries, err := s.dailySummaryService.GetDailySummariesRange(input.StartDate, input.EndDate)
	if err != nil {
		return nil, GetDailySummariesRangeOutput{}, err
	}

	return nil, GetDailySummariesRangeOutput{
		Summaries: summaries,
		Count:     len(summaries),
	}, nil
}

// SaveWeeklySummaryInput 保存周报工具输入
type SaveWeeklySummaryInput struct {
	WeekStart          string                               `json:"week_start" jsonschema:"Week start date in YYYY-MM-DD format (Monday)"`
	WeekEnd            string                               `json:"week_end" jsonschema:"Week end date in YYYY-MM-DD format (Sunday)"`
	Summary            string                               `json:"summary" jsonschema:"Summary content in Markdown format"`
	Language           string                               `json:"language,omitempty" jsonschema:"Language code: zh or en"`
	Projects           []*domainCursor.WeeklyProjectSummary `json:"projects,omitempty" jsonschema:"Array of project summary objects"`
	Categories         *domainCursor.WorkCategories         `json:"categories,omitempty" jsonschema:"Work category statistics"`
	TotalSessions      int                                  `json:"total_sessions,omitempty" jsonschema:"Total session count"`
	WorkingDays        int                                  `json:"working_days,omitempty" jsonschema:"Number of working days with data"`
	CodeChanges        *domainCursor.CodeChangeSummary      `json:"code_changes,omitempty" jsonschema:"Code changes summary"`
	KeyAccomplishments []string                             `json:"key_accomplishments,omitempty" jsonschema:"List of key accomplishments"`
}

// SaveWeeklySummaryOutput 保存周报工具输出
type SaveWeeklySummaryOutput struct {
	Success   bool   `json:"success" jsonschema:"Whether the operation succeeded"`
	SummaryID string `json:"summary_id,omitempty" jsonschema:"Summary ID"`
	Message   string `json:"message" jsonschema:"Message"`
}

// saveWeeklySummaryTool 保存周报工具
func (s *MCPServer) saveWeeklySummaryTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SaveWeeklySummaryInput,
) (*mcp.CallToolResult, SaveWeeklySummaryOutput, error) {
	// 转换为 Service 层输入
	serviceInput := &appCursor.SaveWeeklySummaryInput{
		WeekStart:          input.WeekStart,
		WeekEnd:            input.WeekEnd,
		Summary:            input.Summary,
		Language:           input.Language,
		Projects:           input.Projects,
		Categories:         input.Categories,
		TotalSessions:      input.TotalSessions,
		WorkingDays:        input.WorkingDays,
		CodeChanges:        input.CodeChanges,
		KeyAccomplishments: input.KeyAccomplishments,
	}

	result, err := s.weeklySummaryService.SaveWeeklySummary(serviceInput)
	if err != nil {
		return nil, SaveWeeklySummaryOutput{}, err
	}

	return nil, SaveWeeklySummaryOutput{
		Success:   result.Success,
		SummaryID: result.SummaryID,
		Message:   result.Message,
	}, nil
}

// GetWeeklySummaryInput 查询周报工具输入
type GetWeeklySummaryInput struct {
	WeekStart string `json:"week_start" jsonschema:"Week start date in YYYY-MM-DD format (Monday)"`
}

// GetWeeklySummaryOutput 查询周报工具输出
type GetWeeklySummaryOutput struct {
	Summary     *domainCursor.WeeklySummary `json:"summary,omitempty" jsonschema:"Weekly summary object"`
	Found       bool                        `json:"found" jsonschema:"Whether the summary was found"`
	NeedsUpdate bool                        `json:"needs_update" jsonschema:"Whether source data has changed and summary needs regeneration"`
}

// getWeeklySummaryTool 查询周报工具（带幂等检查）
func (s *MCPServer) getWeeklySummaryTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetWeeklySummaryInput,
) (*mcp.CallToolResult, GetWeeklySummaryOutput, error) {
	result, err := s.weeklySummaryService.GetWeeklySummary(input.WeekStart)
	if err != nil {
		return nil, GetWeeklySummaryOutput{}, err
	}

	return nil, GetWeeklySummaryOutput{
		Summary:     result.Summary,
		Found:       result.Found,
		NeedsUpdate: result.NeedsUpdate,
	}, nil
}
