package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/cocursor/backend/internal/infrastructure/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DaemonStatusInput 守护进程状态工具输入（空输入）
type DaemonStatusInput struct{}

// DaemonStatusOutput 守护进程状态工具输出
type DaemonStatusOutput struct {
	Status      string      `json:"status" jsonschema:"运行状态"`
	Version     string      `json:"version" jsonschema:"版本号"`
	DBPath      string      `json:"db_path" jsonschema:"数据库路径"`
	DailyStats  *DailyStats `json:"daily_stats,omitempty" jsonschema:"今日统计数据"`
	CachedEmail string      `json:"cached_email,omitempty" jsonschema:"缓存的邮箱地址"`
}

// DailyStats 每日统计数据
type DailyStats struct {
	Date                   string `json:"date" jsonschema:"日期"`
	TabSuggestedLines      int    `json:"tab_suggested_lines" jsonschema:"Tab 建议的代码行数"`
	TabAcceptedLines       int    `json:"tab_accepted_lines" jsonschema:"Tab 接受的代码行数"`
	ComposerSuggestedLines int    `json:"composer_suggested_lines" jsonschema:"Composer 建议的代码行数"`
	ComposerAcceptedLines  int    `json:"composer_accepted_lines" jsonschema:"Composer 接受的代码行数"`
}

// DailyReportContextInput 日报上下文工具输入
type DailyReportContextInput struct {
	ProjectPath string `json:"project_path" jsonschema:"项目路径，如 D:/code/cocursor"`
}

// DailyReportContextOutput 日报上下文工具输出
type DailyReportContextOutput struct {
	Date        string   `json:"date" jsonschema:"日期"`
	TotalChats  int      `json:"total_chats" jsonschema:"总对话数"`
	ActiveUsers []string `json:"active_users" jsonschema:"活跃用户列表"`
	Summary     string   `json:"summary" jsonschema:"摘要"`
}

// SessionHealthInput 会话健康工具输入
type SessionHealthInput struct {
	ProjectPath string `json:"project_path,omitempty" jsonschema:"项目路径，如 D:/code/cocursor（可选，如果不提供则尝试自动检测）"`
}

// SessionHealthOutput 会话健康工具输出
type SessionHealthOutput struct {
	Entropy float64 `json:"entropy" jsonschema:"会话熵值"`
	Status  string  `json:"status" jsonschema:"健康状态：healthy/sub_healthy/dangerous"`
	Warning string  `json:"warning,omitempty" jsonschema:"警告信息（如果有）"`
	Message string  `json:"message" jsonschema:"建议消息"`
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
	statsService := appCursor.NewStatsService()
	entropy := statsService.CalculateSessionEntropy(*activeComposer)

	// 获取健康状态
	status, warning := statsService.GetHealthStatus(entropy)

	// 构建消息
	message := fmt.Sprintf("当前会话熵值为 %.2f", entropy)
	if status == appCursor.HealthStatusDangerous {
		message += "，建议执行 /openspec-archive"
	} else if status == appCursor.HealthStatusSubHealthy {
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

// getDaemonStatusTool 获取守护进程状态工具
func getDaemonStatusTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DaemonStatusInput,
) (*mcp.CallToolResult, DaemonStatusOutput, error) {
	// 创建路径解析器和数据库读取器
	pathResolver := infraCursor.NewPathResolver()
	dbReader := infraCursor.NewDBReader()

	// 获取全局存储数据库路径
	dbPath, err := pathResolver.GetGlobalStoragePath()
	if err != nil {
		// 如果无法获取路径，返回错误信息但不失败
		dbPath = fmt.Sprintf("error: %v", err)
	}

	output := DaemonStatusOutput{
		Status:  "running",
		Version: "v0.1.0",
		DBPath:  dbPath,
	}

	// 读取今日统计数据
	if dbPath != "" && err == nil {
		// 获取今天的日期
		today := time.Now().Format("2006-01-02")
		key := fmt.Sprintf("aiCodeTracking.dailyStats.v1.5.%s", today)

		// 读取统计数据
		value, err := dbReader.ReadValueFromTable(dbPath, key)
		if err == nil && len(value) > 0 {
			var stats DailyStats
			if err := json.Unmarshal(value, &stats); err == nil {
				output.DailyStats = &stats
			}
		}

		// 读取缓存的邮箱
		emailValue, err := dbReader.ReadValueFromTable(dbPath, "cursorAuth/cachedEmail")
		if err == nil && len(emailValue) > 0 {
			output.CachedEmail = string(emailValue)
		}
	}

	// 返回 nil result，SDK 会自动处理输出
	return nil, output, nil
}

// generateDailyReportContextTool 生成日报上下文工具
func generateDailyReportContextTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DailyReportContextInput,
) (*mcp.CallToolResult, DailyReportContextOutput, error) {
	// 创建路径解析器和数据库读取器
	pathResolver := cursor.NewPathResolver()
	dbReader := cursor.NewDBReader()

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

// getCursorDBPath 获取 Cursor 数据库路径
func getCursorDBPath() string {
	// Windows: %APPDATA%/Cursor/User/globalStorage/state.vscdb
	// macOS: ~/Library/Application Support/Cursor/User/globalStorage/state.vscdb
	// Linux: ~/.config/Cursor/User/globalStorage/state.vscdb

	var basePath string
	switch {
	case os.Getenv("APPDATA") != "":
		// Windows
		basePath = filepath.Join(os.Getenv("APPDATA"), "Cursor", "User", "globalStorage", "state.vscdb")
	case os.Getenv("HOME") != "":
		// macOS/Linux
		home := os.Getenv("HOME")
		if _, err := os.Stat(filepath.Join(home, "Library")); err == nil {
			// macOS
			basePath = filepath.Join(home, "Library", "Application Support", "Cursor", "User", "globalStorage", "state.vscdb")
		} else {
			// Linux
			basePath = filepath.Join(home, ".config", "Cursor", "User", "globalStorage", "state.vscdb")
		}
	default:
		return "unknown"
	}

	// 检查文件是否存在
	if _, err := os.Stat(basePath); err == nil {
		return basePath
	}

	return basePath
}
