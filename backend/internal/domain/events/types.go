// Package events 定义领域事件类型和接口
// 用于系统内部的事件驱动通信
package events

import "time"

// EventType 事件类型标识
type EventType string

// 会话文件相关事件类型
const (
	// SessionFileCreated 会话文件创建事件
	SessionFileCreated EventType = "session.file.created"
	// SessionFileModified 会话文件修改事件
	SessionFileModified EventType = "session.file.modified"
	// SessionFileDeleted 会话文件删除事件
	SessionFileDeleted EventType = "session.file.deleted"
)

// 工作区相关事件类型
const (
	// WorkspaceCreated 工作区创建事件
	WorkspaceCreated EventType = "workspace.created"
	// WorkspaceDeleted 工作区删除事件
	WorkspaceDeleted EventType = "workspace.deleted"
)

// Event 领域事件接口
// 所有事件类型都必须实现此接口
type Event interface {
	// Type 返回事件类型
	Type() EventType
	// Timestamp 返回事件发生时间
	Timestamp() time.Time
}
