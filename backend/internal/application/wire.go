package application

import (
	"github.com/cocursor/backend/internal/application/cursor"
	"github.com/cocursor/backend/internal/application/marketplace"
	"github.com/cocursor/backend/internal/application/notification"
	appRAG "github.com/cocursor/backend/internal/application/rag"
	"github.com/google/wire"
)

// ProviderSet Application 层总 ProviderSet
var ProviderSet = wire.NewSet(
	notification.ProviderSet,
	cursor.ProviderSet,
	marketplace.ProviderSet,
	appRAG.ProviderSet,
	// 可以继续添加其他应用服务模块
)
