package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DailyReportContextInput 日报上下文工具输入
type DailyReportContextInput struct {
	ProjectPath string `json:"project_path" jsonschema:"Project path, e.g., D:/code/cocursor"`
}

// DailyReportContextOutput 日报上下文工具输出
type DailyReportContextOutput struct {
	Date        string   `json:"date" jsonschema:"Date"`
	TotalChats  int      `json:"total_chats" jsonschema:"Total number of chats"`
	ActiveUsers []string `json:"active_users" jsonschema:"Active users list"`
	Summary     string   `json:"summary" jsonschema:"Summary"`
}

// SessionHealthInput 会话健康工具输入
type SessionHealthInput struct {
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Project path, e.g., D:/code/cocursor (optional, will attempt auto-detection if not provided)"`
}

// SessionHealthOutput 会话健康工具输出
type SessionHealthOutput struct {
	Entropy float64 `json:"entropy" jsonschema:"Session entropy value"`
	Status  string  `json:"status" jsonschema:"Health status: healthy/sub_healthy/dangerous"`
	Warning string  `json:"warning,omitempty" jsonschema:"Warning message (if any)"`
	Message string  `json:"message" jsonschema:"Suggestion message"`
}

// getSessionHealthTool 获取当前活跃会话的健康状态工具
func getSessionHealthTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SessionHealthInput,
) (*mcp.CallToolResult, SessionHealthOutput, error) {
	// 创建路径解析器和数据库读取器
	pathResolver := infraCursor.NewPathResolver()
	dbReader := infraCursor.NewDBReader()

	// 如果没有提供项目路径，尝试自动检测（使用当前工作目录）
	projectPath := input.ProjectPath
	if projectPath == "" {
		// 尝试从当前工作目录获取
		cwd, err := os.Getwd()
		if err != nil {
			return nil, SessionHealthOutput{}, fmt.Errorf("无法获取当前工作目录，请提供 project_path 参数: %w", err)
		}
		projectPath = cwd
	}

	// 根据项目路径查找工作区 ID
	workspaceID, err := pathResolver.GetWorkspaceIDByPath(projectPath)
	if err != nil {
		return nil, SessionHealthOutput{}, fmt.Errorf("无法找到工作区: %w", err)
	}

	// 获取工作区数据库路径
	workspaceDBPath, err := pathResolver.GetWorkspaceDBPath(workspaceID)
	if err != nil {
		return nil, SessionHealthOutput{}, fmt.Errorf("无法找到工作区数据库: %w", err)
	}

	// 读取 composer.composerData
	composerDataValue, err := dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
	if err != nil {
		return nil, SessionHealthOutput{}, fmt.Errorf("无法读取 composer 数据: %w", err)
	}

	// 解析 Composer 数据
	composers, err := domainCursor.ParseComposerData(string(composerDataValue))
	if err != nil {
		return nil, SessionHealthOutput{}, fmt.Errorf("无法解析 composer 数据: %w", err)
	}

	// 获取活跃的会话
	activeComposer := domainCursor.GetActiveComposer(composers)
	if activeComposer == nil {
		// 如果没有活跃会话，返回健康状态
		output := SessionHealthOutput{
			Entropy: 0,
			Status:  "healthy",
			Message: "当前没有活跃的会话",
		}
		return nil, output, nil
	}

	// 计算熵值
	// 注意：这里需要 GlobalDBReader，但 CalculateSessionEntropy 不需要它
	// 使用 mock GlobalDBReader（实际不会用到）
	mockGlobalDBReader := infraCursor.NewMockGlobalDBReader()
	statsService := appCursor.NewStatsService(mockGlobalDBReader)
	entropy := statsService.CalculateSessionEntropy(*activeComposer)

	// 获取健康状态
	status, warning := statsService.GetHealthStatus(entropy)

	// 构建消息
	message := fmt.Sprintf("当前会话熵值为 %.2f", entropy)
	switch status {
	case appCursor.HealthStatusDangerous:
		message += "，建议执行 /openspec-archive"
	case appCursor.HealthStatusSubHealthy:
		message += "，建议总结当前会话"
	}

	output := SessionHealthOutput{
		Entropy: entropy,
		Status:  string(status),
		Warning: warning,
		Message: message,
	}

	return nil, output, nil
}

// generateDailyReportContextTool 生成日报上下文工具
func generateDailyReportContextTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DailyReportContextInput,
) (*mcp.CallToolResult, DailyReportContextOutput, error) {
	// 创建路径解析器和数据库读取器
	pathResolver := infraCursor.NewPathResolver()
	dbReader := infraCursor.NewDBReader()

	// 如果没有提供项目路径，返回错误
	if input.ProjectPath == "" {
		return nil, DailyReportContextOutput{}, fmt.Errorf("project_path 参数是必需的")
	}

	// 根据项目路径查找工作区 ID
	workspaceID, err := pathResolver.GetWorkspaceIDByPath(input.ProjectPath)
	if err != nil {
		return nil, DailyReportContextOutput{}, fmt.Errorf("无法找到工作区: %w", err)
	}

	// 获取工作区数据库路径
	workspaceDBPath, err := pathResolver.GetWorkspaceDBPath(workspaceID)
	if err != nil {
		return nil, DailyReportContextOutput{}, fmt.Errorf("无法找到工作区数据库: %w", err)
	}

	// 读取 aiService.prompts
	promptsValue, err := dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "aiService.prompts")
	if err != nil {
		// 如果读取失败，返回空数据而不是错误
		output := DailyReportContextOutput{
			Date:        time.Now().Format("2006-01-02"),
			TotalChats:  0,
			ActiveUsers: []string{},
			Summary:     fmt.Sprintf("无法读取工作区数据: %v", err),
		}
		return nil, output, nil
	}

	// 解析 prompts 数据
	var prompts []map[string]interface{}
	if err := json.Unmarshal(promptsValue, &prompts); err != nil {
		// 如果解析失败，返回基本信息
		output := DailyReportContextOutput{
			Date:        time.Now().Format("2006-01-02"),
			TotalChats:  0,
			ActiveUsers: []string{},
			Summary:     fmt.Sprintf("无法解析 prompts 数据: %v", err),
		}
		return nil, output, nil
	}

	// 统计对话数（prompts 数组长度）
	totalChats := len(prompts)

	// 构建响应
	output := DailyReportContextOutput{
		Date:        time.Now().Format("2006-01-02"),
		TotalChats:  totalChats,
		ActiveUsers: []string{
			// TODO: 从其他数据源获取活跃用户列表
		},
		Summary: fmt.Sprintf("工作区 %s 共有 %d 条 AI 对话记录", input.ProjectPath, totalChats),
	}

	// 返回 nil result，SDK 会自动处理输出
	return nil, output, nil
}
