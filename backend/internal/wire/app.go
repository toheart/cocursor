package wire

import (
	"database/sql"
	"time"

	"log/slog"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	appLifecycle "github.com/cocursor/backend/internal/application/lifecycle"
	appRAG "github.com/cocursor/backend/internal/application/rag"
	"github.com/cocursor/backend/internal/domain/events"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	applog "github.com/cocursor/backend/internal/infrastructure/log"
	infraMarketplace "github.com/cocursor/backend/internal/infrastructure/marketplace"
	"github.com/cocursor/backend/internal/infrastructure/watcher"
	"github.com/cocursor/backend/internal/infrastructure/websocket"
	"github.com/cocursor/backend/internal/interfaces"
)

// App 应用主结构，组合所有服务
type App struct {
	HTTPServer            *interfaces.HTTPServer
	MCPServer             *interfaces.MCPServer
	wsHub                 *websocket.Hub
	projectManager        *appCursor.ProjectManager
	workspaceCacheService *appCursor.WorkspaceCacheService
	scanScheduler         *appRAG.ScanScheduler
	ragInitializer        *appRAG.RAGInitializer // 用于管理 Qdrant 生命周期
	mcpInitializer        *infraMarketplace.MCPInitializer
	windowManager         *appLifecycle.WindowManager // 窗口生命周期管理
	db                    *sql.DB
	logger                *slog.Logger

	// 文件监听相关
	eventBus    events.EventBus
	fileWatcher *watcher.FileWatcher
}

// NewApp 创建应用实例
func NewApp(
	httpServer *interfaces.HTTPServer,
	mcpServer *interfaces.MCPServer,
	wsHub *websocket.Hub,
	projectManager *appCursor.ProjectManager,
	workspaceCacheService *appCursor.WorkspaceCacheService,
	workAnalysisService *appCursor.WorkAnalysisService,
	scanScheduler *appRAG.ScanScheduler,
	ragInitializer *appRAG.RAGInitializer,
	mcpInitializer *infraMarketplace.MCPInitializer,
	windowManager *appLifecycle.WindowManager,
	db *sql.DB,
) *App {
	logger := applog.NewModuleLogger("app", "main")

	// 设置 WorkspaceCacheService 到 WorkAnalysisService（解决循环依赖）
	if workAnalysisService != nil && workspaceCacheService != nil {
		workAnalysisService.SetWorkspaceCacheService(workspaceCacheService)
	}

	// 初始化事件总线
	eventBus := watcher.NewEventBus()

	// 初始化文件监听器
	pathResolver := infraCursor.NewPathResolver()
	workspaceDir, _ := pathResolver.GetWorkspaceStorageDir()

	config := watcher.DefaultWatchConfig()
	config.WorkspaceDir = workspaceDir

	fileWatcher, err := watcher.NewFileWatcher(config, eventBus)
	if err != nil {
		logger.Error("Failed to create file watcher", "error", err)
	}

	return &App{
		HTTPServer:            httpServer,
		MCPServer:             mcpServer,
		wsHub:                 wsHub,
		projectManager:        projectManager,
		workspaceCacheService: workspaceCacheService,
		scanScheduler:         scanScheduler,
		ragInitializer:        ragInitializer,
		mcpInitializer:        mcpInitializer,
		windowManager:         windowManager,
		db:                    db,
		logger:                logger,
		eventBus:              eventBus,
		fileWatcher:           fileWatcher,
	}
}

// Start 启动所有服务
func (a *App) Start() error {
	a.logger.Info("Starting CoCursor backend application")

	// 启动窗口生命周期管理器
	if a.windowManager != nil {
		a.windowManager.Start()
		a.logger.Info("Window manager started")
	}

	// 初始化项目管理器（扫描所有工作区）
	if a.projectManager != nil {
		if err := a.projectManager.Start(); err != nil {
			a.logger.Error("Failed to start project manager",
				"error", err,
			)
		}
	}

	// 启动工作区缓存服务（依赖 ProjectManager 已启动）
	if a.workspaceCacheService != nil {
		if err := a.workspaceCacheService.Start(); err != nil {
			a.logger.Error("Failed to start workspace cache service",
				"error", err,
			)
		}
	}

	// 启动 RAG 扫描调度器（如果已配置）
	// 注意：scanScheduler 可能为 nil（如果 RAG 未配置）
	if a.scanScheduler != nil {
		if err := a.scanScheduler.Start(); err != nil {
			a.logger.Error("Failed to start RAG scan scheduler",
				"error", err,
			)
		}
	}

	// 注册事件订阅者并启动文件监听
	a.setupEventSubscribers()
	if a.fileWatcher != nil {
		if err := a.fileWatcher.Start(); err != nil {
			a.logger.Error("Failed to start file watcher",
				"error", err,
			)
		} else {
			a.logger.Info("File watcher started successfully")
		}
	}

	// 启动 WebSocket Hub
	a.wsHub.Start()

	// 启动 HTTP 服务器（goroutine）
	go func() {
		if err := a.HTTPServer.Start(); err != nil {
			a.logger.Error("Failed to start HTTP server",
				"error", err,
			)
		}
	}()

	// 初始化默认 MCP 服务器配置（在 HTTP 服务器启动后）
	// 等待一小段时间确保服务器已启动
	go func() {
		// 等待 500ms 确保 HTTP 服务器已启动
		time.Sleep(500 * time.Millisecond)
		if a.mcpInitializer != nil {
			if err := a.mcpInitializer.InitializeDefaultMCP(); err != nil {
				a.logger.Error("Failed to initialize default MCP server",
					"error", err,
				)
			}
		}
	}()

	a.logger.Info("CoCursor backend application started successfully")

	// MCP 服务器通过 HTTP Handler 提供服务，不需要单独启动
	// 已在 HTTP 服务器中注册 /mcp/sse 端点

	return nil
}

// setupEventSubscribers 注册事件订阅者
func (a *App) setupEventSubscribers() {
	if a.eventBus == nil {
		return
	}

	// 注册 RAG Scanner 订阅会话文件事件
	if a.scanScheduler != nil {
		a.eventBus.SubscribeMultiple(
			[]events.EventType{
				events.SessionFileCreated,
				events.SessionFileModified,
			},
			events.HandlerFunc(func(event events.Event) error {
				sessionEvent, ok := event.(*events.SessionFileEvent)
				if !ok {
					return nil
				}
				return a.scanScheduler.HandleSessionFileEvent(
					sessionEvent.SessionID,
					sessionEvent.ProjectKey,
					sessionEvent.FilePath,
				)
			}),
		)
		a.logger.Info("RAG Scanner subscribed to session file events")
	}

	// 注册 WorkspaceCacheService 订阅工作区事件
	if a.workspaceCacheService != nil {
		a.eventBus.Subscribe(
			events.WorkspaceCreated,
			events.HandlerFunc(func(event events.Event) error {
				wsEvent, ok := event.(*events.WorkspaceEvent)
				if !ok {
					return nil
				}
				return a.workspaceCacheService.HandleWorkspaceEvent(
					wsEvent.WorkspaceID,
					wsEvent.ProjectPath,
				)
			}),
		)
		a.logger.Info("WorkspaceCacheService subscribed to workspace events")
	}
}

// GetShutdownChan 获取关闭信号通道（用于 main 函数监听 API 关闭请求）
func (a *App) GetShutdownChan() <-chan struct{} {
	return a.HTTPServer.GetShutdownChan()
}

// Stop 停止所有服务
func (a *App) Stop() error {
	a.logger.Info("Stopping CoCursor backend application")

	// 停止窗口生命周期管理器
	if a.windowManager != nil {
		a.windowManager.Stop()
		a.logger.Info("Window manager stopped")
	}

	// 停止文件监听器
	if a.fileWatcher != nil {
		a.fileWatcher.Stop()
		a.logger.Info("File watcher stopped")
	}

	// 关闭事件总线
	if a.eventBus != nil {
		a.eventBus.Close()
		a.logger.Info("Event bus closed")
	}

	// 停止 RAG 扫描调度器
	if a.scanScheduler != nil {
		if err := a.scanScheduler.Stop(); err != nil {
			a.logger.Error("Failed to stop RAG scan scheduler",
				"error", err,
			)
		}
	}

	// 停止 Qdrant 服务（如果已启动）
	if a.ragInitializer != nil {
		if err := a.ragInitializer.StopQdrant(); err != nil {
			a.logger.Error("Failed to stop Qdrant",
				"error", err,
			)
		}
	}

	// 停止工作区缓存服务
	if a.workspaceCacheService != nil {
		if err := a.workspaceCacheService.Stop(); err != nil {
			a.logger.Error("Failed to stop workspace cache service",
				"error", err,
			)
		}
	}

	if err := a.HTTPServer.Stop(); err != nil {
		a.logger.Error("Failed to stop HTTP server",
			"error", err,
		)
		return err
	}
	if err := a.MCPServer.Stop(); err != nil {
		a.logger.Error("Failed to stop MCP server",
			"error", err,
		)
		return err
	}

	// 关闭数据库连接
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error("Failed to close database connection",
				"error", err,
			)
			return err
		}
	}

	a.logger.Info("CoCursor backend application stopped successfully")

	return nil
}
