package cursor

import (
	"fmt"
	"time"
)

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
	Date                   string  `json:"date"`                      // 日期 YYYY-MM-DD
	TabSuggestedLines      int     `json:"tabSuggestedLines"`         // Tab 建议的代码行数
	TabAcceptedLines       int     `json:"tabAcceptedLines"`          // Tab 接受的代码行数
	TabAcceptanceRate      float64 `json:"tab_acceptance_rate"`       // Tab 接受率（百分比，计算字段）
	ComposerSuggestedLines int     `json:"composerSuggestedLines"`    // Composer 建议的代码行数
	ComposerAcceptedLines  int     `json:"composerAcceptedLines"`     // Composer 接受的代码行数
	ComposerAcceptanceRate float64 `json:"composer_acceptance_rate"`  // Composer 接受率（百分比，计算字段）
	DataQuality            string  `json:"data_quality,omitempty"`    // 数据质量标识：normal/warning/invalid
	WarningMessage         string  `json:"warning_message,omitempty"` // 警告信息
}

// CalculateAcceptanceRate 计算接受率，并检测数据质量
func (s *DailyAcceptanceStats) CalculateAcceptanceRate() {
	s.DataQuality = "normal"
	s.WarningMessage = ""

	// 计算 Tab 接受率
	if s.TabSuggestedLines > 0 {
		s.TabAcceptanceRate = float64(s.TabAcceptedLines) / float64(s.TabSuggestedLines) * 100
		if s.TabAcceptanceRate > 100 {
			s.TabAcceptanceRate = 100
			s.DataQuality = "warning"
			if s.WarningMessage != "" {
				s.WarningMessage += "; "
			}
			s.WarningMessage += fmt.Sprintf("Tab 接受率异常：建议 %d 行，接受 %d 行", s.TabSuggestedLines, s.TabAcceptedLines)
		}
	}

	// 计算 Composer 接受率
	if s.ComposerSuggestedLines > 0 {
		s.ComposerAcceptanceRate = float64(s.ComposerAcceptedLines) / float64(s.ComposerSuggestedLines) * 100
		if s.ComposerAcceptanceRate > 100 {
			s.ComposerAcceptanceRate = 100
			s.DataQuality = "warning"
			if s.WarningMessage != "" {
				s.WarningMessage += "; "
			}
			s.WarningMessage += fmt.Sprintf("Composer 接受率异常：建议 %d 行，接受 %d 行", s.ComposerSuggestedLines, s.ComposerAcceptedLines)
		}
	} else if s.ComposerAcceptedLines > 0 && s.ComposerSuggestedLines == 0 {
		// 建议行数为 0 但有接受行数
		// Cursor 可能不记录 Composer 模式的建议行数，这是正常的数据特征
		// 设置接受率为 -1 表示"不适用"（N/A）
		s.ComposerAcceptanceRate = -1
		// 不再标记为 warning，因为这是 Cursor 的正常行为
	}
}

// GetOverallAcceptanceRate 获取整体接受率
// 只使用有有效建议行数的类型计算
// 返回 -1 表示没有有效数据
func (s *DailyAcceptanceStats) GetOverallAcceptanceRate() float64 {
	totalSuggested := s.TabSuggestedLines + s.ComposerSuggestedLines
	totalAccepted := s.TabAcceptedLines + s.ComposerAcceptedLines

	// 如果两者都有有效数据，正常计算
	if totalSuggested > 0 && totalAccepted <= totalSuggested {
		return float64(totalAccepted) / float64(totalSuggested) * 100
	}

	// 只有 Tab 有有效数据
	if s.TabSuggestedLines > 0 && s.ComposerSuggestedLines == 0 {
		return s.TabAcceptanceRate
	}

	// 只有 Composer 有有效数据
	if s.ComposerSuggestedLines > 0 && s.TabSuggestedLines == 0 {
		return s.ComposerAcceptanceRate
	}

	// 都没有有效数据，返回 -1 表示不适用
	return -1
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
	TotalLinesAdded   int      `json:"total_lines_added"`           // 总添加行数
	TotalLinesRemoved int      `json:"total_lines_removed"`         // 总删除行数
	FilesChanged      int      `json:"files_changed"`               // 变更文件数
	TopChangedFiles   []string `json:"top_changed_files,omitempty"` // Top 变更文件列表（最多5个，可选）
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

// ProjectInfo 项目信息（包含多个工作区）
type ProjectInfo struct {
	ProjectName   string           `json:"project_name"`             // 项目名称（唯一）
	ProjectID     string           `json:"project_id"`               // 项目唯一 ID
	Workspaces    []*WorkspaceInfo `json:"workspaces"`               // 包含的所有工作区
	GitRemoteURL  string           `json:"git_remote_url,omitempty"` // Git 远程仓库 URL（如果有）
	GitBranch     string           `json:"git_branch,omitempty"`     // Git 分支（如果有）
	CreatedAt     time.Time        `json:"created_at"`               // 项目首次发现时间
	LastUpdatedAt time.Time        `json:"last_updated_at"`          // 最后更新时间
}

// WorkspaceInfo 单个工作区信息
type WorkspaceInfo struct {
	WorkspaceID  string `json:"workspace_id"`             // Cursor 工作区 ID
	Path         string `json:"path"`                     // 项目路径
	ProjectName  string `json:"project_name"`             // 所属项目名
	GitRemoteURL string `json:"git_remote_url,omitempty"` // Git 远程 URL
	GitBranch    string `json:"git_branch,omitempty"`     // Git 分支
	IsActive     bool   `json:"is_active"`                // 是否为当前活跃的工作区
	IsPrimary    bool   `json:"is_primary"`               // 是否为主工作区（最新的）
}

// ProjectMatchReason 项目匹配原因
type ProjectMatchReason struct {
	Priority   string  `json:"priority"`       // 优先级：P0/P1/P2
	Method     string  `json:"method"`         // 匹配方法：git_remote_url/physical_path/path_similarity
	Confidence float64 `json:"confidence"`     // 置信度（0.0-1.0）
	Note       string  `json:"note,omitempty"` // 备注
}

// TokenUsage Token 使用统计
type TokenUsage struct {
	Date        string      `json:"date"`         // 日期 YYYY-MM-DD
	TotalTokens int         `json:"total_tokens"` // 总 Token 数
	ByType      TokenByType `json:"by_type"`      // 按类型分类
	Trend       string      `json:"trend"`        // 趋势（与昨日对比），如 "+15%" 或 "-5%"
	Method      string      `json:"method"`       // 计算方法："tiktoken" 或 "estimate"
}

// TokenByType 按类型分类的 Token
type TokenByType struct {
	Tab      int `json:"tab"`      // Tab 自动补全
	Composer int `json:"composer"` // Composer 模式
	Chat     int `json:"chat"`     // 普通聊天
}

// WorkAnalysis 工作分析数据
type WorkAnalysis struct {
	Overview          *WorkAnalysisOverview   `json:"overview"`           // 概览指标
	DailyDetails      []*DailyAnalysis        `json:"daily_details"`      // 每日详情
	CodeChangesTrend  []*DailyCodeChanges     `json:"code_changes_trend"` // 代码变更趋势
	TopFiles          []*FileReference        `json:"top_files"`          // Top N 文件引用
	TimeDistribution  []*TimeDistributionItem `json:"time_distribution"`  // 时间分布（用于热力图）
	EfficiencyMetrics *EfficiencyMetrics      `json:"efficiency_metrics"` // 效率指标
}

// DailyAnalysis 每日分析详情
type DailyAnalysis struct {
	Date             string `json:"date"`               // 日期 YYYY-MM-DD
	LinesAdded       int    `json:"lines_added"`        // 添加行数
	LinesRemoved     int    `json:"lines_removed"`      // 删除行数
	FilesChanged     int    `json:"files_changed"`      // 变更文件数
	ActiveSessions   int    `json:"active_sessions"`    // 活跃会话数
	TokenUsage       int    `json:"token_usage"`        // 当日 Token 消耗
	HasDailyReport   bool   `json:"has_daily_report"`   // 是否有日报
	CompletedChanges int    `json:"completed_changes"`  // 当日完成的 OpenSpec 变更数量
}

// WorkAnalysisOverview 工作分析概览
type WorkAnalysisOverview struct {
	TotalLinesAdded   int    `json:"total_lines_added"`   // 总添加行数
	TotalLinesRemoved int    `json:"total_lines_removed"` // 总删除行数
	FilesChanged      int    `json:"files_changed"`       // 变更文件数
	ActiveSessions    int    `json:"active_sessions"`     // 活跃会话数
	TotalPrompts      int    `json:"total_prompts"`       // 总 Prompts 数（用户输入）
	TotalGenerations  int    `json:"total_generations"`   // 总 Generations 数（AI 回复）
	TotalTokens       int    `json:"total_tokens"`        // 周期内总 Token 消耗
	TokenTrend        string `json:"token_trend"`         // Token 趋势（与上周期对比）
}

// DailyCodeChanges 每日代码变更
type DailyCodeChanges struct {
	Date         string `json:"date"`          // 日期 YYYY-MM-DD
	LinesAdded   int    `json:"lines_added"`   // 添加行数
	LinesRemoved int    `json:"lines_removed"` // 删除行数
	FilesChanged int    `json:"files_changed"` // 变更文件数
}

// TimeDistributionItem 时间分布项（用于热力图）
type TimeDistributionItem struct {
	Hour  int `json:"hour"`  // 小时（0-23）
	Day   int `json:"day"`   // 星期几（0=周日, 1=周一, ...）
	Count int `json:"count"` // 该时段的活跃次数
}

// EfficiencyMetrics 效率指标
type EfficiencyMetrics struct {
	AvgSessionEntropy float64             `json:"avg_session_entropy"` // 平均会话熵值
	AvgContextUsage   float64             `json:"avg_context_usage"`   // 平均上下文使用率
	EntropyTrend      []*EntropyTrendItem `json:"entropy_trend"`       // 熵值趋势
}

// EntropyTrendItem 熵值趋势项
type EntropyTrendItem struct {
	Date  string  `json:"date"`  // 日期 YYYY-MM-DD
	Value float64 `json:"value"` // 熵值
}

// SessionDetail 会话详情（包含消息列表）
type SessionDetail struct {
	Session       *ComposerData `json:"session"`        // 会话元数据
	Messages      []*Message    `json:"messages"`       // 消息列表（按时间排序）
	TotalMessages int           `json:"total_messages"` // 总消息数
	HasMore       bool          `json:"has_more"`       // 是否还有更多消息
}

// Message 消息模型
type Message struct {
	Type       MessageType  `json:"type"`                  // 消息类型：user/ai
	Text       string       `json:"text"`                  // 消息文本
	Timestamp  int64        `json:"timestamp"`             // 时间戳（毫秒）
	CodeBlocks []*CodeBlock `json:"code_blocks,omitempty"` // 代码块（如果有）
	Files      []string     `json:"files,omitempty"`       // 引用的文件（如果有）
	Tools      []*ToolCall  `json:"tools,omitempty"`       // 工具调用（如果有）
}

// ToolCall 工具调用
type ToolCall struct {
	Name      string            `json:"name"`      // 工具名称
	Arguments map[string]string `json:"arguments"` // 工具参数（简化处理，都转为字符串）
}

// MessageType 消息类型
type MessageType string

const (
	MessageTypeUser MessageType = "user" // 用户消息
	MessageTypeAI   MessageType = "ai"   // AI 消息
)

// CodeBlock 代码块
type CodeBlock struct {
	Language string `json:"language"` // 语言（如 "go", "typescript"）
	Code     string `json:"code"`     // 代码内容
}

// DailySummary 每日总结（按天统一，包含多个项目）
type DailySummary struct {
	// 基础信息
	ID       string `json:"id"`       // 唯一ID（UUID）
	Date     string `json:"date"`     // 日期 YYYY-MM-DD
	Language string `json:"language"` // 语言：zh/en（根据聊天内容判断）

	// 总结内容（Markdown格式）
	Summary string `json:"summary"` // 总结文本

	// 项目列表（当天涉及的所有项目）
	Projects []*ProjectSummary `json:"projects"`

	// 工作分类统计（跨所有项目）
	WorkCategories *WorkCategories `json:"work_categories"`

	// 会话统计
	TotalSessions int `json:"total_sessions"` // 总会话数

	// 代码变更统计（可选）
	CodeChanges *CodeChangeSummary `json:"code_changes,omitempty"` // 代码变更统计

	// 时间分布（可选）
	TimeDistribution *TimeDistributionSummary `json:"time_distribution,omitempty"` // 时间分布

	// 效率指标（可选）
	EfficiencyMetrics *EfficiencyMetricsSummary `json:"efficiency_metrics,omitempty"` // 效率指标

	// 元数据
	CreatedAt time.Time `json:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
}

// ProjectSummary 项目摘要（总结中每个项目的摘要）
type ProjectSummary struct {
	ProjectName string `json:"project_name"` // 项目名称
	ProjectPath string `json:"project_path"` // 项目路径
	WorkspaceID string `json:"workspace_id"` // 工作区ID

	// 该项目的工作内容
	WorkItems []*WorkItem `json:"work_items"` // 具体工作项

	// 该项目的会话
	Sessions []*DailySessionSummary `json:"sessions"` // 涉及的会话列表

	// 该项目的统计
	SessionCount int `json:"session_count"` // 会话数量

	// 该项目的代码变更统计（可选）
	CodeChanges *CodeChangeSummary `json:"code_changes,omitempty"` // 代码变更统计

	// 该项目的活跃时段（可选）
	ActiveHours []int `json:"active_hours,omitempty"` // 活跃时段（小时，0-23）
}

// WorkItem 具体工作项
type WorkItem struct {
	Category    string `json:"category"`    // 工作类型：requirements_discussion/coding/problem_solving/refactoring等
	Description string `json:"description"` // 工作描述
	SessionID   string `json:"session_id"`  // 关联的会话ID
}

// WorkCategories 工作分类统计（跨所有项目）
type WorkCategories struct {
	RequirementsDiscussion int `json:"requirements_discussion"` // 需求讨论
	Coding                 int `json:"coding"`                  // 编码
	ProblemSolving         int `json:"problem_solving"`         // 问题排查
	Refactoring            int `json:"refactoring"`             // 重构
	CodeReview             int `json:"code_review"`             // 代码审查
	Documentation          int `json:"documentation"`           // 文档编写
	Testing                int `json:"testing"`                 // 测试
	Other                  int `json:"other"`                   // 其他
}

// DailySessionSummary 会话摘要（用于每日总结）
type DailySessionSummary struct {
	SessionID    string `json:"session_id"`    // 会话ID
	Name         string `json:"name"`          // 会话名称
	ProjectName  string `json:"project_name"`  // 所属项目
	CreatedAt    int64  `json:"created_at"`    // 创建时间戳
	UpdatedAt    int64  `json:"updated_at"`    // 更新时间戳
	MessageCount int    `json:"message_count"` // 消息数量
	Duration     int64  `json:"duration"`      // 持续时长（毫秒，从 CreatedAt 到 UpdatedAt）
}

// TimeDistributionSummary 时间分布汇总
type TimeDistributionSummary struct {
	Morning   TimeSlotStats `json:"morning"`   // 上午（9-12）
	Afternoon TimeSlotStats `json:"afternoon"` // 下午（14-18）
	Evening   TimeSlotStats `json:"evening"`   // 晚上（19-22）
	Night     TimeSlotStats `json:"night"`     // 夜间（22-2）
}

// TimeSlotStats 时段统计
type TimeSlotStats struct {
	Sessions int     `json:"sessions"` // 会话数
	Hours    float64 `json:"hours"`    // 总时长（小时）
}

// EfficiencyMetricsSummary 效率指标汇总
type EfficiencyMetricsSummary struct {
	AvgSessionDuration    float64 `json:"avg_session_duration"`     // 平均会话时长（分钟）
	AvgMessagesPerSession float64 `json:"avg_messages_per_session"` // 平均消息数
	TotalActiveTime       float64 `json:"total_active_time"`        // 总活跃时长（小时）
}

// WeeklySummary 每周总结（按周统一，包含多个项目）
type WeeklySummary struct {
	// 基础信息
	ID        string `json:"id"`         // 唯一ID（UUID）
	WeekStart string `json:"week_start"` // 周起始日期 YYYY-MM-DD（周一）
	WeekEnd   string `json:"week_end"`   // 周结束日期 YYYY-MM-DD（周日）
	Language  string `json:"language"`   // 语言：zh/en

	// 总结内容（Markdown格式）
	Summary string `json:"summary"` // 总结文本

	// 项目列表（本周涉及的所有项目）
	Projects []*WeeklyProjectSummary `json:"projects"`

	// 工作分类统计（跨所有项目）
	WorkCategories *WorkCategories `json:"work_categories"`

	// 统计信息
	TotalSessions int `json:"total_sessions"` // 总会话数
	WorkingDays   int `json:"working_days"`   // 有数据的工作日数

	// 代码变更统计（可选）
	CodeChanges *CodeChangeSummary `json:"code_changes,omitempty"`

	// 关键成就列表
	KeyAccomplishments []string `json:"key_accomplishments,omitempty"`

	// 幂等性支持
	DataHash string `json:"data_hash,omitempty"` // 源数据哈希，用于检测变化

	// 元数据
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WeeklyProjectSummary 周报项目摘要
type WeeklyProjectSummary struct {
	ProjectName  string             `json:"project_name"`  // 项目名称
	ProjectPath  string             `json:"project_path"`  // 项目路径
	SessionCount int                `json:"session_count"` // 会话数量
	CodeChanges  *CodeChangeSummary `json:"code_changes,omitempty"`
	WorkItems    []*WorkItem        `json:"work_items,omitempty"` // 工作项汇总
}

// ActiveLevel 会话活跃等级常量
const (
	ActiveLevelFocused  = 0 // 聚焦（当前活跃）
	ActiveLevelOpen     = 1 // 打开（面板可见）
	ActiveLevelClosed   = 2 // 关闭（面板不可见且未归档）
	ActiveLevelArchived = 3 // 归档
)

// HealthStatus 会话健康状态
type HealthStatus string

const (
	HealthStatusHealthy  HealthStatus = "healthy"  // 健康
	HealthStatusWarning  HealthStatus = "warning"  // 警告
	HealthStatusCritical HealthStatus = "critical" // 危险
)

// ActiveSessionsOverview 活跃会话概览
type ActiveSessionsOverview struct {
	Focused       *ActiveSession   `json:"focused"`        // 当前聚焦的会话
	OpenSessions  []*ActiveSession `json:"open_sessions"`  // 其他打开的会话（按熵值降序）
	ClosedCount   int              `json:"closed_count"`   // 已关闭数量
	ArchivedCount int              `json:"archived_count"` // 已归档数量
}

// ActiveSession 活跃会话信息
type ActiveSession struct {
	ComposerID          string       `json:"composer_id"`           // 会话 ID
	Name                string       `json:"name"`                  // 会话名称
	Entropy             float64      `json:"entropy"`               // 熵值
	ContextUsagePercent float64      `json:"context_usage_percent"` // 上下文使用率
	Status              HealthStatus `json:"status"`                // 健康状态
	Warning             string       `json:"warning,omitempty"`     // 警告提示语
	LastUpdatedAt       int64        `json:"last_updated_at"`       // 最后更新时间戳
}

// CalculateHealthStatus 计算会话健康状态
// 规则：
// - healthy: 熵值 < 40 且 上下文 < 60%
// - warning: 熵值 40-70 或 上下文 60-80%
// - critical: 熵值 > 70 或 上下文 > 80%
func CalculateHealthStatus(entropy, contextUsagePercent float64) (HealthStatus, string) {
	// 危险条件：熵值 > 70 或 上下文 > 80%
	if entropy > 70 || contextUsagePercent > 80 {
		var warning string
		if contextUsagePercent > 80 {
			warning = "会话上下文接近饱和，建议开启新会话继续"
		} else {
			warning = "会话复杂度过高，建议拆分任务"
		}
		return HealthStatusCritical, warning
	}

	// 警告条件：熵值 40-70 或 上下文 60-80%
	if entropy >= 40 || contextUsagePercent >= 60 {
		var warning string
		if contextUsagePercent >= 60 {
			warning = "上下文使用率较高，建议关注会话长度"
		} else {
			warning = "会话复杂度中等，注意保持专注"
		}
		return HealthStatusWarning, warning
	}

	return HealthStatusHealthy, ""
}

// CalculateActiveLevel 计算活跃等级
func CalculateActiveLevel(isArchived, isVisible, isFocused bool) int {
	if isFocused {
		return ActiveLevelFocused
	}
	if isVisible {
		return ActiveLevelOpen
	}
	if isArchived {
		return ActiveLevelArchived
	}
	return ActiveLevelClosed
}

// CalculateEntropy 计算会话熵值
// 基于代码变更行数、文件数量、上下文使用率等指标计算
func CalculateEntropy(linesAdded, linesRemoved, filesChanged int, contextUsagePercent float64) float64 {
	// 代码变更维度（权重 40%）
	totalLines := float64(linesAdded + linesRemoved)
	linesScore := minFloat(totalLines/500.0*100, 100) * 0.4

	// 文件变更维度（权重 30%）
	filesScore := minFloat(float64(filesChanged)/10.0*100, 100) * 0.3

	// 上下文使用率维度（权重 30%）
	contextScore := contextUsagePercent * 0.3

	return linesScore + filesScore + contextScore
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
