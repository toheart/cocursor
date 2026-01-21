package p2p

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/cocursor/backend/internal/infrastructure/log"
)

// WeeklyStatsProvider 周报统计提供者接口
// 避免循环导入，由 interfaces/p2p/handler.WeeklyStatsHandler 实现
type WeeklyStatsProvider interface {
	// GetWeeklyStats 获取周统计数据
	GetWeeklyStats(c *gin.Context)
	// GetDailyDetail 获取日详情数据
	GetDailyDetail(c *gin.Context)
}

// P2PHTTPServer P2P HTTP 服务
// 提供技能下载等 P2P 接口
type P2PHTTPServer struct {
	router              *gin.Engine
	skillProvider       SkillProvider
	weeklyStatsProvider WeeklyStatsProvider
	healthInfo          *HealthInfo
	logger              *slog.Logger
}

// SkillProvider 技能提供者接口
type SkillProvider interface {
	// GetSkillMeta 获取技能元数据
	GetSkillMeta(skillID string) (*SkillMeta, error)
	// GetSkillArchive 获取技能文件打包
	GetSkillArchive(skillID string) ([]byte, error)
}

// SkillMeta 技能元数据
type SkillMeta struct {
	PluginID    string   `json:"plugin_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Files       []string `json:"files"`
	TotalSize   int64    `json:"total_size"`
	Checksum    string   `json:"checksum"`
}

// HealthInfo 健康检查信息
type HealthInfo struct {
	MemberID   string   `json:"member_id"`
	MemberName string   `json:"member_name"`
	Teams      []string `json:"teams"`
}

// NewP2PHTTPServer 创建 P2P HTTP 服务
func NewP2PHTTPServer(router *gin.Engine, skillProvider SkillProvider) *P2PHTTPServer {
	server := &P2PHTTPServer{
		router:        router,
		skillProvider: skillProvider,
		healthInfo:    &HealthInfo{},
		logger:        log.NewModuleLogger("p2p", "http_server"),
	}

	server.registerRoutes()
	return server
}

// registerRoutes 注册路由
func (s *P2PHTTPServer) registerRoutes() {
	p2p := s.router.Group("/p2p")
	{
		p2p.GET("/health", s.handleHealth)
		p2p.GET("/skills/:id/meta", s.handleSkillMeta)
		p2p.GET("/skills/:id/download", s.handleSkillDownload)

		// 周报统计路由（由 WeeklyStatsHandler 处理）
		// 这些路由需要在设置 WeeklyStatsHandler 后才可用
	}
}

// registerWeeklyStatsRoutes 注册周报统计路由
func (s *P2PHTTPServer) registerWeeklyStatsRoutes() {
	if s.weeklyStatsProvider == nil {
		return
	}
	p2p := s.router.Group("/p2p")
	{
		p2p.GET("/weekly-stats", s.weeklyStatsProvider.GetWeeklyStats)
		p2p.GET("/daily-detail", s.weeklyStatsProvider.GetDailyDetail)
	}
}

// handleHealth 健康检查
func (s *P2PHTTPServer) handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":      "online",
		"member_id":   s.healthInfo.MemberID,
		"member_name": s.healthInfo.MemberName,
		"teams":       s.healthInfo.Teams,
	})
}

// handleSkillMeta 获取技能元数据
func (s *P2PHTTPServer) handleSkillMeta(c *gin.Context) {
	skillID := c.Param("id")
	if skillID == "" {
		c.JSON(400, gin.H{"error": "skill id is required"})
		return
	}

	if s.skillProvider == nil {
		c.JSON(500, gin.H{"error": "skill provider not configured"})
		return
	}

	meta, err := s.skillProvider.GetSkillMeta(skillID)
	if err != nil {
		s.logger.Warn("failed to get skill meta",
			"skill_id", skillID,
			"error", err,
		)
		c.JSON(404, gin.H{"error": "skill not found"})
		return
	}

	c.JSON(200, meta)
}

// handleSkillDownload 下载技能文件
func (s *P2PHTTPServer) handleSkillDownload(c *gin.Context) {
	skillID := c.Param("id")
	if skillID == "" {
		c.JSON(400, gin.H{"error": "skill id is required"})
		return
	}

	if s.skillProvider == nil {
		c.JSON(500, gin.H{"error": "skill provider not configured"})
		return
	}

	archive, err := s.skillProvider.GetSkillArchive(skillID)
	if err != nil {
		s.logger.Warn("failed to get skill archive",
			"skill_id", skillID,
			"error", err,
		)
		c.JSON(404, gin.H{"error": "skill not found"})
		return
	}

	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", "attachment; filename="+skillID+".tar.gz")
	c.Data(200, "application/gzip", archive)
}

// UpdateHealthInfo 更新健康检查信息
func (s *P2PHTTPServer) UpdateHealthInfo(info *HealthInfo) {
	s.healthInfo = info
}

// SetSkillProvider 设置技能提供者
func (s *P2PHTTPServer) SetSkillProvider(provider SkillProvider) {
	s.skillProvider = provider
}

// SetWeeklyStatsProvider 设置周报统计提供者并注册路由
func (s *P2PHTTPServer) SetWeeklyStatsProvider(provider WeeklyStatsProvider) {
	s.weeklyStatsProvider = provider
	s.registerWeeklyStatsRoutes()
	s.logger.Info("weekly stats provider registered")
}
