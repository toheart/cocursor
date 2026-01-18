package cursor

import "time"

// ComposerData Composer 会话数据
type ComposerData struct {
	// 基础信息
	Type          string `json:"type"`          // 类型，如 "head"
	ComposerID    string `json:"composerId"`    // 会话唯一标识符
	Name          string `json:"name"`          // 会话名称
	CreatedAt     int64  `json:"createdAt"`     // 创建时间戳（毫秒）
	LastUpdatedAt int64  `json:"lastUpdatedAt"` // 最后更新时间戳（毫秒）

	// 模式信息
	UnifiedMode string `json:"unifiedMode"` // 统一模式：agent/edit/chat
	ForceMode   string `json:"forceMode"`   // 强制模式

	// 代码变更统计（关键字段）
	ContextUsagePercent float64 `json:"contextUsagePercent"` // 上下文使用百分比
	TotalLinesAdded     int     `json:"totalLinesAdded"`     // 总添加行数
	TotalLinesRemoved   int     `json:"totalLinesRemoved"`   // 总删除行数
	FilesChangedCount   int     `json:"filesChangedCount"`   // 变更的文件数量
	Subtitle            string  `json:"subtitle"`            // 涉及的文件列表（逗号分隔）

	// 状态标志
	HasUnreadMessages         bool `json:"hasUnreadMessages"`
	HasBlockingPendingActions bool `json:"hasBlockingPendingActions"`
	IsArchived                bool `json:"isArchived"`
	IsDraft                   bool `json:"isDraft"`
	IsWorktree                bool `json:"isWorktree"`
	IsSpec                    bool `json:"isSpec"`

	// 层级关系
	NumSubComposers int      `json:"numSubComposers"` // 子会话数量
	ReferencedPlans []string `json:"referencedPlans"` // 引用的计划列表
	CreatedOnBranch string   `json:"createdOnBranch"` // 创建时的 Git 分支
}

// ComposerDataList Composer 数据列表（用于解析 JSON）
type ComposerDataList struct {
	AllComposers []ComposerData `json:"allComposers"`
}

// GenerationData AI 生成记录数据
type GenerationData struct {
	UnixMs          int64  `json:"unixMs"`          // Unix 时间戳（毫秒）
	GenerationUUID  string `json:"generationUUID"`  // 生成的唯一标识符
	Type            string `json:"type"`            // 生成类型：composer/chat/tab
	TextDescription string `json:"textDescription"` // AI 的实际回复内容或操作描述
}

// GetCreatedAtTime 获取创建时间的 time.Time 对象
func (c *ComposerData) GetCreatedAtTime() time.Time {
	return time.Unix(0, c.CreatedAt*int64(time.Millisecond))
}

// GetLastUpdatedAtTime 获取最后更新时间的 time.Time 对象
func (c *ComposerData) GetLastUpdatedAtTime() time.Time {
	return time.Unix(0, c.LastUpdatedAt*int64(time.Millisecond))
}

// GetDurationMinutes 获取会话持续时间（分钟）
func (c *ComposerData) GetDurationMinutes() float64 {
	duration := c.LastUpdatedAt - c.CreatedAt
	return float64(duration) / 1000.0 / 60.0
}

// GetTotalLinesChanged 获取总变更行数
func (c *ComposerData) GetTotalLinesChanged() int {
	return c.TotalLinesAdded + c.TotalLinesRemoved
}

// DailyAcceptanceStats 每日接受率统计
type DailyAcceptanceStats struct {
	Date                   string  `json:"date"`                    // 日期 YYYY-MM-DD
	TabSuggestedLines      int     `json:"tabSuggestedLines"`      // Tab 建议的代码行数
	TabAcceptedLines       int     `json:"tabAcceptedLines"`        // Tab 接受的代码行数
	TabAcceptanceRate      float64  `json:"tab_acceptance_rate"`     // Tab 接受率（百分比，计算字段）
	ComposerSuggestedLines int     `json:"composerSuggestedLines"`  // Composer 建议的代码行数
	ComposerAcceptedLines  int     `json:"composerAcceptedLines"`    // Composer 接受的代码行数
	ComposerAcceptanceRate float64 `json:"composer_acceptance_rate"` // Composer 接受率（百分比，计算字段）
}

// CalculateAcceptanceRate 计算接受率
func (s *DailyAcceptanceStats) CalculateAcceptanceRate() {
	if s.TabSuggestedLines > 0 {
		s.TabAcceptanceRate = float64(s.TabAcceptedLines) / float64(s.TabSuggestedLines) * 100
	}
	if s.ComposerSuggestedLines > 0 {
		s.ComposerAcceptanceRate = float64(s.ComposerAcceptedLines) / float64(s.ComposerSuggestedLines) * 100
	}
}

// ConversationOverview 对话统计概览
type ConversationOverview struct {
	TotalChats       int     `json:"total_chats"`         // 总对话数
	TotalGenerations int     `json:"total_generations"`   // 总生成数
	ActiveSessions   int     `json:"active_sessions"`     // 活跃会话数
	LatestChatTime   string  `json:"latest_chat_time"`    // 最近对话时间（ISO 8601）
	TimeSinceLastGen float64 `json:"time_since_last_gen"` // 距离最近生成的时间（分钟）
}

// FileReference 文件引用
type FileReference struct {
	FileName       string `json:"file_name"`       // 文件名
	ReferenceCount int    `json:"reference_count"` // 引用次数
	FileType       string `json:"file_type"`       // 文件类型（从扩展名推断）
}

// DailyReport 日报数据
type DailyReport struct {
	Date           string                `json:"date"`            // 日期 YYYY-MM-DD
	WorkspaceID    string                `json:"workspace_id"`    // 工作区 ID
	CodeChanges    *CodeChangeSummary    `json:"code_changes"`    // 代码变更汇总
	AIUsage        *AIUsageSummary       `json:"ai_usage"`        // AI 使用统计
	AcceptanceRate *DailyAcceptanceStats `json:"acceptance_rate"` // 接受率统计
	TopSessions    []*SessionSummary     `json:"top_sessions"`    // Top N 会话
	TopFiles       []*FileReference      `json:"top_files"`       // Top N 文件引用
}

// CodeChangeSummary 代码变更汇总
type CodeChangeSummary struct {
	TotalLinesAdded   int `json:"total_lines_added"`   // 总添加行数
	TotalLinesRemoved int `json:"total_lines_removed"` // 总删除行数
	FilesChanged      int `json:"files_changed"`       // 变更文件数
}

// AIUsageSummary AI 使用汇总
type AIUsageSummary struct {
	TotalChats       int `json:"total_chats"`       // 总对话数
	TotalGenerations int `json:"total_generations"` // 总生成数
	ActiveSessions   int `json:"active_sessions"`   // 活跃会话数
}

// SessionSummary 会话摘要
type SessionSummary struct {
	ComposerID   string  `json:"composer_id"`   // 会话 ID
	Name         string  `json:"name"`          // 会话名称
	TotalLines   int     `json:"total_lines"`   // 总变更行数
	FilesChanged int     `json:"files_changed"` // 变更文件数
	Entropy      float64 `json:"entropy"`       // 熵值
	Duration     float64 `json:"duration"`      // 持续时间（分钟）
}
