package http

import (
	"context"
	"net/http"
	"time"

	"log/slog"

	appMarketplace "github.com/cocursor/backend/internal/application/marketplace"
	appTeam "github.com/cocursor/backend/internal/application/team"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/cocursor/backend/internal/interfaces/http/handler"
	"github.com/cocursor/backend/internal/interfaces/mcp"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/cocursor/backend/docs" // Swagger docs
)

// HTTPServer HTTP 服务器
type HTTPServer struct {
	router           *gin.Engine
	httpPort         string
	server           *http.Server
	logger           *slog.Logger
	teamComponents   *appTeam.TeamComponents
	pluginService    *appMarketplace.PluginService
	dailySummaryRepo storage.DailySummaryRepository
}

// NewServer 创建 HTTP 服务器
func NewServer(
	notificationHandler *handler.NotificationHandler,
	statsHandler *handler.StatsHandler,
	projectHandler *handler.ProjectHandler,
	analyticsHandler *handler.AnalyticsHandler,
	workspaceHandler *handler.WorkspaceHandler,
	marketplaceHandler *handler.MarketplaceHandler,
	workflowHandler *handler.WorkflowHandler,
	ragHandler *handler.RAGHandler,
	dailySummaryHandler *handler.DailySummaryHandler,
	mcpServer *mcp.MCPServer,
	pluginService *appMarketplace.PluginService,
	dailySummaryRepo storage.DailySummaryRepository,
) *HTTPServer {
	router := gin.Default()

	logger := log.NewModuleLogger("http", "server")

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

		// 插件市场相关路由
		marketplace := api.Group("/marketplace")
		{
			marketplace.GET("/plugins", marketplaceHandler.ListPlugins)
			marketplace.GET("/plugins/:id", marketplaceHandler.GetPlugin)
			marketplace.GET("/installed", marketplaceHandler.GetInstalledPlugins)
			marketplace.POST("/plugins/:id/install", marketplaceHandler.InstallPlugin)
			marketplace.POST("/plugins/:id/uninstall", marketplaceHandler.UninstallPlugin)
			marketplace.GET("/plugins/:id/status", marketplaceHandler.CheckPluginStatus)
		}

		// 工作流相关路由
		api.GET("/workflows", workflowHandler.ListWorkflows)
		api.GET("/workflows/status", workflowHandler.GetWorkflowStatus)
		api.GET("/workflows/:change_id", workflowHandler.GetWorkflowDetail)

		// 日报相关路由
		if dailySummaryHandler != nil {
			api.GET("/daily-summary", dailySummaryHandler.GetDailySummary)
			api.GET("/daily-summary/batch-status", dailySummaryHandler.GetBatchStatus)
		}

		// RAG 相关路由
		if ragHandler != nil {
			rag := api.Group("/rag")
			{
				// 旧的搜索接口（兼容）
				rag.POST("/search", ragHandler.Search)
				// 新的搜索接口（使用 KnowledgeChunk）
				rag.POST("/search/chunks", ragHandler.SearchChunks)
				// 知识片段详情
				rag.GET("/chunks/:id", ragHandler.GetChunkDetail)

				rag.POST("/index", ragHandler.Index)
				rag.GET("/stats", ragHandler.Stats)
				// 获取已索引的项目列表
				rag.GET("/projects", ragHandler.GetIndexedProjects)
				// 新的索引统计（使用 KnowledgeChunk）
				rag.GET("/index/stats", ragHandler.GetIndexStats)
				rag.GET("/config", ragHandler.GetConfig)
				rag.POST("/config", ragHandler.UpdateConfig)
				rag.POST("/config/test", ragHandler.TestConfig)
				rag.POST("/config/llm/test", ragHandler.TestLLMConnection)
				rag.POST("/index/full", ragHandler.TriggerFullIndex)
				rag.GET("/index/progress", ragHandler.GetIndexProgress)
				rag.DELETE("/data", ragHandler.ClearAllData)
				rag.POST("/qdrant/download", ragHandler.DownloadQdrant)
				rag.POST("/qdrant/upload", ragHandler.UploadQdrant)
				rag.POST("/qdrant/start", ragHandler.StartQdrant)
				rag.POST("/qdrant/stop", ragHandler.StopQdrant)
				rag.GET("/qdrant/status", ragHandler.GetQdrantStatus)

				// 增强队列相关
				rag.GET("/enrichment/stats", ragHandler.GetEnrichmentStats)
				rag.POST("/enrichment/retry", ragHandler.RetryEnrichment)
			}
		}
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

	server := &HTTPServer{
		router:           router,
		httpPort:         ":19960",
		logger:           logger,
		pluginService:    pluginService,
		dailySummaryRepo: dailySummaryRepo,
	}

	// 尝试初始化团队服务（可选功能，失败不影响主服务）
	server.initTeamRoutes(api)

	return server
}

// Start 启动服务器
func (s *HTTPServer) Start() error {
	s.server = &http.Server{
		Addr:    s.httpPort,
		Handler: s.router,
	}

	s.logger.Info("HTTP server starting",
		"port", s.httpPort,
	)

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

// initTeamRoutes 初始化团队相关路由（可选功能）
func (s *HTTPServer) initTeamRoutes(api *gin.RouterGroup) {
	// 尝试初始化团队组件
	factory := appTeam.NewTeamFactory()
	components, err := factory.Initialize(19960, "1.0.0", s.dailySummaryRepo)
	if err != nil {
		s.logger.Warn("team service initialization failed, team features disabled",
			"error", err,
		)
		return
	}

	s.teamComponents = components
	s.logger.Info("team service initialized successfully")

	// 创建团队处理器
	teamHandler := handler.NewTeamHandler(
		components.TeamService,
		s.pluginService,
		components.SkillPublisher,
		components.SkillDownloader,
	)

	// 注册团队路由
	team := api.Group("/team")
	{
		// 身份管理
		team.GET("/identity", teamHandler.GetIdentity)
		team.POST("/identity", teamHandler.CreateOrUpdateIdentity)

		// 网络管理
		team.GET("/network/interfaces", teamHandler.GetNetworkInterfaces)

		// 团队管理
		team.POST("/create", teamHandler.CreateTeam)
		team.GET("/discover", teamHandler.DiscoverTeams)
		team.POST("/join", teamHandler.JoinTeam)
		team.GET("/list", teamHandler.ListTeams)
		team.GET("/:id/members", teamHandler.GetTeamMembers)
		team.POST("/:id/leave", teamHandler.LeaveTeam)
		team.POST("/:id/dissolve", teamHandler.DissolveTeam)

		// 技能管理
		team.POST("/skills/validate", teamHandler.ValidateSkill)
		team.GET("/:id/skills", teamHandler.GetSkillIndex)
		team.POST("/:id/skills/publish", teamHandler.PublishSkill)
		team.POST("/:id/skills/publish-with-metadata", teamHandler.PublishSkillWithMetadata)
		team.POST("/:id/skills/download", teamHandler.DownloadSkill)
		team.POST("/:id/skills/:plugin_id/install", teamHandler.InstallTeamSkill)
		team.POST("/:id/skills/:plugin_id/uninstall", teamHandler.UninstallTeamSkill)

		// 协作功能
		collaborationHandler := handler.NewTeamCollaborationHandler(
			components.TeamService,
			components.CollaborationService,
		)
		team.POST("/:id/share-code", collaborationHandler.ShareCode)
		team.POST("/:id/status", collaborationHandler.UpdateWorkStatus)
		team.POST("/:id/daily-summaries/share", collaborationHandler.ShareDailySummary)
		team.GET("/:id/daily-summaries", collaborationHandler.GetDailySummaries)
		team.GET("/:id/daily-summaries/:member_id", collaborationHandler.GetDailySummaryDetail)
	}

	// 注册 P2P 路由（所有成员都暴露）
	p2p := s.router.Group("/p2p")
	{
		p2pHandler := handler.NewP2PHandler(components.SkillPublisher)
		p2p.GET("/health", p2pHandler.Health)
		p2p.GET("/skills/:id/meta", p2pHandler.GetSkillMeta)
		p2p.GET("/skills/:id/download", p2pHandler.DownloadSkill)
		p2p.GET("/daily-summary", p2pHandler.GetDailySummary)
	}
}

// GetTeamComponents 获取团队组件（如果已初始化）
func (s *HTTPServer) GetTeamComponents() *appTeam.TeamComponents {
	return s.teamComponents
}
