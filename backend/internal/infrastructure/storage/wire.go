package storage

import "github.com/google/wire"

// ProviderSet Storage 基础设施层 ProviderSet
var ProviderSet = wire.NewSet(
	ProvideDB,                          // 提供数据库连接
	NewDailySummaryRepository,          // 每日总结仓储
	NewOpenSpecWorkflowRepository,      // OpenSpec 工作流仓储
	NewWorkspaceSessionRepository,      // 工作区会话仓储
	NewWorkspaceFileMetadataRepository, // 工作区文件元数据仓储
	NewChunkRepository,                 // RAG 知识片段仓储
	NewIndexStatusRepository,           // RAG 索引状态仓储
	NewEnrichmentQueueRepository,       // RAG 增强队列仓储
)
