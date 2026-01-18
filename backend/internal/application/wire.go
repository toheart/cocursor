package application

import (
	"github.com/cocursor/backend/internal/application/cursor"
	"github.com/cocursor/backend/internal/application/marketplace"
	"github.com/cocursor/backend/internal/application/notification"
	"github.com/google/wire"
)

// ProviderSet Application 层总 ProviderSet
var ProviderSet = wire.NewSet(
	notification.ProviderSet,
	cursor.ProviderSet,
	marketplace.ProviderSet,
	// 可以继续添加其他应用服务模块
)
