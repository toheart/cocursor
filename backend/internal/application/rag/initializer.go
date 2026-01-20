package rag

import (
	"fmt"
	"os"

	"log/slog"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	"github.com/cocursor/backend/internal/infrastructure/embedding"
	"github.com/cocursor/backend/internal/infrastructure/log"
	infraRAG "github.com/cocursor/backend/internal/infrastructure/rag"
	"github.com/cocursor/backend/internal/infrastructure/vector"
)

// RAGInitializer RAG 服务初始化器
type RAGInitializer struct {
	configManager     *infraRAG.ConfigManager
	sessionService    *appCursor.SessionService
	projectManager    *appCursor.ProjectManager
	chunkRepo         domainRAG.ChunkRepository
	indexStatusRepo   domainRAG.IndexStatusRepository
	enrichmentQueue   domainRAG.EnrichmentQueueRepository
	qdrantManager     *vector.QdrantManager  // Qdrant 管理器
	chunkService      *ChunkService          // 知识片段服务
	searchService     *SearchService         // 搜索服务
	enrichmentService *EnrichmentService     // 增强服务
	embeddingClient   *embedding.Client      // Embedding 客户端
	llmClient         *LLMClient             // LLM 客户端（可选）
	initialized       bool                   // 是否已初始化
	logger            *slog.Logger
}

// NewRAGInitializer 创建 RAG 初始化器
func NewRAGInitializer(
	configManager *infraRAG.ConfigManager,
	sessionService *appCursor.SessionService,
	projectManager *appCursor.ProjectManager,
	chunkRepo domainRAG.ChunkRepository,
	indexStatusRepo domainRAG.IndexStatusRepository,
	enrichmentQueue domainRAG.EnrichmentQueueRepository,
) *RAGInitializer {
	return &RAGInitializer{
		configManager:   configManager,
		sessionService:  sessionService,
		projectManager:  projectManager,
		chunkRepo:       chunkRepo,
		indexStatusRepo: indexStatusRepo,
		enrichmentQueue: enrichmentQueue,
		logger:          log.NewModuleLogger("rag", "initializer"),
	}
}

// InitializeServices 初始化 RAG 服务（如果已配置）
func (i *RAGInitializer) InitializeServices() (*ChunkService, *SearchService, *ScanScheduler, *EnrichmentService, *vector.QdrantManager, error) {
	// 如果已初始化，直接返回
	if i.initialized {
		return i.chunkService, i.searchService, nil, i.enrichmentService, i.qdrantManager, nil
	}

	config, err := i.configManager.ReadConfig()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to read config: %w", err)
	}

	// 检查是否已配置 Embedding API
	if config.EmbeddingAPI.URL == "" || config.EmbeddingAPI.APIKey == "" || config.EmbeddingAPI.Model == "" {
		// 未配置，返回 nil（服务可选）
		return nil, nil, nil, nil, nil, nil
	}

	// 创建 Embedding 客户端
	i.embeddingClient = embedding.NewClient(
		config.EmbeddingAPI.URL,
		config.EmbeddingAPI.APIKey,
		config.EmbeddingAPI.Model,
	)

	// 创建 LLM 客户端（可选，用于增强）
	if config.LLMChatAPI.URL != "" && config.LLMChatAPI.APIKey != "" && config.LLMChatAPI.Model != "" {
		i.llmClient = NewLLMClient(
			config.LLMChatAPI.URL,
			config.LLMChatAPI.APIKey,
			config.LLMChatAPI.Model,
		)
		i.logger.Info("LLM client initialized for enrichment",
			"url", config.LLMChatAPI.URL,
			"model", config.LLMChatAPI.Model,
		)
	}

	// 创建 Qdrant 管理器
	binaryPath := config.Qdrant.BinaryPath
	dataPath := config.Qdrant.DataPath

	// 如果路径为空，使用默认路径
	if binaryPath == "" {
		var err error
		binaryPath, err = vector.GetQdrantInstallPath()
		if err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("failed to get qdrant install path: %w", err)
		}
	}
	if dataPath == "" {
		var err error
		dataPath, err = vector.GetQdrantDataPath()
		if err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("failed to get qdrant data path: %w", err)
		}
	}

	i.qdrantManager = vector.NewQdrantManager(binaryPath, dataPath)

	// 启动 Qdrant（如果二进制文件存在）
	if _, err := os.Stat(binaryPath); err == nil {
		if err := i.qdrantManager.Start(); err != nil {
			// 记录错误但不阻止初始化
			i.logger.Warn("Failed to start Qdrant",
				"error", err,
			)
		} else {
			// 获取向量维度并确保集合存在
			dimension, err := i.embeddingClient.GetVectorDimension()
			if err == nil {
				if err := i.qdrantManager.EnsureCollections(uint64(dimension)); err != nil {
					i.logger.Warn("Failed to ensure Qdrant collections",
						"vector_dimension", dimension,
						"error", err,
					)
				}
			}
		}
	}

	// 创建 ChunkService（核心索引服务）
	i.chunkService = NewChunkService(
		i.sessionService,
		i.embeddingClient,
		i.qdrantManager,
		i.chunkRepo,
		i.indexStatusRepo,
		i.enrichmentQueue,
		i.projectManager,
	)

	// 创建搜索服务
	i.searchService = NewSearchService(
		i.embeddingClient,
		i.qdrantManager,
		i.chunkRepo,
	)

	// 创建扫描调度器
	scanConfig := &ScanConfig{
		Enabled:     config.ScanConfig.Enabled,
		Interval:    ParseScanInterval(config.ScanConfig.Interval),
		BatchSize:   config.ScanConfig.BatchSize,
		Concurrency: config.ScanConfig.Concurrency,
	}

	scanScheduler := NewScanScheduler(
		i.chunkService,
		i.projectManager,
		i.indexStatusRepo,
		scanConfig,
	)

	// 创建增强服务
	i.enrichmentService = NewEnrichmentService(
		i.chunkRepo,
		i.enrichmentQueue,
		i.llmClient,
		i.qdrantManager,
	)

	// 启动增强服务 Worker
	i.enrichmentService.StartWorkers()

	i.initialized = true
	i.logger.Info("RAG services initialized successfully")

	return i.chunkService, i.searchService, scanScheduler, i.enrichmentService, i.qdrantManager, nil
}

// GetChunkService 获取 ChunkService（延迟初始化时使用）
func (i *RAGInitializer) GetChunkService() *ChunkService {
	if i.chunkService != nil {
		return i.chunkService
	}

	// 尝试初始化
	chunkService, _, _, _, _, err := i.InitializeServices()
	if err != nil {
		i.logger.Warn("Failed to initialize services", "error", err)
		return nil
	}
	return chunkService
}

// GetSearchService 获取 SearchService
func (i *RAGInitializer) GetSearchService() *SearchService {
	if i.searchService != nil {
		return i.searchService
	}

	// 尝试初始化
	_, searchService, _, _, _, err := i.InitializeServices()
	if err != nil {
		i.logger.Warn("Failed to initialize services", "error", err)
		return nil
	}
	return searchService
}

// GetEnrichmentService 获取 EnrichmentService
func (i *RAGInitializer) GetEnrichmentService() *EnrichmentService {
	if i.enrichmentService != nil {
		return i.enrichmentService
	}

	// 尝试初始化
	_, _, _, enrichmentService, _, err := i.InitializeServices()
	if err != nil {
		i.logger.Warn("Failed to initialize services", "error", err)
		return nil
	}
	return enrichmentService
}

// StopQdrant 停止 Qdrant 服务（如果已启动）
func (i *RAGInitializer) StopQdrant() error {
	if i.qdrantManager != nil {
		return i.qdrantManager.Stop()
	}
	return nil
}

// StopEnrichmentWorkers 停止增强服务 Worker
func (i *RAGInitializer) StopEnrichmentWorkers() {
	if i.enrichmentService != nil {
		i.enrichmentService.StopWorkers()
	}
}

// Shutdown 关闭所有 RAG 服务
func (i *RAGInitializer) Shutdown() error {
	i.StopEnrichmentWorkers()
	return i.StopQdrant()
}

// GetQdrantManager 获取 Qdrant 管理器（如果已初始化）
func (i *RAGInitializer) GetQdrantManager() *vector.QdrantManager {
	return i.qdrantManager
}

// IsInitialized 检查是否已初始化
func (i *RAGInitializer) IsInitialized() bool {
	return i.initialized
}
