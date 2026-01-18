package handler

import (
	"net/http"
	"time"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// ProjectHandler 项目查询处理器
type ProjectHandler struct {
	projectManager *appCursor.ProjectManager
	dataMerger     *appCursor.DataMerger
}

// NewProjectHandler 创建项目查询处理器
func NewProjectHandler(projectManager *appCursor.ProjectManager, dataMerger *appCursor.DataMerger) *ProjectHandler {
	return &ProjectHandler{
		projectManager: projectManager,
		dataMerger:     dataMerger,
	}
}

// ListProjects 列出所有项目
// @Summary 列出所有已发现的项目
// @Tags 项目
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]domainCursor.ProjectInfo}
// @Failure 500 {object} response.ErrorResponse
// @Router /project/list [get]
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	projects := h.projectManager.ListAllProjects()

	response.Success(c, gin.H{
		"projects": projects,
		"total":    len(projects),
	})
}

// ActivateProject 激活项目（更新活跃状态）
// @Summary 激活项目
// @Description 接收前端上报的当前项目路径，更新活跃状态
// @Tags 项目
// @Accept json
// @Produce json
// @Param request body ActivateProjectRequest true "激活请求"
// @Success 200 {object} response.Response{data=ActivateProjectResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /project/activate [post]
func (h *ProjectHandler) ActivateProject(c *gin.Context) {
	var req ActivateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 100001, "参数错误: "+err.Error())
		return
	}

	// 查找匹配的项目
	projectName, workspaceInfo := h.projectManager.FindByPath(req.Path)

	if projectName == "" {
		response.Error(c, http.StatusNotFound, 800001, "项目不存在")
		return
	}

	// 标记为活跃
	h.projectManager.MarkWorkspaceActive(workspaceInfo.WorkspaceID)

	// 获取项目信息
	projectInfo := h.projectManager.GetProject(projectName)

	response.Success(c, ActivateProjectResponse{
		Success:      true,
		ProjectName:  projectName,
		ProjectID:    projectInfo.ProjectID,
		IsActive:     true,
		WorkspaceInfo: workspaceInfo,
		Message:      "活跃状态已更新",
	})
}

// GetProjectPrompts 查询项目的 AI 对话历史
// @Summary 查询项目的 AI 对话历史
// @Tags 项目
// @Accept json
// @Produce json
// @Param project_name path string true "项目名称"
// @Success 200 {object} response.Response{data=ProjectPromptsResponse}
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /project/{project_name}/prompts [get]
func (h *ProjectHandler) GetProjectPrompts(c *gin.Context) {
	projectName := c.Param("project_name")

	projectInfo := h.projectManager.GetProject(projectName)
	if projectInfo == nil {
		response.Error(c, http.StatusNotFound, 800001, "项目不存在")
		return
	}

	// 合并所有工作区的 Prompts
	prompts, err := h.dataMerger.MergePrompts(projectInfo.Workspaces)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 800003, "查询失败: "+err.Error())
		return
	}

	response.Success(c, ProjectPromptsResponse{
		ProjectName: projectName,
		Workspaces:   projectInfo.Workspaces,
		Prompts:      prompts,
		Total:        len(prompts),
	})
}

// GetProjectGenerations 查询项目的 AI 生成记录
// @Summary 查询项目的 AI 生成记录
// @Tags 项目
// @Accept json
// @Produce json
// @Param project_name path string true "项目名称"
// @Success 200 {object} response.Response{data=ProjectGenerationsResponse}
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /project/{project_name}/generations [get]
func (h *ProjectHandler) GetProjectGenerations(c *gin.Context) {
	projectName := c.Param("project_name")

	projectInfo := h.projectManager.GetProject(projectName)
	if projectInfo == nil {
		response.Error(c, http.StatusNotFound, 800001, "项目不存在")
		return
	}

	// 合并所有工作区的 Generations
	generations, err := h.dataMerger.MergeGenerations(projectInfo.Workspaces)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 800003, "查询失败: "+err.Error())
		return
	}

	response.Success(c, ProjectGenerationsResponse{
		ProjectName: projectName,
		Workspaces:   projectInfo.Workspaces,
		Generations: generations,
		Total:       len(generations),
	})
}

// GetProjectSessions 查询项目的 Composer 会话
// @Summary 查询项目的 Composer 会话
// @Tags 项目
// @Accept json
// @Produce json
// @Param project_name path string true "项目名称"
// @Success 200 {object} response.Response{data=ProjectSessionsResponse}
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /project/{project_name}/sessions [get]
func (h *ProjectHandler) GetProjectSessions(c *gin.Context) {
	projectName := c.Param("project_name")

	projectInfo := h.projectManager.GetProject(projectName)
	if projectInfo == nil {
		response.Error(c, http.StatusNotFound, 800001, "项目不存在")
		return
	}

	// 合并所有工作区的 Sessions
	sessions, err := h.dataMerger.MergeSessions(projectInfo.Workspaces)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 800003, "查询失败: "+err.Error())
		return
	}

	response.Success(c, ProjectSessionsResponse{
		ProjectName: projectName,
		Workspaces:   projectInfo.Workspaces,
		Sessions:    sessions,
		Total:       len(sessions),
	})
}

// GetProjectAcceptanceStats 查询项目的接受率统计（合并）
// @Summary 查询项目的接受率统计
// @Tags 项目
// @Accept json
// @Produce json
// @Param project_name path string true "项目名称"
// @Param start_date query string false "开始日期 YYYY-MM-DD，默认最近 7 天"
// @Param end_date query string false "结束日期 YYYY-MM-DD，默认今天"
// @Success 200 {object} response.Response{data=ProjectAcceptanceStatsResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /project/{project_name}/stats/acceptance [get]
func (h *ProjectHandler) GetProjectAcceptanceStats(c *gin.Context) {
	projectName := c.Param("project_name")

	projectInfo := h.projectManager.GetProject(projectName)
	if projectInfo == nil {
		response.Error(c, http.StatusNotFound, 800001, "项目不存在")
		return
	}

	// 获取日期参数
	startDate := c.DefaultQuery("start_date", "")
	endDate := c.DefaultQuery("end_date", "")

	// 如果没有提供日期，默认最近 7 天
	if startDate == "" || endDate == "" {
		now := time.Now()
		endDate = now.Format("2006-01-02")
		start := now.AddDate(0, 0, -7)
		startDate = start.Format("2006-01-02")
	}

	// 合并接受率统计
	mergedStats, rawStats, err := h.dataMerger.MergeAcceptanceStats(startDate, endDate)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 800003, "查询失败: "+err.Error())
		return
	}

	response.Success(c, ProjectAcceptanceStatsResponse{
		ProjectName:  projectName,
		Workspaces:    projectInfo.Workspaces,
		MergedStats:   mergedStats,
		RawStats:      rawStats,
	})
}

// ActivateProjectRequest 激活项目请求
type ActivateProjectRequest struct {
	Path      string `json:"path" binding:"required"`       // 项目路径
	Timestamp int64  `json:"timestamp"`                     // 时间戳
}

// ActivateProjectResponse 激活项目响应
type ActivateProjectResponse struct {
	Success       bool                      `json:"success"`
	ProjectName   string                    `json:"project_name"`
	ProjectID     string                    `json:"project_id"`
	IsActive      bool                      `json:"is_active"`
	WorkspaceInfo *domainCursor.WorkspaceInfo `json:"workspace_info"`
	Message       string                    `json:"message,omitempty"`
}

// ProjectPromptsResponse 项目 Prompts 响应
type ProjectPromptsResponse struct {
	ProjectName string                          `json:"project_name"`
	Workspaces  []*domainCursor.WorkspaceInfo   `json:"workspaces"`
	Prompts     []appCursor.PromptWithSource    `json:"prompts"`
	Total       int                             `json:"total"`
}

// ProjectGenerationsResponse 项目 Generations 响应
type ProjectGenerationsResponse struct {
	ProjectName string                            `json:"project_name"`
	Workspaces  []*domainCursor.WorkspaceInfo      `json:"workspaces"`
	Generations []appCursor.GenerationWithSource   `json:"generations"`
	Total       int                               `json:"total"`
}

// ProjectSessionsResponse 项目 Sessions 响应
type ProjectSessionsResponse struct {
	ProjectName string                          `json:"project_name"`
	Workspaces  []*domainCursor.WorkspaceInfo   `json:"workspaces"`
	Sessions    []appCursor.SessionWithSource    `json:"sessions"`
	Total       int                             `json:"total"`
}

// ProjectAcceptanceStatsResponse 项目接受率统计响应
type ProjectAcceptanceStatsResponse struct {
	ProjectName string                              `json:"project_name"`
	Workspaces  []*domainCursor.WorkspaceInfo       `json:"workspaces"`
	MergedStats *domainCursor.DailyAcceptanceStats  `json:"merged_stats"`
	RawStats    []*domainCursor.DailyAcceptanceStats `json:"raw_stats"`
}
