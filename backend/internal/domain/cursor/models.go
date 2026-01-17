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
