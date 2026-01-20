package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"log/slog"

	appRAG "github.com/cocursor/backend/internal/application/rag"
	"github.com/cocursor/backend/internal/infrastructure/embedding"
	"github.com/cocursor/backend/internal/infrastructure/log"
	infraRAG "github.com/cocursor/backend/internal/infrastructure/rag"
	"github.com/cocursor/backend/internal/infrastructure/vector"
	"github.com/gin-gonic/gin"
	"github.com/qdrant/go-client/qdrant"
)

// QdrantDownloadStatus Qdrant 下载状态
type QdrantDownloadStatus struct {
	Status      string    `json:"status"`                 // idle, downloading, success, failed
	Message     string    `json:"message"`                // 状态消息
	Version     string    `json:"version"`                // 下载的版本
	InstallPath string    `json:"install_path,omitempty"` // 安装路径（成功时）
	Error       string    `json:"error,omitempty"`        // 错误信息（失败时）
	StartedAt   time.Time `json:"started_at,omitempty"`   // 开始时间
	UpdatedAt   time.Time `json:"updated_at,omitempty"`   // 更新时间
	// 进度信息
	Downloaded int64   `json:"downloaded,omitempty"` // 已下载字节数
	TotalSize  int64   `json:"total_size,omitempty"` // 总字节数
	Percent    float64 `json:"percent,omitempty"`    // 下载百分比 (0-100)
}

// qdrantDownloadState 全局下载状态管理
var (
	qdrantDownloadState = &QdrantDownloadStatus{Status: "idle"}
	qdrantDownloadMutex sync.RWMutex
)

// getQdrantDownloadStatus 获取下载状态
func getQdrantDownloadStatus() QdrantDownloadStatus {
	qdrantDownloadMutex.RLock()
	defer qdrantDownloadMutex.RUnlock()
	return *qdrantDownloadState
}

// setQdrantDownloadStatus 设置下载状态
func setQdrantDownloadStatus(status QdrantDownloadStatus) {
	qdrantDownloadMutex.Lock()
	defer qdrantDownloadMutex.Unlock()
	status.UpdatedAt = time.Now()
	*qdrantDownloadState = status
}

// RAGHandler RAG 处理器
type RAGHandler struct {
	ragInitializer    *appRAG.RAGInitializer
	configManager     *infraRAG.ConfigManager
	scanScheduler     *appRAG.ScanScheduler
	enrichmentService *appRAG.EnrichmentService // 新增：增强服务
	chunkService      *appRAG.ChunkService      // 新增：知识片段服务
	logger            *slog.Logger
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

// SetEnrichmentService 设置增强服务
func (h *RAGHandler) SetEnrichmentService(svc *appRAG.EnrichmentService) {
	h.enrichmentService = svc
}

// SetChunkService 设置知识片段服务
func (h *RAGHandler) SetChunkService(svc *appRAG.ChunkService) {
	h.chunkService = svc
}

// getServices 获取 RAG 服务（如果已初始化）
func (h *RAGHandler) getServices() (*appRAG.ChunkService, *appRAG.SearchService, *appRAG.ScanScheduler, error) {
	chunkService, searchService, scanScheduler, _, _, err := h.ragInitializer.InitializeServices()
	if err != nil {
		return nil, nil, nil, err
	}
	if chunkService == nil {
		return nil, nil, nil, fmt.Errorf("RAG services not initialized. Please configure RAG first.")
	}
	return chunkService, searchService, scanScheduler, nil
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

// Index 处理索引请求（已弃用，请使用 /index/full）
// POST /api/v1/rag/index
func (h *RAGHandler) Index(c *gin.Context) {
	var req IndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chunkService, _, _, err := h.getServices()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 统一使用全量索引
	err = h.scanScheduler.TriggerFullScan(chunkService, 10, 3, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.SessionID != "" {
		c.JSON(http.StatusOK, gin.H{
			"message":    "Full index triggered (single session indexing deprecated)",
			"session_id": req.SessionID,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message": "Full index triggered",
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

	// 尝试从 Qdrant 获取实际索引数量
	totalIndexed := 0
	if h.ragInitializer != nil {
		qdrantManager := h.ragInitializer.GetQdrantManager()
		if qdrantManager != nil {
			count, err := qdrantManager.GetCollectionPointsCount("cursor_knowledge")
			if err == nil {
				totalIndexed = int(count)
			} else {
				h.logger.Debug("Failed to get collection points count, using config value",
					"error", err,
				)
				// 回退到配置文件中的值
				totalIndexed = config.TotalIndexed
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_indexed":  totalIndexed,
		"last_full_scan": config.LastFullScan,
		"index_config": gin.H{
			"batch_size":  config.IndexConfig.BatchSize,
			"concurrency": config.IndexConfig.Concurrency,
		},
	})
}

// ProjectInfo 项目信息
type ProjectInfo struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	ChunkCount  int    `json:"chunk_count"`
}

// GetIndexedProjects 获取已索引的项目列表
// GET /api/v1/rag/projects
func (h *RAGHandler) GetIndexedProjects(c *gin.Context) {
	qdrantManager := h.ragInitializer.GetQdrantManager()
	if qdrantManager == nil {
		c.JSON(http.StatusOK, gin.H{
			"projects": []ProjectInfo{},
			"total":    0,
		})
		return
	}

	client := qdrantManager.GetClient()
	if client == nil {
		c.JSON(http.StatusOK, gin.H{
			"projects": []ProjectInfo{},
			"total":    0,
		})
		return
	}

	// 使用 scroll 获取所有点的项目信息
	ctx := context.Background()
	projectCounts := make(map[string]*ProjectInfo)

	// 分页获取所有点
	var offsetID *qdrant.PointId
	limit := uint32(100)

	for {
		scrollResp, err := client.Scroll(ctx, &qdrant.ScrollPoints{
			CollectionName: "cursor_knowledge",
			Offset:         offsetID,
			Limit:          &limit,
			WithPayload:    qdrant.NewWithPayload(true),
		})
		if err != nil {
			h.logger.Error("Failed to scroll points", "error", err)
			break
		}

		for _, point := range scrollResp {
			payload := point.GetPayload()
			if payload == nil {
				continue
			}

			projectID := ""
			projectName := ""

			if val, ok := payload["project_id"]; ok {
				projectID = val.GetStringValue()
			}
			if val, ok := payload["project_name"]; ok {
				projectName = val.GetStringValue()
			}

			if projectID == "" {
				continue
			}

			if _, exists := projectCounts[projectID]; !exists {
				projectCounts[projectID] = &ProjectInfo{
					ProjectID:   projectID,
					ProjectName: projectName,
					ChunkCount:  0,
				}
			}
			projectCounts[projectID].ChunkCount++
		}

		// 检查是否还有更多数据
		if len(scrollResp) < int(limit) {
			break
		}

		// 获取最后一个点的 ID 作为下一次的 offset
		if len(scrollResp) > 0 {
			lastPoint := scrollResp[len(scrollResp)-1]
			offsetID = lastPoint.GetId()
		}
	}

	// 转换为列表
	projects := make([]ProjectInfo, 0, len(projectCounts))
	for _, info := range projectCounts {
		projects = append(projects, *info)
	}

	c.JSON(http.StatusOK, gin.H{
		"projects": projects,
		"total":    len(projects),
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

	// 隐藏 API Key，但返回已配置状态
	response := gin.H{
		"embedding_api": gin.H{
			"url":         config.EmbeddingAPI.URL,
			"model":       config.EmbeddingAPI.Model,
			"has_api_key": config.EmbeddingAPI.APIKey != "",
		},
		"llm_chat_api": gin.H{
			"url":         config.LLMChatAPI.URL,
			"model":       config.LLMChatAPI.Model,
			"has_api_key": config.LLMChatAPI.APIKey != "",
		},
		"qdrant": gin.H{
			"version":     config.Qdrant.Version,
			"binary_path": config.Qdrant.BinaryPath,
			"data_path":   config.Qdrant.DataPath,
		},
		"index_config": gin.H{
			"batch_size":  config.IndexConfig.BatchSize,
			"concurrency": config.IndexConfig.Concurrency,
		},
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
	IndexConfig struct {
		BatchSize   int `json:"batch_size"`
		Concurrency int `json:"concurrency"`
	} `json:"index_config"`
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

	// 更新索引配置
	if req.IndexConfig.BatchSize > 0 {
		config.IndexConfig.BatchSize = req.IndexConfig.BatchSize
	}
	if req.IndexConfig.Concurrency > 0 {
		config.IndexConfig.Concurrency = req.IndexConfig.Concurrency
	}

	if err := h.configManager.WriteConfig(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新初始化 RAG 服务以应用新配置
	// 1. 重置初始化状态（清理旧的服务实例）
	h.ragInitializer.Reset()

	// 2. 重新初始化服务（使用新配置）
	_, _, _, _, _, err = h.ragInitializer.InitializeServices()
	if err != nil {
		h.logger.Error("Failed to reinitialize RAG services",
			"error", err,
			"message", "Config saved but service reinitialization failed",
		)
	}

	// 3. 更新 ScanScheduler 配置
	if h.scanScheduler != nil {
		scanConfig := &appRAG.ScanConfig{
			BatchSize:   config.IndexConfig.BatchSize,
			Concurrency: config.IndexConfig.Concurrency,
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

// DownloadQdrant 异步下载并安装 Qdrant
// POST /api/v1/rag/qdrant/download
// 该接口立即返回，下载在后台进行，前端通过 GET /api/v1/rag/qdrant/status 轮询状态
func (h *RAGHandler) DownloadQdrant(c *gin.Context) {
	var req DownloadQdrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体，使用默认版本
		req.Version = ""
	}

	// 获取版本号
	version := req.Version
	if version == "" {
		version = "v1.16.3"
	}

	// 检查是否已经在下载中
	currentStatus := getQdrantDownloadStatus()
	if currentStatus.Status == "downloading" {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error":   "download already in progress",
			"status":  currentStatus,
		})
		return
	}

	// 记录请求日志
	h.logger.Info("Starting async Qdrant download",
		"version", version,
	)

	// 设置下载状态为"下载中"
	setQdrantDownloadStatus(QdrantDownloadStatus{
		Status:    "downloading",
		Message:   "Starting download...",
		Version:   version,
		StartedAt: time.Now(),
	})

	// 启动异步下载任务
	go h.asyncDownloadQdrant(version)

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"message": "Download started, poll /api/v1/rag/qdrant/status for progress",
		"version": version,
	})
}

// asyncDownloadQdrant 异步执行 Qdrant 下载任务
func (h *RAGHandler) asyncDownloadQdrant(version string) {
	// 创建下载选项，包含进度回调
	opts := vector.QdrantDownloadOptions{
		Version: version,
		OnProgress: func(downloaded, total int64) {
			// 更新下载进度状态
			var message string
			var percent float64
			if total > 0 {
				percent = float64(downloaded) / float64(total) * 100
				message = fmt.Sprintf("Downloading... %.1f%% (%d/%d MB)",
					percent,
					downloaded/(1024*1024),
					total/(1024*1024))
			} else {
				message = fmt.Sprintf("Downloading... %d MB downloaded",
					downloaded/(1024*1024))
			}
			setQdrantDownloadStatus(QdrantDownloadStatus{
				Status:     "downloading",
				Message:    message,
				Version:    version,
				Downloaded: downloaded,
				TotalSize:  total,
				Percent:    percent,
			})
		},
	}

	// 使用带 Context 的下载方法（支持取消和进度报告）
	// 目前使用 Background context，后续可以添加取消支持
	installPath, err := vector.DownloadQdrantWithContext(context.Background(), opts)
	if err != nil {
		h.logger.Error("Failed to download Qdrant",
			"version", version,
			"error", err,
		)
		setQdrantDownloadStatus(QdrantDownloadStatus{
			Status:  "failed",
			Message: "Download failed",
			Version: version,
			Error:   err.Error(),
		})
		return
	}

	// 更新配置
	config, err := h.configManager.ReadConfig()
	if err != nil {
		h.logger.Error("Failed to read config after download",
			"error", err,
		)
		setQdrantDownloadStatus(QdrantDownloadStatus{
			Status:      "failed",
			Message:     "Download succeeded but failed to update config",
			Version:     version,
			InstallPath: installPath,
			Error:       fmt.Sprintf("failed to read config: %v", err),
		})
		return
	}

	// 更新配置中的 Qdrant 信息
	config.Qdrant.BinaryPath = installPath
	dataPath, err := vector.GetQdrantDataPath()
	if err == nil {
		config.Qdrant.DataPath = dataPath
	}
	config.Qdrant.Version = version

	if err := h.configManager.WriteConfig(config); err != nil {
		h.logger.Error("Failed to write config after download",
			"error", err,
		)
		setQdrantDownloadStatus(QdrantDownloadStatus{
			Status:      "failed",
			Message:     "Download succeeded but failed to save config",
			Version:     version,
			InstallPath: installPath,
			Error:       fmt.Sprintf("failed to update config: %v", err),
		})
		return
	}

	// 下载成功
	h.logger.Info("Qdrant download completed successfully",
		"version", version,
		"install_path", installPath,
	)
	setQdrantDownloadStatus(QdrantDownloadStatus{
		Status:      "success",
		Message:     "Qdrant downloaded and installed successfully",
		Version:     version,
		InstallPath: installPath,
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

	// 获取下载状态
	downloadStatus := getQdrantDownloadStatus()

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"binary_exists":   binaryExists,
		"binary_path":     binaryPath,
		"data_path":       dataPath,
		"is_running":      isRunning,
		"grpc_port":       6334,
		"http_port":       6333,
		"download_status": downloadStatus.Status, // idle, downloading, success, failed
		"download_info":   downloadStatus,        // 完整下载状态信息
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

// TriggerFullIndexRequest 触发全量索引请求
type TriggerFullIndexRequest struct {
	BatchSize   int `json:"batch_size"`  // 批量大小（可选，默认 10）
	Concurrency int `json:"concurrency"` // 并发数（可选，默认 3）
}

// TriggerFullIndex 触发全量建索引
// POST /api/v1/rag/index/full
func (h *RAGHandler) TriggerFullIndex(c *gin.Context) {
	var req TriggerFullIndexRequest
	// 允许空请求体
	_ = c.ShouldBindJSON(&req)

	// 检查是否已在运行
	if h.scanScheduler != nil && h.scanScheduler.IsRunning() {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Full index is already running",
			"running": true,
		})
		return
	}

	chunkService, _, _, err := h.getServices()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认值
	if req.BatchSize <= 0 {
		req.BatchSize = 10
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 3
	}

	// 触发全量扫描
	h.logger.Info("Triggering full RAG index",
		"batch_size", req.BatchSize,
		"concurrency", req.Concurrency,
	)

	// 创建进度回调，在索引完成后更新配置
	progressCallback := func(progress *appRAG.IndexProgress) {
		if progress.Status == "completed" {
			// 更新配置文件中的索引统计
			config, err := h.configManager.ReadConfig()
			if err != nil {
				h.logger.Error("Failed to read config for updating index stats", "error", err)
				return
			}
			config.TotalIndexed = progress.IndexedMessages
			config.LastFullScan = progress.StartTime.Unix()
			if err := h.configManager.WriteConfig(config); err != nil {
				h.logger.Error("Failed to update config with index stats", "error", err)
			} else {
				h.logger.Info("Updated index stats in config",
					"total_indexed", progress.IndexedMessages,
					"last_full_scan", progress.StartTime.Unix(),
				)
			}
		}
	}

	err = h.scanScheduler.TriggerFullScan(chunkService, req.BatchSize, req.Concurrency, progressCallback)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Full index started",
		"batch_size":  req.BatchSize,
		"concurrency": req.Concurrency,
	})
}

// GetIndexProgress 获取索引进度
// GET /api/v1/rag/index/progress
func (h *RAGHandler) GetIndexProgress(c *gin.Context) {
	if h.scanScheduler == nil {
		c.JSON(http.StatusOK, gin.H{
			"running":  false,
			"progress": nil,
		})
		return
	}

	progress := h.scanScheduler.GetProgress()
	if progress == nil {
		c.JSON(http.StatusOK, gin.H{
			"running":  false,
			"progress": nil,
		})
		return
	}

	// 计算百分比和耗时
	percentage := 0
	if progress.TotalFiles > 0 {
		percentage = progress.ProcessedFiles * 100 / progress.TotalFiles
	}

	c.JSON(http.StatusOK, gin.H{
		"running": progress.Status == "running",
		"progress": gin.H{
			"status":           progress.Status,
			"total_files":      progress.TotalFiles,
			"processed_files":  progress.ProcessedFiles,
			"indexed_messages": progress.IndexedMessages,
			"percentage":       percentage,
			"elapsed_seconds":  int(time.Since(progress.StartTime).Seconds()),
			"error_message":    progress.ErrorMessage,
		},
	})
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
	_, _, scanScheduler, _, _, err := h.ragInitializer.InitializeServices()
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

// SearchChunks 使用新的知识片段模型搜索
// POST /api/v1/rag/search/chunks
func (h *RAGHandler) SearchChunks(c *gin.Context) {
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

	results, err := searchService.SearchChunks(context.Background(), &appRAG.SearchRequest{
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

// GetChunkDetail 获取知识片段详情
// GET /api/v1/rag/chunks/:id
func (h *RAGHandler) GetChunkDetail(c *gin.Context) {
	chunkID := c.Param("id")
	if chunkID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chunk_id is required"})
		return
	}

	_, searchService, _, err := h.getServices()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	detail, err := searchService.GetChunkDetail(chunkID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chunk not found"})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// GetEnrichmentStats 获取增强队列统计
// GET /api/v1/rag/enrichment/stats
func (h *RAGHandler) GetEnrichmentStats(c *gin.Context) {
	if h.enrichmentService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "enrichment service not configured"})
		return
	}

	stats, err := h.enrichmentService.GetQueueStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats":      stats,
		"is_running": h.enrichmentService.IsRunning(),
	})
}

// RetryEnrichment 重试失败的增强任务
// POST /api/v1/rag/enrichment/retry
func (h *RAGHandler) RetryEnrichment(c *gin.Context) {
	if h.enrichmentService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "enrichment service not configured"})
		return
	}

	count, err := h.enrichmentService.RetryFailed()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Failed tasks reset",
		"reset_count": count,
	})
}

// GetIndexStats 获取索引统计（新模型）
// GET /api/v1/rag/index/stats
func (h *RAGHandler) GetIndexStats(c *gin.Context) {
	if h.chunkService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chunk service not configured"})
		return
	}

	stats, err := h.chunkService.GetIndexStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// UploadQdrant 上传 Qdrant 安装包进行本地安装
// POST /api/v1/rag/qdrant/upload
// 当网络下载失败时，用户可上传预先下载的 Qdrant 安装包（tar.gz/zip）
func (h *RAGHandler) UploadQdrant(c *gin.Context) {
	// 1. 接收 multipart 文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.logger.Error("Failed to receive upload file",
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "failed to receive file: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// 验证文件扩展名
	filename := header.Filename
	if !isValidArchiveFile(filename) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "unsupported file format, only .tar.gz and .zip are supported",
		})
		return
	}

	h.logger.Info("Receiving Qdrant upload",
		"filename", filename,
		"size", header.Size,
	)

	// 2. 保存到临时目录
	tmpDir, err := os.MkdirTemp("", "qdrant-upload-*")
	if err != nil {
		h.logger.Error("Failed to create temp directory",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to create temp directory",
		})
		return
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			h.logger.Warn("Failed to remove temp directory",
				"path", tmpDir,
				"error", err,
			)
		}
	}()

	// 保存上传的文件
	archivePath := tmpDir + "/" + filename
	outFile, err := os.Create(archivePath)
	if err != nil {
		h.logger.Error("Failed to create archive file",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to save uploaded file",
		})
		return
	}

	written, err := outFile.ReadFrom(file)
	outFile.Close()
	if err != nil {
		h.logger.Error("Failed to write archive file",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to write uploaded file",
		})
		return
	}

	h.logger.Info("Upload file saved",
		"path", archivePath,
		"size", written,
	)

	// 3. 调用 extractor 解压
	extractor := vector.NewArchiveExtractor()
	extractDir := tmpDir + "/extracted"
	if err := extractor.Extract(archivePath, extractDir); err != nil {
		h.logger.Error("Failed to extract archive",
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "failed to extract archive: " + err.Error(),
		})
		return
	}

	// 4. 查找并验证二进制文件
	osName, _ := vector.GetPlatformInfo()
	binaryName := "qdrant"
	if osName == "windows" {
		binaryName = "qdrant.exe"
	}

	binaryPath, err := extractor.FindBinary(extractDir, binaryName)
	if err != nil {
		h.logger.Error("Failed to find Qdrant binary in archive",
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "qdrant binary not found in archive",
		})
		return
	}

	h.logger.Info("Binary found",
		"path", binaryPath,
	)

	// 5. 复制到安装目录
	installPath, err := vector.GetQdrantInstallPath()
	if err != nil {
		h.logger.Error("Failed to get install path",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to get install path",
		})
		return
	}

	// 确保安装目录存在
	installDir := installPath[:len(installPath)-len(binaryName)-1]
	if err := os.MkdirAll(installDir, 0755); err != nil {
		h.logger.Error("Failed to create install directory",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to create install directory",
		})
		return
	}

	// 复制二进制文件
	if err := copyBinaryFile(binaryPath, installPath); err != nil {
		h.logger.Error("Failed to copy binary",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to install binary: " + err.Error(),
		})
		return
	}

	// 设置可执行权限（非 Windows）
	if osName != "windows" {
		if err := os.Chmod(installPath, 0755); err != nil {
			h.logger.Warn("Failed to set executable permission",
				"error", err,
			)
		}
	}

	h.logger.Info("Binary installed",
		"path", installPath,
	)

	// 6. 更新配置
	config, err := h.configManager.ReadConfig()
	if err != nil {
		h.logger.Warn("Failed to read config, will use defaults",
			"error", err,
		)
	} else {
		config.Qdrant.BinaryPath = installPath
		dataPath, err := vector.GetQdrantDataPath()
		if err == nil {
			config.Qdrant.DataPath = dataPath
		}
		// 版本未知，标记为 uploaded
		config.Qdrant.Version = "uploaded"

		if err := h.configManager.WriteConfig(config); err != nil {
			h.logger.Warn("Failed to update config",
				"error", err,
			)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Qdrant installed successfully from uploaded package",
		"install_path": installPath,
	})
}

// isValidArchiveFile 检查文件是否为有效的归档格式
func isValidArchiveFile(filename string) bool {
	lower := filename
	for i := 0; i < len(lower); i++ {
		if lower[i] >= 'A' && lower[i] <= 'Z' {
			lower = lower[:i] + string(lower[i]+32) + lower[i+1:]
		}
	}
	return len(lower) > 7 && lower[len(lower)-7:] == ".tar.gz" ||
		len(lower) > 4 && lower[len(lower)-4:] == ".tgz" ||
		len(lower) > 4 && lower[len(lower)-4:] == ".zip"
}

// copyBinaryFile 复制二进制文件
func copyBinaryFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := destFile.ReadFrom(sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
