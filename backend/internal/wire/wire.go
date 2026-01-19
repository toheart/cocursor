//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/cocursor/backend/internal/application"
	appNotification "github.com/cocursor/backend/internal/application/notification"
	"github.com/cocursor/backend/internal/domain/notification"
	"github.com/cocursor/backend/internal/infrastructure"
	infraNotification "github.com/cocursor/backend/internal/infrastructure/notification"
	"github.com/cocursor/backend/internal/interfaces"
	"github.com/google/wire"
)

// InitializeAll 初始化所有服务（HTTP + MCP）
func InitializeAll() (*App, error) {
	wire.Build(
		// 按层组合 ProviderSet
		infrastructure.ProviderSet, // 基础设施层
		notification.ProviderSet,   // 领域层（按需引入）
		application.ProviderSet,    // 应用层
		interfaces.ProviderSet,      // 接口层
		// 接口绑定：application.Pusher -> infrastructure.Pusher
		wire.Bind(
			new(appNotification.Pusher),
			new(*infraNotification.WebSocketPusher),
		),
		NewApp, // 组合所有服务的应用结构
	)
	return nil, nil
}
