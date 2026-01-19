package handler

import (
	"context"
	"fmt"
	"net/http"

	appRAG "github.com/cocursor/backend/internal/application/rag"
	"github.com/cocursor/backend/internal/infrastructure/embedding"
	infraRAG "github.com/cocursor/backend/internal/infrastructure/rag"
	"github.com/cocursor/backend/internal/infrastructure/vector"
	"github.com/gin-gonic/gin"
)

// RAGHandler RAG 处理器
type RAGHandler struct {
	ragInitializer *appRAG.RAGInitializer
	configManager  *infraRAG.ConfigManager
}

// NewRAGHandler 创建 RAG 处理器
func NewRAGHandler(
	ragInitializer *appRAG.RAGInitializer,
	configManager *infraRAG.ConfigManager,
) *RAGHandler {
	return &RAGHandler{
		ragInitializer: ragInitializer,
		configManager:  configManager,
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
	Limit      int      `json:"limit"`
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
	config.ScanConfig.Enabled = req.ScanConfig.Enabled
	config.ScanConfig.Interval = req.ScanConfig.Interval
	config.ScanConfig.BatchSize = req.ScanConfig.BatchSize
	config.ScanConfig.Concurrency = req.ScanConfig.Concurrency

	if err := h.configManager.WriteConfig(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 创建临时客户端测试连接
	embeddingClient := embedding.NewClient(req.URL, req.APIKey, req.Model)
	if err := embeddingClient.TestConnection(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

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

	// 下载 Qdrant
	installPath, err := vector.DownloadQdrant(req.Version)
	if err != nil {
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
	})
}
