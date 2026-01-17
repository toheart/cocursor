package notification

import (
	"github.com/google/wire"
	"github.com/cocursor/backend/internal/domain/notification"
)

// ProviderSet 通知基础设施层 ProviderSet
var ProviderSet = wire.NewSet(
	NewMemoryRepository,
	NewWebSocketPusher,
	// 接口绑定：domain.Repository -> infrastructure.Repository
	wire.Bind(
		new(notification.Repository),
		new(*MemoryRepository),
	),
)
