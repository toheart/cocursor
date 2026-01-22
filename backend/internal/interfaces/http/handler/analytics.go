package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// 确保 domainCursor 类型在 Swagger 注释中被识别
var (
	_ domainCursor.TokenUsage
	_ domainCursor.WorkAnalysis
	_ domainCursor.SessionDetail
	_ domainCursor.ActiveSessionsOverview
)

// AnalyticsHandler 分析处理器
type AnalyticsHandler struct {
	tokenService       *appCursor.TokenService
	workAnalysisService *appCursor.WorkAnalysisService
	sessionService     *appCursor.SessionService
}

// NewAnalyticsHandler 创建分析处理器
func NewAnalyticsHandler(
	tokenService *appCursor.TokenService,
	workAnalysisService *appCursor.WorkAnalysisService,
	sessionService *appCursor.SessionService,
) *AnalyticsHandler {
	return &AnalyticsHandler{
		tokenService:       tokenService,
		workAnalysisService: workAnalysisService,
		sessionService:     sessionService,
	}
}

// TokenUsage 获取 Token 使用统计
// @Summary 获取 Token 使用统计
// @Tags 统计
// @Accept json
// @Produce json
// @Param date query string false "日期，格式 YYYY-MM-DD，默认今天"
// @Param project_name query string false "项目名称，如 cocursor（可通过 GET /api/v1/project/list 查看所有项目）"
// @Success 200 {object} response.Response{data=domainCursor.TokenUsage}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /stats/token-usage [get]
func (h *AnalyticsHandler) TokenUsage(c *gin.Context) {
	// 获取日期参数
	date := c.DefaultQuery("date", "")
	
	// 获取项目名参数
	projectName := c.Query("project_name")

	// 调用服务获取 Token 使用统计
	usage, err := h.tokenService.GetTokenUsage(date, projectName)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 700001, "获取 Token 使用统计失败: "+err.Error())
		return
	}

	response.Success(c, usage)
}

// WorkAnalysis 获取工作分析数据（全局视图）
// @Summary 获取工作分析数据
// @Tags 统计
// @Accept json
// @Produce json
// @Param start_date query string false "开始日期，格式 YYYY-MM-DD，默认最近 7 天"
// @Param end_date query string false "结束日期，格式 YYYY-MM-DD，默认今天"
// @Success 200 {object} response.Response{data=domainCursor.WorkAnalysis}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /stats/work-analysis [get]
func (h *AnalyticsHandler) WorkAnalysis(c *gin.Context) {
	// 获取日期参数
	endDate := c.DefaultQuery("end_date", "")
	startDate := c.DefaultQuery("start_date", "")

	// 如果没有提供日期，默认最近 7 天
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}
	if startDate == "" {
		end, _ := time.Parse("2006-01-02", endDate)
		start := end.AddDate(0, 0, -7)
		startDate = start.Format("2006-01-02")
	}

	// 调用服务获取工作分析数据（全局视图，不按项目过滤）
	analysis, err := h.workAnalysisService.GetWorkAnalysis(startDate, endDate)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 700001, "获取工作分析数据失败: "+err.Error())
		return
	}

	response.Success(c, analysis)
}

// ActiveSessions 获取活跃会话概览
// @Summary 获取活跃会话概览
// @Tags 会话
// @Accept json
// @Produce json
// @Param workspace_id query string false "工作区 ID，如果不提供则聚合所有工作区"
// @Success 200 {object} response.Response{data=domainCursor.ActiveSessionsOverview}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /sessions/active [get]
func (h *AnalyticsHandler) ActiveSessions(c *gin.Context) {
	workspaceID := c.Query("workspace_id")

	// 调用服务获取活跃会话概览
	overview, err := h.workAnalysisService.GetActiveSessionsOverview(workspaceID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 700001, "获取活跃会话概览失败: "+err.Error())
		return
	}

	response.Success(c, overview)
}

// SessionList 获取会话列表（分页）
// @Summary 获取会话列表
// @Tags 会话
// @Accept json
// @Produce json
// @Param project_name query string false "项目名称"
// @Param limit query int false "每页条数，默认 20，最大 100"
// @Param offset query int false "偏移量，默认 0"
// @Param search query string false "搜索关键词（会话名称）"
// @Success 200 {object} response.Response{data=[]domainCursor.ComposerData}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /sessions/list [get]
func (h *AnalyticsHandler) SessionList(c *gin.Context) {
	// 获取参数
	projectName := c.Query("project_name")
	search := c.Query("search")

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	// 调用服务获取会话列表
	sessions, total, _, err := h.sessionService.GetSessionList(projectName, limit, offset, search)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 700001, "获取会话列表失败: "+err.Error())
		return
	}

	// 计算页码（从 1 开始）
	page := offset/limit + 1
	if offset == 0 {
		page = 1
	}

	// 返回分页响应
	response.SuccessWithPage(c, sessions, page, limit, total)
}

// SessionDetail 获取会话详情（完整对话）
// @Summary 获取会话详情
// @Tags 会话
// @Accept json
// @Produce json
// @Param sessionId path string true "会话 ID"
// @Param limit query int false "限制消息数量，默认 100，最大 1000"
// @Success 200 {object} response.Response{data=domainCursor.SessionDetail}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /sessions/{sessionId}/detail [get]
func (h *AnalyticsHandler) SessionDetail(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		response.Error(c, http.StatusBadRequest, 700001, "sessionId 参数不能为空")
		return
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	// 调用服务获取会话详情
	detail, err := h.sessionService.GetSessionDetail(sessionID, limit)
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			response.Error(c, http.StatusNotFound, 700001, err.Error())
		} else {
			response.Error(c, http.StatusBadRequest, 700001, "获取会话详情失败: "+err.Error())
		}
		return
	}

	response.Success(c, detail)
}
