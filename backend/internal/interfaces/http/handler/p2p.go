package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/cocursor/backend/internal/infrastructure/marketplace"
	"github.com/cocursor/backend/internal/interfaces/http/response"
)

// P2PHandler P2P 通用处理器（所有成员暴露）
type P2PHandler struct {
	skillPublisher *marketplace.TeamSkillPublisher
}

// NewP2PHandler 创建 P2P 处理器
func NewP2PHandler(skillPublisher *marketplace.TeamSkillPublisher) *P2PHandler {
	return &P2PHandler{
		skillPublisher: skillPublisher,
	}
}

// Health P2P 健康检查
// @Summary P2P 健康检查
// @Tags P2P
// @Produce json
// @Success 200 {object} response.Response
// @Router /p2p/health [get]
func (h *P2PHandler) Health(c *gin.Context) {
	response.Success(c, gin.H{
		"status": "ok",
		"p2p":    true,
	})
}

// GetSkillMeta 获取技能元数据
// @Summary 获取技能元数据
// @Tags P2P
// @Produce json
// @Param id path string true "技能 ID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.ErrorResponse
// @Router /p2p/skills/{id}/meta [get]
func (h *P2PHandler) GetSkillMeta(c *gin.Context) {
	skillID := c.Param("id")

	meta, err := h.skillPublisher.GetSkillMeta(skillID)
	if err != nil {
		response.Error(c, http.StatusNotFound, 700001, "Skill not found: "+err.Error())
		return
	}

	response.Success(c, meta)
}

// DownloadSkill 下载技能文件
// @Summary 下载技能文件
// @Tags P2P
// @Produce application/octet-stream
// @Param id path string true "技能 ID"
// @Success 200 {file} binary
// @Failure 404 {object} response.ErrorResponse
// @Router /p2p/skills/{id}/download [get]
func (h *P2PHandler) DownloadSkill(c *gin.Context) {
	skillID := c.Param("id")

	archive, err := h.skillPublisher.GetSkillArchive(skillID)
	if err != nil {
		response.Error(c, http.StatusNotFound, 700002, "Skill not found: "+err.Error())
		return
	}

	// 设置响应头
	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", "attachment; filename="+skillID+".tar.gz")
	c.Data(http.StatusOK, "application/gzip", archive)
}
