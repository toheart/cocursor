package application

import (
	"github.com/google/wire"
	"github.com/cocursor/backend/internal/application/cursor"
	"github.com/cocursor/backend/internal/application/notification"
)

// ProviderSet Application 层总 ProviderSet
var ProviderSet = wire.NewSet(
	notification.ProviderSet,
	cursor.ProviderSet,
	// 可以继续添加其他应用服务模块
)
