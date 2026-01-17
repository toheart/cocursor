package notification

import (
	"github.com/cocursor/backend/internal/application/notification"
	domainNotification "github.com/cocursor/backend/internal/domain/notification"
	"github.com/cocursor/backend/internal/infrastructure/websocket"
)

// WebSocketPusher WebSocket 推送实现
type WebSocketPusher struct {
	hub *websocket.Hub
}

// NewWebSocketPusher 创建 WebSocket 推送器
func NewWebSocketPusher(hub *websocket.Hub) *WebSocketPusher {
	return &WebSocketPusher{hub: hub}
}

// PushToTeam 推送到团队
func (p *WebSocketPusher) PushToTeam(teamCode string, n *domainNotification.Notification) error {
	return p.hub.BroadcastToTeam(teamCode, n)
}

// 编译时检查接口实现
var _ notification.Pusher = (*WebSocketPusher)(nil)
