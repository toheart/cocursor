package rag

import "github.com/google/wire"

// ProviderSet RAG 基础设施层 ProviderSet
var ProviderSet = wire.NewSet(
	NewConfigManager,
)
