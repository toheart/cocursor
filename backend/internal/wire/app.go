package wire

import (
	"database/sql"
	"time"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	appRAG "github.com/cocursor/backend/internal/application/rag"
	infraMarketplace "github.com/cocursor/backend/internal/infrastructure/marketplace"
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
	mcpInitializer        *infraMarketplace.MCPInitializer
	db                    *sql.DB
}

// NewApp 创建应用实例
func NewApp(
	httpServer *interfaces.HTTPServer,
	mcpServer *interfaces.MCPServer,
	wsHub *websocket.Hub,
	projectManager *appCursor.ProjectManager,
	workspaceCacheService *appCursor.WorkspaceCacheService,
	scanScheduler *appRAG.ScanScheduler,
	mcpInitializer *infraMarketplace.MCPInitializer,
	db *sql.DB,
) *App {
	return &App{
		HTTPServer:            httpServer,
		MCPServer:             mcpServer,
		wsHub:                 wsHub,
		projectManager:        projectManager,
		workspaceCacheService: workspaceCacheService,
		scanScheduler:         scanScheduler,
		mcpInitializer:        mcpInitializer,
		db:                    db,
	}
}

// Start 启动所有服务
func (a *App) Start() error {
	// 初始化项目管理器（扫描所有工作区）
	if a.projectManager != nil {
		if err := a.projectManager.Start(); err != nil {
			// 记录错误但不阻止启动
			// TODO: 使用日志记录错误
		}
	}

	// 启动工作区缓存服务（依赖 ProjectManager 已启动）
	if a.workspaceCacheService != nil {
		if err := a.workspaceCacheService.Start(); err != nil {
			// 记录错误但不阻止启动
			// TODO: 使用日志记录错误
		}
	}

	// 启动 RAG 扫描调度器（如果已配置）
	// 注意：scanScheduler 可能为 nil（如果 RAG 未配置）
	if a.scanScheduler != nil {
		if err := a.scanScheduler.Start(); err != nil {
			// 记录错误但不阻止启动
			// TODO: 使用日志记录错误
		}
	}

	// 启动 WebSocket Hub
	a.wsHub.Start()

	// 启动 HTTP 服务器（goroutine）
	go func() {
		if err := a.HTTPServer.Start(); err != nil {
			// TODO: 使用日志记录错误
		}
	}()

	// 初始化默认 MCP 服务器配置（在 HTTP 服务器启动后）
	// 等待一小段时间确保服务器已启动
	go func() {
		// 等待 500ms 确保 HTTP 服务器已启动
		time.Sleep(500 * time.Millisecond)
		if a.mcpInitializer != nil {
			if err := a.mcpInitializer.InitializeDefaultMCP(); err != nil {
				// 记录错误但不阻止启动
				// TODO: 使用日志记录错误
			}
		}
	}()

	// MCP 服务器通过 HTTP Handler 提供服务，不需要单独启动
	// 已在 HTTP 服务器中注册 /mcp/sse 端点

	return nil
}

// Stop 停止所有服务
func (a *App) Stop() error {
	// 停止 RAG 扫描调度器
	if a.scanScheduler != nil {
		if err := a.scanScheduler.Stop(); err != nil {
			// 记录错误但不阻止停止
		}
	}

	// 停止工作区缓存服务
	if a.workspaceCacheService != nil {
		if err := a.workspaceCacheService.Stop(); err != nil {
			// 记录错误但不阻止停止
		}
	}

	if err := a.HTTPServer.Stop(); err != nil {
		return err
	}
	if err := a.MCPServer.Stop(); err != nil {
		return err
	}

	// 关闭数据库连接
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			return err
		}
	}

	return nil
}
