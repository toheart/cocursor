package http

import (
	"context"
	"net/http"
	"time"

	"log/slog"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	appMarketplace "github.com/cocursor/backend/internal/application/marketplace"
	appTeam "github.com/cocursor/backend/internal/application/team"
	"github.com/cocursor/backend/internal/infrastructure/git"
	"github.com/cocursor/backend/internal/infrastructure/log"
	infraP2P "github.com/cocursor/backend/internal/infrastructure/p2p"
	"github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/cocursor/backend/internal/interfaces/http/handler"
	"github.com/cocursor/backend/internal/interfaces/http/middleware"
	"github.com/cocursor/backend/internal/interfaces/mcp"
	p2pHandler "github.com/cocursor/backend/internal/interfaces/p2p/handler"
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
	sessionRepo      storage.WorkspaceSessionRepository
	projectManager   *appCursor.ProjectManager
	shutdownChan     chan struct{} // 用于接收关闭信号
}

// NewServer 创建 HTTP 服务器
func NewServer(
	notificationHandler *handler.NotificationHandler,
	statsHandler *handler.StatsHandler,
	projectHandler *handler.ProjectHandler,
	analyticsHandler *handler.AnalyticsHandler,
	workspaceHandler *handler.WorkspaceHandler,
	marketplaceHandler *handler.MarketplaceHandler,
	ragHandler *handler.RAGHandler,
	dailySummaryHandler *handler.DailySummaryHandler,
	weeklySummaryHandler *handler.WeeklySummaryHandler,
	profileHandler *handler.ProfileHandler,
	openspecHandler *handler.OpenSpecHandler,
	todoHandler *handler.TodoHandler,
	codeAnalysisHandler *handler.CodeAnalysisHandler,
	lifecycleHandler *handler.LifecycleHandler,
	mcpServer *mcp.MCPServer,
	pluginService *appMarketplace.PluginService,
	dailySummaryRepo storage.DailySummaryRepository,
	sessionRepo storage.WorkspaceSessionRepository,
	projectManager *appCursor.ProjectManager,
) *HTTPServer {
	router := gin.Default()

	// 添加编码转换中间件，确保请求体是 UTF-8 编码
	// 解决 Windows 下 curl 使用 GBK 编码导致的中文乱码问题
	router.Use(middleware.EnsureUTF8Body())

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
		api.GET("/sessions/active", analyticsHandler.ActiveSessions)

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

		// 日报相关路由
		if dailySummaryHandler != nil {
			api.GET("/daily-summary", dailySummaryHandler.GetDailySummary)
			api.POST("/daily-summary", dailySummaryHandler.SaveDailySummary)
			api.GET("/daily-summary/batch-status", dailySummaryHandler.GetBatchStatus)
			api.GET("/daily-summary/range", dailySummaryHandler.GetDailySummariesRange)
			api.GET("/sessions/daily", dailySummaryHandler.GetDailySessions)
			api.GET("/sessions/conversations", dailySummaryHandler.GetDailyConversations)
			api.GET("/sessions/:sessionId/content", dailySummaryHandler.GetSessionContent)
		}

		// Profile 相关路由
		if profileHandler != nil {
			profile := api.Group("/profile")
			{
				profile.POST("/messages", profileHandler.GetMessages)
				profile.POST("", profileHandler.Save)
			}
		}

		// OpenSpec 相关路由
		if openspecHandler != nil {
			openspec := api.Group("/openspec")
			{
				openspec.GET("/list", openspecHandler.List)
				openspec.POST("/validate", openspecHandler.Validate)
			}
		}

		// 待办事项相关路由
		if todoHandler != nil {
			todos := api.Group("/todos")
			{
				todos.GET("", todoHandler.List)
				todos.POST("", todoHandler.Create)
				todos.PATCH("/:id", todoHandler.Update)
				todos.DELETE("/completed", todoHandler.DeleteCompleted)
				todos.DELETE("/:id", todoHandler.Delete)
			}
		}

		// 周报相关路由
		if weeklySummaryHandler != nil {
			api.GET("/weekly-summary", weeklySummaryHandler.GetWeeklySummary)
			api.POST("/weekly-summary", weeklySummaryHandler.SaveWeeklySummary)
		}

		// 代码分析相关路由
		if codeAnalysisHandler != nil {
			analysis := api.Group("/analysis")
			{
				analysis.POST("/scan-entry-points", codeAnalysisHandler.ScanEntryPoints)
				analysis.POST("/projects", codeAnalysisHandler.RegisterProject)
				analysis.POST("/callgraph/status", codeAnalysisHandler.CheckCallGraphStatus)
				analysis.POST("/callgraph/generate", codeAnalysisHandler.GenerateCallGraph)
				analysis.POST("/callgraph/generate-async", codeAnalysisHandler.GenerateCallGraphAsync)
				analysis.GET("/callgraph/progress/:task_id", codeAnalysisHandler.GetGenerationProgress)
				analysis.POST("/diff", codeAnalysisHandler.AnalyzeDiff)
				analysis.POST("/impact", codeAnalysisHandler.QueryImpact)
			}
		}

		// 设置相关路由
		settingsHandler := handler.NewSettingsHandler()
		settings := api.Group("/settings")
		{
			settings.GET("/cursor-paths", settingsHandler.GetPathStatus)
			settings.POST("/cursor-paths/validate", settingsHandler.ValidatePath)
			settings.POST("/cursor-paths", settingsHandler.SetPaths)
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

		// 生命周期管理相关路由
		if lifecycleHandler != nil {
			api.POST("/heartbeat", lifecycleHandler.Heartbeat)
			lifecycle := api.Group("/lifecycle")
			{
				lifecycle.GET("/status", lifecycleHandler.GetStatus)
			}
		}
	}

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 创建 shutdown channel
	shutdownChan := make(chan struct{}, 1)

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
		sessionRepo:      sessionRepo,
		projectManager:   projectManager,
		shutdownChan:     shutdownChan,
	}

	// 关闭接口（用于 VSCode 插件关闭后端进程）
	api.POST("/shutdown", server.handleShutdown)

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
		team.POST("/:id/status", collaborationHandler.UpdateWorkStatus)
		team.POST("/:id/daily-summaries/share", collaborationHandler.ShareDailySummary)
		team.GET("/:id/daily-summaries", collaborationHandler.GetDailySummaries)
		team.GET("/:id/daily-summaries/:member_id", collaborationHandler.GetDailySummaryDetail)

		// 周报功能
		weeklyReportHandler := handler.NewTeamWeeklyReportHandler(components.WeeklyReportService)
		team.GET("/:id/project-config", weeklyReportHandler.GetProjectConfig)
		team.POST("/:id/project-config", weeklyReportHandler.UpdateProjectConfig)
		team.POST("/:id/project-config/add", weeklyReportHandler.AddProject)
		team.POST("/:id/project-config/add-by-path", weeklyReportHandler.AddProjectByPath)
		team.POST("/:id/project-config/remove", weeklyReportHandler.RemoveProject)
		team.GET("/:id/weekly-report", weeklyReportHandler.GetWeeklyReport)
		team.GET("/:id/members/:member_id/daily-detail", weeklyReportHandler.GetMemberDailyDetail)
		team.POST("/:id/weekly-report/refresh", weeklyReportHandler.RefreshWeeklyStats)
	}

	// 注册 P2P 路由（所有成员都暴露）
	p2p := s.router.Group("/p2p")
	{
		skillHandler := handler.NewP2PHandler(components.SkillPublisher)
		p2p.GET("/health", skillHandler.Health)
		p2p.GET("/skills/:id/meta", skillHandler.GetSkillMeta)
		p2p.GET("/skills/:id/download", skillHandler.DownloadSkill)
		p2p.GET("/daily-summary", skillHandler.GetDailySummary)

		// 周报统计 P2P 接口
		gitCollector := git.NewStatsCollector()
		weeklyStatsHandler := p2pHandler.NewWeeklyStatsHandler(
			gitCollector,
			s.sessionRepo,
			s.dailySummaryRepo,
			s.projectManager,
		)
		p2p.GET("/weekly-stats", weeklyStatsHandler.GetWeeklyStats)
		p2p.GET("/daily-detail", weeklyStatsHandler.GetDailyDetail)
	}

	// 注册团队 P2P 路由（用于团队加入、成员管理等）
	var wsServer *infraP2P.WebSocketServer
	if ws, ok := components.TeamService.GetWebSocketServer().(*infraP2P.WebSocketServer); ok && ws != nil {
		wsServer = ws
	}
	p2pTeamHandler := p2pHandler.NewP2PTeamHandler(
		components.TeamService,
		wsServer,
	)
	p2pTeamHandler.RegisterRoutes(s.router)
}

// GetTeamComponents 获取团队组件（如果已初始化）
func (s *HTTPServer) GetTeamComponents() *appTeam.TeamComponents {
	return s.teamComponents
}

// GetShutdownChan 获取关闭信号通道
func (s *HTTPServer) GetShutdownChan() <-chan struct{} {
	return s.shutdownChan
}

// handleShutdown 处理关闭请求
func (s *HTTPServer) handleShutdown(c *gin.Context) {
	s.logger.Info("Received shutdown request from API")
	c.JSON(http.StatusOK, gin.H{"status": "shutting_down"})

	// 在响应发送后触发关闭
	go func() {
		// 给响应一点时间发送出去
		time.Sleep(100 * time.Millisecond)
		// 发送关闭信号
		select {
		case s.shutdownChan <- struct{}{}:
		default:
			// 已经有关闭信号在队列中
		}
	}()
}
