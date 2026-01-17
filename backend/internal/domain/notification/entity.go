package notification

import "time"

// Notification 通知实体
type Notification struct {
	ID        string
	TeamCode  string
	Title     string
	Message   string
	Type      Type
	CreatedAt time.Time
}

// Type 通知类型
type Type int

const (
	// TypeInfo 信息通知
	TypeInfo Type = iota + 1
	// TypeWarning 警告通知
	TypeWarning
	// TypeError 错误通知
	TypeError
)
