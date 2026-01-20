package events

import "time"

// WorkspaceEvent 工作区变更事件
// 当 workspaceStorage 目录下的工作区被创建或删除时触发
type WorkspaceEvent struct {
	// EventType 事件类型（created/deleted）
	EventType EventType
	// WorkspaceID 工作区 ID（目录名/哈希值）
	WorkspaceID string
	// ProjectPath 项目路径（从 workspace.json 解析，可能为空）
	ProjectPath string
	// EventTime 事件发生时间
	EventTime time.Time
}

// Type 实现 Event 接口
func (e *WorkspaceEvent) Type() EventType {
	return e.EventType
}

// Timestamp 实现 Event 接口
func (e *WorkspaceEvent) Timestamp() time.Time {
	return e.EventTime
}
