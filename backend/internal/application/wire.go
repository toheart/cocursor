package application

import (
	appCodeanalysis "github.com/cocursor/backend/internal/application/codeanalysis"
	"github.com/cocursor/backend/internal/application/cursor"
	appLifecycle "github.com/cocursor/backend/internal/application/lifecycle"
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
	appCodeanalysis.ProviderSet, // 代码分析应用层
	appLifecycle.ProviderSet,    // 生命周期管理
	// 可以继续添加其他应用服务模块
)
