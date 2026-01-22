package handler

import (
	"net/http"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// WeeklySummaryHandler 周报处理器
type WeeklySummaryHandler struct {
	weeklySummaryService *appCursor.WeeklySummaryService
}

// NewWeeklySummaryHandler 创建周报处理器
func NewWeeklySummaryHandler(weeklySummaryService *appCursor.WeeklySummaryService) *WeeklySummaryHandler {
	return &WeeklySummaryHandler{
		weeklySummaryService: weeklySummaryService,
	}
}

// SaveWeeklySummaryRequest 保存周报请求
type SaveWeeklySummaryRequest struct {
	WeekStart          string                               `json:"week_start" binding:"required"`
	WeekEnd            string                               `json:"week_end" binding:"required"`
	Summary            string                               `json:"summary" binding:"required"`
	Language           string                               `json:"language"`
	Projects           []*domainCursor.WeeklyProjectSummary `json:"projects"`
	Categories         *domainCursor.WorkCategories         `json:"categories"`
	TotalSessions      int                                  `json:"total_sessions"`
	WorkingDays        int                                  `json:"working_days"`
	CodeChanges        *domainCursor.CodeChangeSummary      `json:"code_changes"`
	KeyAccomplishments []string                             `json:"key_accomplishments"`
}

// SaveWeeklySummary 保存周报
// @Summary 保存周报
// @Tags 周报
// @Accept json
// @Produce json
// @Param body body SaveWeeklySummaryRequest true "周报内容"
// @Success 200 {object} response.Response{data=appCursor.SaveWeeklySummaryResult}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /weekly-summary [post]
func (h *WeeklySummaryHandler) SaveWeeklySummary(c *gin.Context) {
	var req SaveWeeklySummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 700001, "请求参数错误: "+err.Error())
		return
	}

	input := &appCursor.SaveWeeklySummaryInput{
		WeekStart:          req.WeekStart,
		WeekEnd:            req.WeekEnd,
		Summary:            req.Summary,
		Language:           req.Language,
		Projects:           req.Projects,
		Categories:         req.Categories,
		TotalSessions:      req.TotalSessions,
		WorkingDays:        req.WorkingDays,
		CodeChanges:        req.CodeChanges,
		KeyAccomplishments: req.KeyAccomplishments,
	}

	result, err := h.weeklySummaryService.SaveWeeklySummary(input)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 700002, "保存周报失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// GetWeeklySummary 获取周报（带幂等检查）
// @Summary 获取周报
// @Tags 周报
// @Accept json
// @Produce json
// @Param week_start query string true "周起始日期，格式 YYYY-MM-DD（周一）"
// @Success 200 {object} response.Response{data=appCursor.GetWeeklySummaryResult}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /weekly-summary [get]
func (h *WeeklySummaryHandler) GetWeeklySummary(c *gin.Context) {
	weekStart := c.Query("week_start")

	if weekStart == "" {
		response.Error(c, http.StatusBadRequest, 700001, "week_start 参数是必需的")
		return
	}

	result, err := h.weeklySummaryService.GetWeeklySummary(weekStart)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 700002, "查询周报失败: "+err.Error())
		return
	}

	response.Success(c, result)
}
