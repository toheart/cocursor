package infrastructure

import (
	"github.com/cocursor/backend/internal/infrastructure/config"
	"github.com/cocursor/backend/internal/infrastructure/marketplace"
	"github.com/cocursor/backend/internal/infrastructure/notification"
	"github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/cocursor/backend/internal/infrastructure/websocket"
	"github.com/google/wire"
)

// ProviderSet Infrastructure 层总 ProviderSet
var ProviderSet = wire.NewSet(
	config.ProviderSet,
	websocket.ProviderSet,
	notification.ProviderSet,
	marketplace.ProviderSet,
	storage.ProviderSet,
	// 可以继续添加其他基础设施模块
)
