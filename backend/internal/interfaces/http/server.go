package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cocursor/backend/internal/interfaces/http/handler"
	"github.com/cocursor/backend/internal/interfaces/mcp"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/cocursor/backend/docs" // Swagger docs
)

// HTTPServer HTTP 服务器
type HTTPServer struct {
	router   *gin.Engine
	httpPort string
	server   *http.Server
}

// NewServer 创建 HTTP 服务器
func NewServer(
	notificationHandler *handler.NotificationHandler,
	statsHandler *handler.StatsHandler,
	projectHandler *handler.ProjectHandler,
	analyticsHandler *handler.AnalyticsHandler,
	workspaceHandler *handler.WorkspaceHandler,
	mcpServer *mcp.MCPServer,
) *HTTPServer {
	router := gin.Default()

	// 注册路由
	api := router.Group("/api/v1")
	{
		api.POST("/notifications", notificationHandler.Create)
		api.GET("/stats/current-session", statsHandler.CurrentSession)
		api.GET("/stats/acceptance-rate", statsHandler.AcceptanceRate)
		api.GET("/stats/conversation-overview", statsHandler.ConversationOverview)
		api.GET("/stats/file-references", statsHandler.FileReferences)
		api.GET("/stats/daily-report", statsHandler.DailyReport)

		// 分析相关路由
		api.GET("/stats/token-usage", analyticsHandler.TokenUsage)
		api.GET("/stats/work-analysis", analyticsHandler.WorkAnalysis)
		api.GET("/sessions/list", analyticsHandler.SessionList)
		api.GET("/sessions/:sessionId/detail", analyticsHandler.SessionDetail)

		// 项目相关路由
		api.GET("/project/list", projectHandler.ListProjects)
		api.POST("/project/activate", projectHandler.ActivateProject)
		api.GET("/project/:project_name/prompts", projectHandler.GetProjectPrompts)
		api.GET("/project/:project_name/generations", projectHandler.GetProjectGenerations)
		api.GET("/project/:project_name/sessions", projectHandler.GetProjectSessions)
		api.GET("/project/:project_name/stats/acceptance", projectHandler.GetProjectAcceptanceStats)

		// 工作区相关路由
		api.POST("/workspace/register", workspaceHandler.Register)
		api.POST("/workspace/focus", workspaceHandler.Focus)
	}

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// MCP SSE 端点
	if mcpServer != nil {
		router.Any("/mcp/sse", gin.WrapH(mcpServer.GetHandler()))
	}

	return &HTTPServer{
		router:   router,
		httpPort: ":19960",
	}
}

// Start 启动服务器
func (s *HTTPServer) Start() error {
	s.server = &http.Server{
		Addr:    s.httpPort,
		Handler: s.router,
	}

	fmt.Printf("HTTP 服务器启动在端口 %s\n", s.httpPort)
	return s.server.ListenAndServe()
}

// Shutdown 优雅关闭
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// Stop 停止服务器
func (s *HTTPServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.Shutdown(ctx)
}
