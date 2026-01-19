package infrastructure

import (
	"github.com/cocursor/backend/internal/infrastructure/config"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/cocursor/backend/internal/infrastructure/embedding"
	"github.com/cocursor/backend/internal/infrastructure/marketplace"
	"github.com/cocursor/backend/internal/infrastructure/notification"
	infraRAG "github.com/cocursor/backend/internal/infrastructure/rag"
	"github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/cocursor/backend/internal/infrastructure/vector"
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
	infraCursor.ProviderSet, // Cursor 基础设施层
	infraRAG.ProviderSet,    // RAG 基础设施层
	embedding.ProviderSet,    // Embedding 基础设施层
	vector.ProviderSet,      // Vector 基础设施层
	// 可以继续添加其他基础设施模块
)
