package p2p

import (
	"encoding/json"
	"time"
)

// EventType WebSocket 事件类型
type EventType string

const (
	// 技能相关事件
	EventSkillPublished EventType = "skill_published" // 技能发布
	EventSkillUpdated   EventType = "skill_updated"   // 技能更新
	EventSkillDeleted   EventType = "skill_deleted"   // 技能删除

	// 成员相关事件
	EventMemberJoined  EventType = "member_joined"  // 成员加入
	EventMemberLeft    EventType = "member_left"    // 成员离开
	EventMemberOnline  EventType = "member_online"  // 成员上线
	EventMemberOffline EventType = "member_offline" // 成员离线

	// 团队相关事件
	EventTeamDissolved EventType = "team_dissolved" // 团队解散

	// 连接相关事件
	EventPing EventType = "ping" // 心跳
	EventPong EventType = "pong" // 心跳响应

	// 认证事件
	EventAuth       EventType = "auth"        // 认证请求
	EventAuthResult EventType = "auth_result" // 认证结果

	// 同步事件
	EventSyncRequest  EventType = "sync_request"  // 请求同步
	EventSyncResponse EventType = "sync_response" // 同步响应

	// 协作相关事件
	EventMemberStatusChanged EventType = "member_status_changed" // 成员工作状态变更
	EventDailySummaryShared  EventType = "daily_summary_shared"  // 日报分享

	// 周报相关事件
	EventProjectConfigUpdated EventType = "project_config_updated" // 项目配置更新
)

// Event WebSocket 事件
type Event struct {
	Type      EventType       `json:"type"`      // 事件类型
	TeamID    string          `json:"team_id"`   // 团队 ID
	Timestamp time.Time       `json:"timestamp"` // 时间戳
	Payload   json.RawMessage `json:"payload"`   // 事件数据
}

// NewEvent 创建新事件
func NewEvent(eventType EventType, teamID string, payload interface{}) (*Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Event{
		Type:      eventType,
		TeamID:    teamID,
		Timestamp: time.Now(),
		Payload:   data,
	}, nil
}

// ParsePayload 解析事件数据
func (e *Event) ParsePayload(v interface{}) error {
	return json.Unmarshal(e.Payload, v)
}

// SkillPublishedPayload 技能发布事件数据
type SkillPublishedPayload struct {
	PluginID       string    `json:"plugin_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Version        string    `json:"version"`
	AuthorID       string    `json:"author_id"`
	AuthorName     string    `json:"author_name"`
	AuthorEndpoint string    `json:"author_endpoint"`
	FileCount      int       `json:"file_count"`
	TotalSize      int64     `json:"total_size"`
	Checksum       string    `json:"checksum"`
	PublishedAt    time.Time `json:"published_at"`
}

// SkillUpdatedPayload 技能更新事件数据
type SkillUpdatedPayload struct {
	PluginID    string    `json:"plugin_id"`
	Version     string    `json:"version"`
	Checksum    string    `json:"checksum"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SkillDeletedPayload 技能删除事件数据
type SkillDeletedPayload struct {
	PluginID  string    `json:"plugin_id"`
	DeletedBy string    `json:"deleted_by"`
	DeletedAt time.Time `json:"deleted_at"`
}

// MemberStatusPayload 成员状态事件数据
type MemberStatusPayload struct {
	MemberID   string `json:"member_id"`
	MemberName string `json:"member_name"`
	Endpoint   string `json:"endpoint,omitempty"`
	IsOnline   bool   `json:"is_online"`
}

// MemberJoinedPayload 成员加入事件数据
type MemberJoinedPayload struct {
	MemberID   string    `json:"member_id"`
	MemberName string    `json:"member_name"`
	Endpoint   string    `json:"endpoint"`
	JoinedAt   time.Time `json:"joined_at"`
}

// MemberLeftPayload 成员离开事件数据
type MemberLeftPayload struct {
	MemberID   string    `json:"member_id"`
	MemberName string    `json:"member_name"`
	LeftAt     time.Time `json:"left_at"`
}

// TeamDissolvedPayload 团队解散事件数据
type TeamDissolvedPayload struct {
	TeamID      string    `json:"team_id"`
	TeamName    string    `json:"team_name"`
	DissolvedBy string    `json:"dissolved_by"`
	DissolvedAt time.Time `json:"dissolved_at"`
}

// AuthPayload 认证请求数据
type AuthPayload struct {
	MemberID   string `json:"member_id"`
	MemberName string `json:"member_name"`
	Endpoint   string `json:"endpoint"`
}

// AuthResultPayload 认证结果数据
type AuthResultPayload struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// SyncRequestPayload 同步请求数据
type SyncRequestPayload struct {
	LastSyncTime time.Time `json:"last_sync_time"`
}

// SyncResponsePayload 同步响应数据
type SyncResponsePayload struct {
	Skills   []SkillPublishedPayload `json:"skills"`
	Deleted  []string                `json:"deleted"`   // 已删除的技能 ID
	SyncTime time.Time               `json:"sync_time"`
	HasMore  bool                    `json:"has_more"`
}

// MemberWorkStatusPayload 成员工作状态变更事件数据
type MemberWorkStatusPayload struct {
	MemberID      string    `json:"member_id"`       // 成员 ID
	MemberName    string    `json:"member_name"`     // 成员名称
	ProjectName   string    `json:"project_name"`    // 当前项目名
	CurrentFile   string    `json:"current_file"`    // 当前文件（相对路径）
	LastActiveAt  time.Time `json:"last_active_at"`  // 最后活跃时间
	StatusVisible bool      `json:"status_visible"`  // 是否公开状态
}

// DailySummarySharedPayload 日报分享事件数据
type DailySummarySharedPayload struct {
	MemberID      string    `json:"member_id"`      // 成员 ID
	MemberName    string    `json:"member_name"`    // 成员名称
	Date          string    `json:"date"`           // 日期 YYYY-MM-DD
	TotalSessions int       `json:"total_sessions"` // 会话总数
	ProjectCount  int       `json:"project_count"`  // 项目数量
	SharedAt      time.Time `json:"shared_at"`      // 分享时间
}

// ProjectConfigPayload 项目配置事件数据
type ProjectConfigPayload struct {
	Projects  []ProjectMatcherPayload `json:"projects"`   // 项目列表
	UpdatedAt time.Time               `json:"updated_at"` // 更新时间
}

// ProjectMatcherPayload 项目匹配规则
type ProjectMatcherPayload struct {
	ID      string `json:"id"`       // 规则 ID
	Name    string `json:"name"`     // 显示名称
	RepoURL string `json:"repo_url"` // Git Remote URL
}
