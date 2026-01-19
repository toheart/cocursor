package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"log/slog"

	appRAG "github.com/cocursor/backend/internal/application/rag"
	"github.com/cocursor/backend/internal/infrastructure/embedding"
	"github.com/cocursor/backend/internal/infrastructure/log"
	infraRAG "github.com/cocursor/backend/internal/infrastructure/rag"
	"github.com/cocursor/backend/internal/infrastructure/vector"
	"github.com/gin-gonic/gin"
)

// RAGHandler RAG 处理器
type RAGHandler struct {
	ragInitializer *appRAG.RAGInitializer
	configManager  *infraRAG.ConfigManager
	scanScheduler  *appRAG.ScanScheduler
	logger         *slog.Logger
}

// NewRAGHandler 创建 RAG 处理器
func NewRAGHandler(
	ragInitializer *appRAG.RAGInitializer,
	configManager *infraRAG.ConfigManager,
	scanScheduler *appRAG.ScanScheduler,
) *RAGHandler {
	return &RAGHandler{
		ragInitializer: ragInitializer,
		configManager:  configManager,
		scanScheduler:  scanScheduler,
		logger:         log.NewModuleLogger("rag", "handler"),
	}
}

// getServices 获取 RAG 服务（如果已初始化）
func (h *RAGHandler) getServices() (*appRAG.RAGService, *appRAG.SearchService, *appRAG.ScanScheduler, error) {
	ragService, searchService, scanScheduler, _, err := h.ragInitializer.InitializeServices()
	if err != nil {
		return nil, nil, nil, err
	}
	if ragService == nil {
		return nil, nil, nil, fmt.Errorf("RAG services not initialized. Please configure RAG first.")
	}
	return ragService, searchService, scanScheduler, nil
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Query      string   `json:"query" binding:"required"`
	ProjectIDs []string `json:"project_ids,omitempty"`
	Limit      int      `json:"limit,omitempty"`
}

// Search 处理搜索请求
// POST /api/v1/rag/search
func (h *RAGHandler) Search(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	_, searchService, _, err := h.getServices()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := searchService.Search(context.Background(), &appRAG.SearchRequest{
		Query:      req.Query,
		ProjectIDs: req.ProjectIDs,
		Limit:      req.Limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
	})
}

// IndexRequest 索引请求
type IndexRequest struct {
	SessionID string `json:"session_id,omitempty"` // 如果为空，索引所有会话
}

// Index 处理索引请求
// POST /api/v1/rag/index
func (h *RAGHandler) Index(c *gin.Context) {
	var req IndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, _, scanScheduler, err := h.getServices()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SessionID != "" {
		// 索引单个会话
		// 需要从文件路径查找
		// 简化实现：触发全量扫描
		scanScheduler.TriggerScan()
		c.JSON(http.StatusOK, gin.H{
			"message":    "Indexing triggered",
			"session_id": req.SessionID,
		})
	} else {
		// 触发全量扫描
		scanScheduler.TriggerScan()
		c.JSON(http.StatusOK, gin.H{
			"message": "Full scan triggered",
		})
	}
}

// Stats 获取统计信息
// GET /api/v1/rag/stats
func (h *RAGHandler) Stats(c *gin.Context) {
	config, err := h.configManager.ReadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_indexed":         config.TotalIndexed,
		"last_full_scan":        config.LastFullScan,
		"last_incremental_scan": config.LastIncrementalScan,
		"scan_config": gin.H{
			"enabled":     config.ScanConfig.Enabled,
			"interval":    config.ScanConfig.Interval,
			"batch_size":  config.ScanConfig.BatchSize,
			"concurrency": config.ScanConfig.Concurrency,
		},
	})
}

// GetConfig 获取配置
// GET /api/v1/rag/config
func (h *RAGHandler) GetConfig(c *gin.Context) {
	config, err := h.configManager.ReadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 隐藏 API Key
	response := gin.H{
		"embedding_api": gin.H{
			"url":   config.EmbeddingAPI.URL,
			"model": config.EmbeddingAPI.Model,
			// API Key 不返回
		},
		"qdrant": gin.H{
			"version":     config.Qdrant.Version,
			"binary_path": config.Qdrant.BinaryPath,
			"data_path":   config.Qdrant.DataPath,
		},
		"scan_config": config.ScanConfig,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	EmbeddingAPI struct {
		URL    string `json:"url"`
		APIKey string `json:"api_key"`
		Model  string `json:"model"`
	} `json:"embedding_api"`
	LLMChatAPI struct {
		URL    string `json:"url"`     // API URL
		APIKey string `json:"api_key"` // API Key
		Model  string `json:"model"`   // 模型名称
	} `json:"llm_chat_api"`
	ScanConfig struct {
		Enabled     bool   `json:"enabled"`
		Interval    string `json:"interval"` // "30m", "1h", "2h", "6h", "24h", "manual"
		BatchSize   int    `json:"batch_size"`
		Concurrency int    `json:"concurrency"`
	} `json:"scan_config"`
}

// UpdateConfig 更新配置
// POST /api/v1/rag/config
func (h *RAGHandler) UpdateConfig(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.configManager.ReadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 更新配置
	config.EmbeddingAPI.URL = req.EmbeddingAPI.URL
	config.EmbeddingAPI.APIKey = req.EmbeddingAPI.APIKey
	config.EmbeddingAPI.Model = req.EmbeddingAPI.Model

	// 更新 LLM Chat API 配置
	config.LLMChatAPI.URL = req.LLMChatAPI.URL
	config.LLMChatAPI.APIKey = req.LLMChatAPI.APIKey
	config.LLMChatAPI.Model = req.LLMChatAPI.Model

	config.ScanConfig.Enabled = req.ScanConfig.Enabled
	config.ScanConfig.Interval = req.ScanConfig.Interval
	config.ScanConfig.BatchSize = req.ScanConfig.BatchSize
	config.ScanConfig.Concurrency = req.ScanConfig.Concurrency

	if err := h.configManager.WriteConfig(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新初始化 RAG 服务以应用新配置
	// 1. 停止旧的 Qdrant 服务
	if err := h.ragInitializer.StopQdrant(); err != nil {
		h.logger.Error("Failed to stop Qdrant during config update",
			"error", err,
		)
	}

	// 2. 重新初始化服务（使用新配置）
	_, _, _, _, err = h.ragInitializer.InitializeServices()
	if err != nil {
		h.logger.Error("Failed to reinitialize RAG services",
			"error", err,
			"message", "Config saved but service reinitialization failed",
		)
		// 配置已保存，但服务重新初始化失败，仍然返回成功
		// 用户可能需要手动重启服务
	}

	// 3. 更新 ScanScheduler 配置
	if h.scanScheduler != nil {
		scanConfig := &appRAG.ScanConfig{
			Enabled:     config.ScanConfig.Enabled,
			Interval:    appRAG.ParseScanInterval(config.ScanConfig.Interval),
			BatchSize:   config.ScanConfig.BatchSize,
			Concurrency: config.ScanConfig.Concurrency,
		}
		h.scanScheduler.UpdateConfig(scanConfig)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Config updated"})
}

// TestConfigRequest 测试配置请求
type TestConfigRequest struct {
	URL    string `json:"url" binding:"required"`
	APIKey string `json:"api_key" binding:"required"`
	Model  string `json:"model" binding:"required"`
}

// TestConfig 测试配置连接
// POST /api/v1/rag/config/test
func (h *RAGHandler) TestConfig(c *gin.Context) {
	var req TestConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug("Invalid test config request",
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// API Key 脱敏
	apiKeyMasked := ""
	if len(req.APIKey) > 8 {
		apiKeyMasked = req.APIKey[:4] + "..." + req.APIKey[len(req.APIKey)-4:]
	} else {
		apiKeyMasked = "***"
	}

	h.logger.Debug("Testing RAG configuration",
		"url", req.URL,
		"model", req.Model,
		"api_key", apiKeyMasked,
	)

	// 创建临时客户端测试连接
	embeddingClient := embedding.NewClient(req.URL, req.APIKey, req.Model)
	if err := embeddingClient.TestConnection(); err != nil {
		h.logger.Error("RAG configuration test failed",
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	h.logger.Info("RAG configuration test successful")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection test successful",
	})
}

// DownloadQdrantRequest 下载 Qdrant 请求
type DownloadQdrantRequest struct {
	Version string `json:"version,omitempty"` // 版本号，如 "v1.16.3"，为空则使用默认版本
}

// DownloadQdrant 下载并安装 Qdrant
// POST /api/v1/rag/qdrant/download
func (h *RAGHandler) DownloadQdrant(c *gin.Context) {
	var req DownloadQdrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体，使用默认版本
		req.Version = ""
	}

	// 记录请求日志
	h.logger.Debug("Download Qdrant request",
		"version", req.Version,
	)

	// 下载 Qdrant
	installPath, err := vector.DownloadQdrant(req.Version)
	if err != nil {
		h.logger.Error("Failed to download Qdrant",
			"version", req.Version,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"success": false,
		})
		return
	}

	// 更新配置
	config, err := h.configManager.ReadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   fmt.Sprintf("failed to read config: %v", err),
			"success": false,
		})
		return
	}

	// 获取版本号
	version := req.Version
	if version == "" {
		version = "v1.16.3"
	}

	// 更新配置中的 Qdrant 信息
	config.Qdrant.BinaryPath = installPath
	dataPath, err := vector.GetQdrantDataPath()
	if err == nil {
		config.Qdrant.DataPath = dataPath
	}
	config.Qdrant.Version = version

	if err := h.configManager.WriteConfig(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   fmt.Sprintf("failed to update config: %v", err),
			"success": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Qdrant downloaded and installed successfully",
		"install_path": installPath,
		"version":      version,
		"note":         "Please start Qdrant service separately using the start endpoint",
	})
}

// StartQdrant 启动 Qdrant 服务
// POST /api/v1/rag/qdrant/start
func (h *RAGHandler) StartQdrant(c *gin.Context) {
	config, err := h.configManager.ReadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("failed to read config: %v", err),
		})
		return
	}

	binaryPath := config.Qdrant.BinaryPath
	dataPath := config.Qdrant.DataPath

	// 如果路径为空，使用默认路径
	if binaryPath == "" {
		var err error
		binaryPath, err = vector.GetQdrantInstallPath()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   fmt.Sprintf("failed to get qdrant install path: %v", err),
			})
			return
		}
	}
	if dataPath == "" {
		var err error
		dataPath, err = vector.GetQdrantDataPath()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   fmt.Sprintf("failed to get qdrant data path: %v", err),
			})
			return
		}
	}

	qdrantManager := vector.NewQdrantManager(binaryPath, dataPath)

	// 检查是否已在运行
	if qdrantManager.IsRunning() {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Qdrant is already running",
		})
		return
	}

	// 启动 Qdrant
	if err := qdrantManager.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("failed to start Qdrant: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Qdrant started successfully",
	})
}

// StopQdrant 停止 Qdrant 服务
// POST /api/v1/rag/qdrant/stop
func (h *RAGHandler) StopQdrant(c *gin.Context) {
	config, err := h.configManager.ReadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("failed to read config: %v", err),
		})
		return
	}

	binaryPath := config.Qdrant.BinaryPath
	dataPath := config.Qdrant.DataPath

	// 如果路径为空，使用默认路径
	if binaryPath == "" {
		var err error
		binaryPath, err = vector.GetQdrantInstallPath()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   fmt.Sprintf("failed to get qdrant install path: %v", err),
			})
			return
		}
	}
	if dataPath == "" {
		var err error
		dataPath, err = vector.GetQdrantDataPath()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   fmt.Sprintf("failed to get qdrant data path: %v", err),
			})
			return
		}
	}

	qdrantManager := vector.NewQdrantManager(binaryPath, dataPath)

	// 检查是否在运行
	if !qdrantManager.IsRunning() {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Qdrant is not running",
		})
		return
	}

	// 停止 Qdrant
	if err := qdrantManager.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("failed to stop Qdrant: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Qdrant stopped successfully",
	})
}

// GetQdrantStatus 获取 Qdrant 服务状态
// GET /api/v1/rag/qdrant/status
func (h *RAGHandler) GetQdrantStatus(c *gin.Context) {
	config, err := h.configManager.ReadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("failed to read config: %v", err),
		})
		return
	}

	binaryPath := config.Qdrant.BinaryPath
	dataPath := config.Qdrant.DataPath

	// 如果路径为空，使用默认路径
	if binaryPath == "" {
		var err error
		binaryPath, err = vector.GetQdrantInstallPath()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   fmt.Sprintf("failed to get qdrant install path: %v", err),
			})
			return
		}
	}
	if dataPath == "" {
		var err error
		dataPath, err = vector.GetQdrantDataPath()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   fmt.Sprintf("failed to get qdrant data path: %v", err),
			})
			return
		}
	}

	qdrantManager := vector.NewQdrantManager(binaryPath, dataPath)

	// 检查二进制文件是否存在
	binaryExists := false
	if _, err := os.Stat(binaryPath); err == nil {
		binaryExists = true
	}

	// 检查是否在运行
	isRunning := qdrantManager.IsRunning()

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"binary_exists": binaryExists,
		"binary_path":   binaryPath,
		"data_path":     dataPath,
		"is_running":    isRunning,
		"grpc_port":     6334,
		"http_port":     6333,
	})
}

// TestLLMConnectionRequest 测试 LLM 连接请求
type TestLLMConnectionRequest struct {
	URL    string `json:"url" binding:"required"`     // LLM API URL
	APIKey string `json:"api_key" binding:"required"` // LLM API Key
	Model  string `json:"model" binding:"required"`   // 模型名称
}

// TestLLMConnection 测试 LLM API 连接
// POST /api/v1/rag/config/llm/test
func (h *RAGHandler) TestLLMConnection(c *gin.Context) {
	var req TestLLMConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	// API Key 脱敏
	apiKeyMasked := ""
	if len(req.APIKey) > 8 {
		apiKeyMasked = req.APIKey[:4] + "..." + req.APIKey[len(req.APIKey)-4:]
	} else {
		apiKeyMasked = "***"
	}

	h.logger.Debug("Testing LLM connection",
		"url", req.URL,
		"model", req.Model,
		"api_key", apiKeyMasked,
	)

	// 创建 LLM 客户端
	llmClient := appRAG.NewLLMClient(req.URL, req.APIKey, req.Model)

	// 测试连接
	if err := llmClient.TestConnection(); err != nil {
		h.logger.Error("LLM connection test failed",
			"error", err,
			"url", req.URL,
			"model", req.Model,
		)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// TriggerFullIndex 触发全量建索引
// POST /api/v1/rag/index/full
func (h *RAGHandler) TriggerFullIndex(c *gin.Context) {
	ragService, _, scanScheduler, err := h.getServices()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 触发全量扫描
	h.logger.Info("Triggering full RAG index")
	go func() {
		scanScheduler.TriggerFullScan(ragService)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Full index started"})
}

// ClearAllDataRequest 清空数据请求
type ClearAllDataRequest struct {
	Confirm bool `json:"confirm"` // 确认标志
}

// ClearAllData 清空所有 RAG 数据
// DELETE /api/v1/rag/data
func (h *RAGHandler) ClearAllData(c *gin.Context) {
	var req ClearAllDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !req.Confirm {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Please set confirm=true to clear all data",
		})
		return
	}

	h.logger.Info("Clearing all RAG data")

	// 获取 Qdrant 管理器
	qdrantManager := h.ragInitializer.GetQdrantManager()
	if qdrantManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Qdrant manager not initialized"})
		return
	}

	// 1. 删除 Qdrant Collection 中的所有点
	if err := qdrantManager.ClearCollections(); err != nil {
		h.logger.Error("Failed to clear Qdrant collections",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2. 清空元数据表
	_, _, scanScheduler, _, err := h.ragInitializer.InitializeServices()
	if err != nil {
		h.logger.Error("Failed to initialize services for clearing metadata",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if scanScheduler != nil {
		if err := scanScheduler.ClearMetadata(); err != nil {
			h.logger.Error("Failed to clear metadata",
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	h.logger.Info("All RAG data cleared successfully")
	c.JSON(http.StatusOK, gin.H{"message": "All data cleared"})
}
