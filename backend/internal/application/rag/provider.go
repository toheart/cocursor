package rag

import (
	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	infraRAG "github.com/cocursor/backend/internal/infrastructure/rag"
)

// ProvideScanScheduler 提供扫描调度器（事件驱动模式）
// 注意：这个函数在 wire 初始化时调用，此时配置可能还未设置
// RAG 索引通过 FileWatcher 事件驱动 + 用户手动触发全量索引
func ProvideScanScheduler(
	ragInitializer *RAGInitializer,
	projectManager *appCursor.ProjectManager,
	indexStatusRepo domainRAG.IndexStatusRepository,
	configManager *infraRAG.ConfigManager,
) *ScanScheduler {
	// 读取配置
	config, err := configManager.ReadConfig()

	// 创建扫描配置（仅 BatchSize 和 Concurrency）
	scanConfig := &ScanConfig{
		BatchSize:   10,
		Concurrency: 3,
	}

	if err == nil {
		if config.IndexConfig.BatchSize > 0 {
			scanConfig.BatchSize = config.IndexConfig.BatchSize
		}
		if config.IndexConfig.Concurrency > 0 {
			scanConfig.Concurrency = config.IndexConfig.Concurrency
		}
	}

	// 创建调度器（chunkService 稍后通过 initializer 设置）
	scheduler := NewScanScheduler(
		nil, // 稍后通过 initializer 设置
		projectManager,
		indexStatusRepo,
		scanConfig,
	)
	scheduler.SetRAGInitializer(ragInitializer)

	return scheduler
}
