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
	configManager  *infraRAG.ConfigManager
	sessionService *appCursor.SessionService
	projectManager *appCursor.ProjectManager
	ragRepo        domainRAG.RAGRepository
	qdrantManager  *vector.QdrantManager // 保存 QdrantManager 引用以便管理生命周期
	logger         *slog.Logger
}

// NewRAGInitializer 创建 RAG 初始化器
func NewRAGInitializer(
	configManager *infraRAG.ConfigManager,
	sessionService *appCursor.SessionService,
	projectManager *appCursor.ProjectManager,
	ragRepo domainRAG.RAGRepository,
) *RAGInitializer {
	return &RAGInitializer{
		configManager:  configManager,
		sessionService: sessionService,
		projectManager: projectManager,
		ragRepo:        ragRepo,
		logger:         log.NewModuleLogger("rag", "initializer"),
	}
}

// InitializeServices 初始化 RAG 服务（如果已配置）
func (i *RAGInitializer) InitializeServices() (*RAGService, *SearchService, *ScanScheduler, *vector.QdrantManager, error) {
	config, err := i.configManager.ReadConfig()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to read config: %w", err)
	}

	// 检查是否已配置 Embedding API
	if config.EmbeddingAPI.URL == "" || config.EmbeddingAPI.APIKey == "" || config.EmbeddingAPI.Model == "" {
		// 未配置，返回 nil（服务可选）
		return nil, nil, nil, nil, nil
	}

	// 检查是否已配置 LLM Chat API（必需）
	if config.LLMChatAPI.URL == "" || config.LLMChatAPI.APIKey == "" || config.LLMChatAPI.Model == "" {
		// LLM Chat API 是必需的，未配置则返回错误
		return nil, nil, nil, nil, fmt.Errorf("LLM Chat API configuration is required")
	}

	// 创建 Embedding 客户端
	embeddingClient := embedding.NewClient(
		config.EmbeddingAPI.URL,
		config.EmbeddingAPI.APIKey,
		config.EmbeddingAPI.Model,
	)

	// 创建 LLM 客户端（如果配置了 LLM Chat API）
	var llmClient *LLMClient
	if config.LLMChatAPI.URL != "" && config.LLMChatAPI.APIKey != "" && config.LLMChatAPI.Model != "" {
		llmClient = NewLLMClient(
			config.LLMChatAPI.URL,
			config.LLMChatAPI.APIKey,
			config.LLMChatAPI.Model,
		)
		i.logger.Info("LLM client initialized",
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
			return nil, nil, nil, nil, fmt.Errorf("failed to get qdrant install path: %w", err)
		}
	}
	if dataPath == "" {
		var err error
		dataPath, err = vector.GetQdrantDataPath()
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to get qdrant data path: %w", err)
		}
	}

	qdrantManager := vector.NewQdrantManager(binaryPath, dataPath)
	i.qdrantManager = qdrantManager // 保存引用

	// 启动 Qdrant（如果二进制文件存在）
	if _, err := os.Stat(binaryPath); err == nil {
		if err := qdrantManager.Start(); err != nil {
			// 记录错误但不阻止初始化
			i.logger.Warn("Failed to start Qdrant",
				"error", err,
			)
		} else {
			// 获取向量维度并确保集合存在
			dimension, err := embeddingClient.GetVectorDimension()
			if err == nil {
				if err := qdrantManager.EnsureCollections(uint64(dimension)); err != nil {
					i.logger.Warn("Failed to ensure Qdrant collections",
						"vector_dimension", dimension,
						"error", err,
					)
				}
			}
		}
	}

	// 创建 RAG 服务
	ragService := NewRAGService(
		i.sessionService,
		embeddingClient,
		llmClient,
		qdrantManager,
		i.ragRepo,
		i.projectManager,
	)

	// 创建搜索服务
	searchService := NewSearchService(
		embeddingClient,
		qdrantManager,
		i.ragRepo,
	)

	// 创建扫描调度器
	scanConfig := &ScanConfig{
		Enabled:     config.ScanConfig.Enabled,
		Interval:    ParseScanInterval(config.ScanConfig.Interval),
		BatchSize:   config.ScanConfig.BatchSize,
		Concurrency: config.ScanConfig.Concurrency,
	}

	scanScheduler := NewScanScheduler(
		ragService,
		i.projectManager,
		i.ragRepo,
		scanConfig,
	)

	return ragService, searchService, scanScheduler, qdrantManager, nil
}

// StopQdrant 停止 Qdrant 服务（如果已启动）
func (i *RAGInitializer) StopQdrant() error {
	if i.qdrantManager != nil {
		return i.qdrantManager.Stop()
	}
	return nil
}

// GetQdrantManager 获取 Qdrant 管理器（如果已初始化）
func (i *RAGInitializer) GetQdrantManager() *vector.QdrantManager {
	return i.qdrantManager
}
