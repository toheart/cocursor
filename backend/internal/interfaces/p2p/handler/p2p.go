package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/cocursor/backend/internal/infrastructure/marketplace"
)

// P2PHandler P2P 通用处理器
// 提供所有成员都暴露的接口（健康检查、技能下载）
type P2PHandler struct {
	skillPublisher *marketplace.TeamSkillPublisher
	memberID       string
	memberName     string
	teamIDs        []string
}

// NewP2PHandler 创建 P2P 通用处理器
func NewP2PHandler(skillPublisher *marketplace.TeamSkillPublisher) *P2PHandler {
	return &P2PHandler{
		skillPublisher: skillPublisher,
	}
}

// UpdateInfo 更新成员信息
func (h *P2PHandler) UpdateInfo(memberID, memberName string, teamIDs []string) {
	h.memberID = memberID
	h.memberName = memberName
	h.teamIDs = teamIDs
}

// Health 健康检查
// 路由: GET /p2p/health
func (h *P2PHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":      "online",
		"member_id":   h.memberID,
		"member_name": h.memberName,
		"teams":       h.teamIDs,
	})
}

// GetSkillMeta 获取技能元数据
// 路由: GET /p2p/skills/:id/meta
func (h *P2PHandler) GetSkillMeta(c *gin.Context) {
	skillID := c.Param("id")
	if skillID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "skill id required"})
		return
	}

	if h.skillPublisher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "skill provider not available"})
		return
	}

	meta, err := h.skillPublisher.GetSkillMeta(skillID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "skill not found"})
		return
	}

	c.JSON(http.StatusOK, meta)
}

// DownloadSkill 下载技能
// 路由: GET /p2p/skills/:id/download
func (h *P2PHandler) DownloadSkill(c *gin.Context) {
	skillID := c.Param("id")
	if skillID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "skill id required"})
		return
	}

	if h.skillPublisher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "skill provider not available"})
		return
	}

	archive, err := h.skillPublisher.GetSkillArchive(skillID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "skill not found"})
		return
	}

	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", "attachment; filename="+skillID+".tar.gz")
	c.Data(http.StatusOK, "application/gzip", archive)
}

// RegisterRoutes 注册路由
func (h *P2PHandler) RegisterRoutes(router *gin.Engine) {
	p2p := router.Group("/p2p")
	{
		p2p.GET("/health", h.Health)
		p2p.GET("/skills/:id/meta", h.GetSkillMeta)
		p2p.GET("/skills/:id/download", h.DownloadSkill)
	}
}
