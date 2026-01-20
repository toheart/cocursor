package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appTeam "github.com/cocursor/backend/internal/application/team"
	domainP2P "github.com/cocursor/backend/internal/domain/p2p"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/interfaces/http/response"
)

// TeamCollaborationHandler 团队协作处理器
type TeamCollaborationHandler struct {
	teamService          *appTeam.TeamService
	collaborationService *appTeam.CollaborationService
}

// NewTeamCollaborationHandler 创建团队协作处理器
func NewTeamCollaborationHandler(
	teamService *appTeam.TeamService,
	collaborationService *appTeam.CollaborationService,
) *TeamCollaborationHandler {
	return &TeamCollaborationHandler{
		teamService:          teamService,
		collaborationService: collaborationService,
	}
}

// ShareCodeRequest 分享代码请求
type ShareCodeRequest struct {
	FileName  string `json:"file_name" binding:"required"`
	FilePath  string `json:"file_path"`
	Language  string `json:"language"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Code      string `json:"code" binding:"required"`
	Message   string `json:"message"`
}

// ShareCode 分享代码片段
// @Summary 分享代码片段到团队
// @Tags 团队协作
// @Accept json
// @Produce json
// @Param id path string true "团队 ID"
// @Param body body ShareCodeRequest true "代码片段信息"
// @Success 200 {object} response.Response
// @Router /team/{id}/share-code [post]
func (h *TeamCollaborationHandler) ShareCode(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		response.Error(c, http.StatusBadRequest, 610001, "Team ID is required")
		return
	}

	var req ShareCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 610002, "Invalid request: "+err.Error())
		return
	}

	// 获取身份
	identity, err := h.teamService.GetIdentity()
	if err != nil {
		response.Error(c, http.StatusBadRequest, 610003, "Please set your identity first")
		return
	}

	// 获取团队信息，确认团队存在
	team, err := h.teamService.GetTeam(teamID)
	if err != nil {
		response.Error(c, http.StatusNotFound, 610004, "Team not found")
		return
	}

	// 检查 Leader 是否在线（只有 Leader 在线才能广播）
	if !team.LeaderOnline && !team.IsLeader {
		response.Error(c, http.StatusServiceUnavailable, 610005, "Team leader is offline, cannot share code")
		return
	}

	// 创建代码片段
	snippet := &domainTeam.CodeSnippet{
		ID:         uuid.New().String(),
		TeamID:     teamID,
		SenderID:   identity.ID,
		SenderName: identity.Name,
		FileName:   req.FileName,
		FilePath:   req.FilePath,
		Language:   req.Language,
		StartLine:  req.StartLine,
		EndLine:    req.EndLine,
		Code:       req.Code,
		Message:    req.Message,
		CreatedAt:  time.Now(),
	}

	// 截断过大的代码片段
	truncated := snippet.Truncate()

	// 广播代码分享事件
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	err = h.collaborationService.ShareCode(ctx, snippet)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 610006, "Failed to share code: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"snippet_id": snippet.ID,
		"truncated":  truncated,
	})
}

// UpdateWorkStatusRequest 更新工作状态请求
type UpdateWorkStatusRequest struct {
	ProjectName   string `json:"project_name"`
	CurrentFile   string `json:"current_file"`
	StatusVisible bool   `json:"status_visible"`
}

// UpdateWorkStatus 更新工作状态
// @Summary 更新成员工作状态
// @Tags 团队协作
// @Accept json
// @Produce json
// @Param id path string true "团队 ID"
// @Param body body UpdateWorkStatusRequest true "工作状态信息"
// @Success 200 {object} response.Response
// @Router /team/{id}/status [post]
func (h *TeamCollaborationHandler) UpdateWorkStatus(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		response.Error(c, http.StatusBadRequest, 610010, "Team ID is required")
		return
	}

	var req UpdateWorkStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 610011, "Invalid request: "+err.Error())
		return
	}

	// 获取身份
	identity, err := h.teamService.GetIdentity()
	if err != nil {
		response.Error(c, http.StatusBadRequest, 610012, "Please set your identity first")
		return
	}

	// 确认团队存在
	_, err = h.teamService.GetTeam(teamID)
	if err != nil {
		response.Error(c, http.StatusNotFound, 610013, "Team not found")
		return
	}

	// 创建工作状态
	status := &domainP2P.MemberWorkStatusPayload{
		MemberID:      identity.ID,
		MemberName:    identity.Name,
		ProjectName:   req.ProjectName,
		CurrentFile:   req.CurrentFile,
		LastActiveAt:  time.Now(),
		StatusVisible: req.StatusVisible,
	}

	// 广播状态变更
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err = h.collaborationService.UpdateWorkStatus(ctx, teamID, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 610014, "Failed to update status: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"success": true,
	})
}

// ShareDailySummaryRequest 分享日报请求
type ShareDailySummaryRequest struct {
	Date string `json:"date" binding:"required"`
}

// ShareDailySummary 分享日报到团队
// @Summary 分享日报到团队
// @Tags 团队协作
// @Accept json
// @Produce json
// @Param id path string true "团队 ID"
// @Param body body ShareDailySummaryRequest true "日报日期"
// @Success 200 {object} response.Response
// @Router /team/{id}/daily-summaries/share [post]
func (h *TeamCollaborationHandler) ShareDailySummary(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		response.Error(c, http.StatusBadRequest, 610020, "Team ID is required")
		return
	}

	var req ShareDailySummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 610021, "Invalid request: "+err.Error())
		return
	}

	// 获取身份
	identity, err := h.teamService.GetIdentity()
	if err != nil {
		response.Error(c, http.StatusBadRequest, 610022, "Please set your identity first")
		return
	}

	// 确认团队存在
	team, err := h.teamService.GetTeam(teamID)
	if err != nil {
		response.Error(c, http.StatusNotFound, 610023, "Team not found")
		return
	}

	// 检查 Leader 是否在线
	if !team.LeaderOnline && !team.IsLeader {
		response.Error(c, http.StatusServiceUnavailable, 610024, "Team leader is offline, cannot share daily summary")
		return
	}

	// 分享日报
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	err = h.collaborationService.ShareDailySummary(ctx, teamID, identity.ID, identity.Name, req.Date)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 610025, "Failed to share daily summary: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"success": true,
	})
}

// GetDailySummaries 获取团队日报列表
// @Summary 获取团队日报列表
// @Tags 团队协作
// @Produce json
// @Param id path string true "团队 ID"
// @Param date query string true "日期 YYYY-MM-DD"
// @Success 200 {object} response.Response
// @Router /team/{id}/daily-summaries [get]
func (h *TeamCollaborationHandler) GetDailySummaries(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		response.Error(c, http.StatusBadRequest, 610030, "Team ID is required")
		return
	}

	date := c.Query("date")
	if date == "" {
		// 默认今天
		date = time.Now().Format("2006-01-02")
	}

	// 确认团队存在
	_, err := h.teamService.GetTeam(teamID)
	if err != nil {
		response.Error(c, http.StatusNotFound, 610031, "Team not found")
		return
	}

	// 获取日报列表
	summaries, err := h.collaborationService.GetDailySummaries(teamID, date)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 610032, "Failed to get daily summaries: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"summaries": summaries,
		"date":      date,
	})
}

// GetDailySummaryDetail 获取日报详情
// @Summary 获取成员日报详情
// @Tags 团队协作
// @Produce json
// @Param id path string true "团队 ID"
// @Param member_id path string true "成员 ID"
// @Param date query string true "日期 YYYY-MM-DD"
// @Success 200 {object} response.Response
// @Router /team/{id}/daily-summaries/{member_id} [get]
func (h *TeamCollaborationHandler) GetDailySummaryDetail(c *gin.Context) {
	teamID := c.Param("id")
	memberID := c.Param("member_id")

	if teamID == "" {
		response.Error(c, http.StatusBadRequest, 610040, "Team ID is required")
		return
	}
	if memberID == "" {
		response.Error(c, http.StatusBadRequest, 610041, "Member ID is required")
		return
	}

	date := c.Query("date")
	if date == "" {
		response.Error(c, http.StatusBadRequest, 610042, "Date is required")
		return
	}

	// 确认团队存在
	_, err := h.teamService.GetTeam(teamID)
	if err != nil {
		response.Error(c, http.StatusNotFound, 610043, "Team not found")
		return
	}

	// 获取日报详情
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	summary, err := h.collaborationService.GetDailySummaryDetail(ctx, teamID, memberID, date)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 610044, "Failed to get daily summary: "+err.Error())
		return
	}

	if summary == nil {
		response.Error(c, http.StatusNotFound, 610045, "Daily summary not found")
		return
	}

	response.Success(c, summary)
}
