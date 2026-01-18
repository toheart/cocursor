package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
