package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appTeam "github.com/cocursor/backend/internal/application/team"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

// TeamWeeklyReportHandler 团队周报处理器
type TeamWeeklyReportHandler struct {
	weeklyReportService *appTeam.WeeklyReportService
}

// NewTeamWeeklyReportHandler 创建团队周报处理器
func NewTeamWeeklyReportHandler(weeklyReportService *appTeam.WeeklyReportService) *TeamWeeklyReportHandler {
	return &TeamWeeklyReportHandler{
		weeklyReportService: weeklyReportService,
	}
}

// GetProjectConfig 获取项目配置
// @Summary 获取团队项目配置
// @Description 获取团队的项目配置列表
// @Tags Team Weekly Report
// @Accept json
// @Produce json
// @Param id path string true "团队 ID"
// @Success 200 {object} domainTeam.TeamProjectConfig
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/team/{id}/project-config [get]
func (h *TeamWeeklyReportHandler) GetProjectConfig(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id is required"})
		return
	}

	config, err := h.weeklyReportService.GetProjectConfig(teamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateProjectConfigRequest 更新项目配置请求
type UpdateProjectConfigRequest struct {
	Projects []ProjectMatcherRequest `json:"projects" binding:"required"`
}

// ProjectMatcherRequest 项目匹配规则请求
type ProjectMatcherRequest struct {
	ID      string `json:"id"`
	Name    string `json:"name" binding:"required"`
	RepoURL string `json:"repo_url" binding:"required"`
}

// UpdateProjectConfig 更新项目配置
// @Summary 更新团队项目配置
// @Description 更新团队的项目配置列表（仅 Leader 可操作）
// @Tags Team Weekly Report
// @Accept json
// @Produce json
// @Param id path string true "团队 ID"
// @Param request body UpdateProjectConfigRequest true "项目配置"
// @Success 200 {object} domainTeam.TeamProjectConfig
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/team/{id}/project-config [post]
func (h *TeamWeeklyReportHandler) UpdateProjectConfig(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id is required"})
		return
	}

	var req UpdateProjectConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转换请求
	projects := make([]domainTeam.ProjectMatcher, len(req.Projects))
	for i, p := range req.Projects {
		projects[i] = domainTeam.ProjectMatcher{
			ID:      p.ID,
			Name:    p.Name,
			RepoURL: p.RepoURL,
		}
	}

	if err := h.weeklyReportService.UpdateProjectConfig(c.Request.Context(), teamID, projects); err != nil {
		if err.Error() == "only leader can update project config" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回更新后的配置
	config, err := h.weeklyReportService.GetProjectConfig(teamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// AddProjectRequest 添加项目请求
type AddProjectRequest struct {
	Name    string `json:"name" binding:"required"`
	RepoURL string `json:"repo_url" binding:"required"`
}

// AddProject 添加项目
// @Summary 添加团队项目
// @Description 向团队添加新的项目配置（仅 Leader 可操作）
// @Tags Team Weekly Report
// @Accept json
// @Produce json
// @Param id path string true "团队 ID"
// @Param request body AddProjectRequest true "项目信息"
// @Success 200 {object} domainTeam.ProjectMatcher
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/team/{id}/project-config/add [post]
func (h *TeamWeeklyReportHandler) AddProject(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id is required"})
		return
	}

	var req AddProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project, err := h.weeklyReportService.AddProject(c.Request.Context(), teamID, req.Name, req.RepoURL)
	if err != nil {
		if err.Error() == "only leader can add project" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, project)
}

// RemoveProjectRequest 移除项目请求
type RemoveProjectRequest struct {
	ProjectID string `json:"project_id" binding:"required"`
}

// RemoveProject 移除项目
// @Summary 移除团队项目
// @Description 从团队移除项目配置（仅 Leader 可操作）
// @Tags Team Weekly Report
// @Accept json
// @Produce json
// @Param id path string true "团队 ID"
// @Param request body RemoveProjectRequest true "项目 ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/team/{id}/project-config/remove [post]
func (h *TeamWeeklyReportHandler) RemoveProject(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id is required"})
		return
	}

	var req RemoveProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.weeklyReportService.RemoveProject(c.Request.Context(), teamID, req.ProjectID); err != nil {
		if err.Error() == "only leader can remove project" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetWeeklyReport 获取周报
// @Summary 获取团队周报
// @Description 获取团队的周报数据，包括日历视图和项目汇总
// @Tags Team Weekly Report
// @Accept json
// @Produce json
// @Param id path string true "团队 ID"
// @Param week_start query string true "周起始日期（YYYY-MM-DD，会自动调整到周一）"
// @Success 200 {object} domainTeam.TeamWeeklyView
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/team/{id}/weekly-report [get]
func (h *TeamWeeklyReportHandler) GetWeeklyReport(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id is required"})
		return
	}

	weekStart := c.Query("week_start")
	if weekStart == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "week_start is required"})
		return
	}

	view, err := h.weeklyReportService.GetWeeklyReport(c.Request.Context(), teamID, weekStart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, view)
}

// GetMemberDailyDetail 获取成员日详情
// @Summary 获取成员日详情
// @Description 获取指定成员指定日期的详细工作数据
// @Tags Team Weekly Report
// @Accept json
// @Produce json
// @Param id path string true "团队 ID"
// @Param member_id path string true "成员 ID"
// @Param date query string true "日期（YYYY-MM-DD）"
// @Success 200 {object} domainTeam.MemberDailyDetail
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/team/{id}/members/{member_id}/daily-detail [get]
func (h *TeamWeeklyReportHandler) GetMemberDailyDetail(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id is required"})
		return
	}

	memberID := c.Param("member_id")
	if memberID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "member_id is required"})
		return
	}

	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required"})
		return
	}

	detail, err := h.weeklyReportService.GetMemberDailyDetail(c.Request.Context(), teamID, memberID, date)
	if err != nil {
		if err.Error() == "member not found: "+memberID {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// RefreshWeeklyStats 刷新周统计数据
// @Summary 刷新周统计数据
// @Description 强制刷新所有在线成员的周统计数据
// @Tags Team Weekly Report
// @Accept json
// @Produce json
// @Param id path string true "团队 ID"
// @Param week_start query string true "周起始日期（YYYY-MM-DD）"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/team/{id}/weekly-report/refresh [post]
func (h *TeamWeeklyReportHandler) RefreshWeeklyStats(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id is required"})
		return
	}

	weekStart := c.Query("week_start")
	if weekStart == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "week_start is required"})
		return
	}

	if err := h.weeklyReportService.RefreshWeeklyStats(c.Request.Context(), teamID, weekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
