package notification

import "github.com/cocursor/backend/internal/domain/notification"

// Pusher 推送接口（定义在 application 层）
// 这是应用层需要的技术能力，不是领域概念
type Pusher interface {
	PushToTeam(teamCode string, notification *notification.Notification) error
}
