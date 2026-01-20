package events

import "time"

// SessionFileEvent 会话文件变更事件
// 当 ~/.cursor/projects/*/agent-transcripts/*.txt 文件发生变更时触发
type SessionFileEvent struct {
	// EventType 事件类型（created/modified/deleted）
	EventType EventType
	// SessionID 会话 ID（文件名去掉 .txt 后缀）
	SessionID string
	// ProjectKey 项目标识（目录名，如 "Users-xibaobao-code-cocursor"）
	ProjectKey string
	// FilePath 文件完整路径
	FilePath string
	// ModTime 文件最后修改时间
	ModTime time.Time
	// FileSize 文件大小（字节）
	FileSize int64
	// EventTime 事件发生时间
	EventTime time.Time
}

// Type 实现 Event 接口
func (e *SessionFileEvent) Type() EventType {
	return e.EventType
}

// Timestamp 实现 Event 接口
func (e *SessionFileEvent) Timestamp() time.Time {
	return e.EventTime
}
