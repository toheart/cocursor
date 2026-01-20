package rag

import (
	"time"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	infraRAG "github.com/cocursor/backend/internal/infrastructure/rag"
)

// ProvideScanScheduler 提供扫描调度器（如果未配置，返回禁用的调度器）
// 注意：这个函数在 wire 初始化时调用，此时配置可能还未设置
// 实际的启用/禁用逻辑在 Start() 方法中根据配置决定
func ProvideScanScheduler(
	ragInitializer *RAGInitializer,
	projectManager *appCursor.ProjectManager,
	indexStatusRepo domainRAG.IndexStatusRepository,
	configManager *infraRAG.ConfigManager,
) *ScanScheduler {
	// 读取配置
	config, err := configManager.ReadConfig()
	if err != nil {
		// 配置读取失败，返回禁用的调度器
		scheduler := NewScanScheduler(
			nil, // chunkService 稍后通过 initializer 获取
			projectManager,
			indexStatusRepo,
			&ScanConfig{
				Enabled:     false,
				Interval:    1 * time.Hour,
				BatchSize:   10,
				Concurrency: 3,
			},
		)
		scheduler.SetRAGInitializer(ragInitializer)
		return scheduler
	}

	// 创建扫描配置
	scanConfig := &ScanConfig{
		Enabled:     config.ScanConfig.Enabled,
		Interval:    ParseScanInterval(config.ScanConfig.Interval),
		BatchSize:   config.ScanConfig.BatchSize,
		Concurrency: config.ScanConfig.Concurrency,
	}

	// 如果未配置 Embedding API，返回禁用的调度器
	if config.EmbeddingAPI.URL == "" || config.EmbeddingAPI.APIKey == "" {
		scanConfig.Enabled = false
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
