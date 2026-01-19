package rag

import "github.com/google/wire"

// ProviderSet RAG 应用层 ProviderSet
var ProviderSet = wire.NewSet(
	NewRAGInitializer,
	ProvideScanScheduler,
	// 注意：RAGService, SearchService 通过 RAGInitializer 初始化
	// ScanScheduler 通过 ProvideScanScheduler 提供（总是可用，但可能禁用）
)
