package wire

import (
	appCursor "github.com/cocursor/backend/internal/application/cursor"
	"github.com/cocursor/backend/internal/infrastructure/websocket"
	"github.com/cocursor/backend/internal/interfaces"
)

// App 应用主结构，组合所有服务
type App struct {
	HTTPServer     *interfaces.HTTPServer
	MCPServer      *interfaces.MCPServer
	wsHub          *websocket.Hub
	projectManager *appCursor.ProjectManager
}

// NewApp 创建应用实例
func NewApp(
	httpServer *interfaces.HTTPServer,
	mcpServer *interfaces.MCPServer,
	wsHub *websocket.Hub,
	projectManager *appCursor.ProjectManager,
) *App {
	return &App{
		HTTPServer:     httpServer,
		MCPServer:      mcpServer,
		wsHub:          wsHub,
		projectManager: projectManager,
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

	// 启动 WebSocket Hub
	a.wsHub.Start()

	// 启动 HTTP 服务器（goroutine）
	go func() {
		if err := a.HTTPServer.Start(); err != nil {
			// TODO: 使用日志记录错误
		}
	}()

	// MCP 服务器通过 HTTP Handler 提供服务，不需要单独启动
	// 已在 HTTP 服务器中注册 /mcp/sse 端点

	return nil
}

// Stop 停止所有服务
func (a *App) Stop() error {
	if err := a.HTTPServer.Stop(); err != nil {
		return err
	}
	if err := a.MCPServer.Stop(); err != nil {
		return err
	}
	return nil
}
