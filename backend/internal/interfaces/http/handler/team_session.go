package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	appTeam "github.com/cocursor/backend/internal/application/team"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/interfaces/http/response"
)

// TeamSessionHandler 团队会话分享处理器
type TeamSessionHandler struct {
	sessionSharingService *appTeam.SessionSharingService
	identityService       *appTeam.IdentityService
}

// NewTeamSessionHandler 创建团队会话分享处理器
func NewTeamSessionHandler(
	sessionSharingService *appTeam.SessionSharingService,
	identityService *appTeam.IdentityService,
) *TeamSessionHandler {
	return &TeamSessionHandler{
		sessionSharingService: sessionSharingService,
		identityService:       identityService,
	}
}

// ShareSessionRequest 分享会话请求
type ShareSessionRequest struct {
	SessionID   string                 `json:"session_id" binding:"required"`
	Title       string                 `json:"title" binding:"required"`
	Messages    []map[string]interface{} `json:"messages" binding:"required"`
	Description string                 `json:"description"`
}

// ShareSession 分享会话到团队
// @Summary 分享会话到团队
// @Tags 团队会话
// @Accept json
// @Produce json
// @Param teamId path string true "团队 ID"
// @Param body body ShareSessionRequest true "分享请求"
// @Success 200 {object} response.Response
// @Router /team/{teamId}/sessions/share [post]
func (h *TeamSessionHandler) ShareSession(c *gin.Context) {
	teamID := c.Param("teamId")
	if teamID == "" {
		response.Error(c, http.StatusBadRequest, 610001, "Team ID is required")
		return
	}

	var req domainTeam.ShareSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 610002, "Invalid request: "+err.Error())
		return
	}

	// 获取当前用户身份
	identity, err := h.identityService.GetIdentity()
	if err != nil {
		response.Error(c, http.StatusUnauthorized, 610003, "Identity not found, please create identity first")
		return
	}

	// 分享会话
	shareID, err := h.sessionSharingService.ShareSession(c.Request.Context(), teamID, &req, identity.ID, identity.Name)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 610004, "Failed to share session: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"share_id": shareID,
	})
}

// GetSharedSessions 获取团队分享的会话列表
// @Summary 获取团队分享的会话列表
// @Tags 团队会话
// @Produce json
// @Param teamId path string true "团队 ID"
// @Param limit query int false "每页数量" default(20)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} response.Response
// @Router /team/{teamId}/sessions [get]
func (h *TeamSessionHandler) GetSharedSessions(c *gin.Context) {
	teamID := c.Param("teamId")
	if teamID == "" {
		response.Error(c, http.StatusBadRequest, 610001, "Team ID is required")
		return
	}

	// 解析分页参数
	limit := 20
	offset := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// 获取分享列表
	sessions, total, err := h.sessionSharingService.GetSharedSessions(c.Request.Context(), teamID, limit, offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 610005, "Failed to get shared sessions: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"sessions": sessions,
		"total":    total,
	})
}

// GetSharedSessionDetail 获取分享会话详情
// @Summary 获取分享会话详情
// @Tags 团队会话
// @Produce json
// @Param teamId path string true "团队 ID"
// @Param shareId path string true "分享 ID"
// @Success 200 {object} response.Response
// @Router /team/{teamId}/sessions/{shareId} [get]
func (h *TeamSessionHandler) GetSharedSessionDetail(c *gin.Context) {
	teamID := c.Param("teamId")
	shareID := c.Param("shareId")
	if teamID == "" || shareID == "" {
		response.Error(c, http.StatusBadRequest, 610001, "Team ID and Share ID are required")
		return
	}

	// 获取分享详情
	session, comments, err := h.sessionSharingService.GetSharedSessionDetail(c.Request.Context(), teamID, shareID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 610006, "Failed to get shared session detail: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"session":  session,
		"comments": comments,
	})
}

// AddCommentRequest 添加评论请求
type AddCommentRequest struct {
	Content  string   `json:"content" binding:"required"`
	Mentions []string `json:"mentions"`
}

// AddComment 添加评论
// @Summary 添加评论
// @Tags 团队会话
// @Accept json
// @Produce json
// @Param teamId path string true "团队 ID"
// @Param shareId path string true "分享 ID"
// @Param body body AddCommentRequest true "评论请求"
// @Success 200 {object} response.Response
// @Router /team/{teamId}/sessions/{shareId}/comments [post]
func (h *TeamSessionHandler) AddComment(c *gin.Context) {
	teamID := c.Param("teamId")
	shareID := c.Param("shareId")
	if teamID == "" || shareID == "" {
		response.Error(c, http.StatusBadRequest, 610001, "Team ID and Share ID are required")
		return
	}

	var req domainTeam.AddCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 610002, "Invalid request: "+err.Error())
		return
	}

	// 获取当前用户身份
	identity, err := h.identityService.GetIdentity()
	if err != nil {
		response.Error(c, http.StatusUnauthorized, 610003, "Identity not found, please create identity first")
		return
	}

	// 添加评论
	commentID, err := h.sessionSharingService.AddComment(c.Request.Context(), teamID, shareID, &req, identity.ID, identity.Name)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 610007, "Failed to add comment: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"comment_id": commentID,
	})
}
