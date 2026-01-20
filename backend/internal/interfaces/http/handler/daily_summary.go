package handler

import (
	"net/http"

	"github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// DailySummaryHandler 日报处理器
type DailySummaryHandler struct {
	summaryRepo storage.DailySummaryRepository
}

// NewDailySummaryHandler 创建日报处理器
func NewDailySummaryHandler(summaryRepo storage.DailySummaryRepository) *DailySummaryHandler {
	return &DailySummaryHandler{
		summaryRepo: summaryRepo,
	}
}

// BatchStatusResponse 批量状态响应
type BatchStatusResponse struct {
	Statuses map[string]bool `json:"statuses"`
}

// GetBatchStatus 批量查询日报状态
// @Summary 批量查询日报状态
// @Tags 日报
// @Accept json
// @Produce json
// @Param start_date query string true "开始日期，格式 YYYY-MM-DD"
// @Param end_date query string true "结束日期，格式 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=BatchStatusResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /daily-summary/batch-status [get]
func (h *DailySummaryHandler) GetBatchStatus(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		response.Error(c, http.StatusBadRequest, 700001, "start_date 和 end_date 参数是必需的")
		return
	}

	// 查询日期范围内的日报状态
	statuses, err := h.summaryRepo.FindDatesByRange(startDate, endDate)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 700002, "查询日报状态失败: "+err.Error())
		return
	}

	response.Success(c, BatchStatusResponse{
		Statuses: statuses,
	})
}

// GetDailySummary 获取指定日期的日报
// @Summary 获取指定日期的日报
// @Tags 日报
// @Accept json
// @Produce json
// @Param date query string true "日期，格式 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=domainCursor.DailySummary}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /daily-summary [get]
func (h *DailySummaryHandler) GetDailySummary(c *gin.Context) {
	date := c.Query("date")

	if date == "" {
		response.Error(c, http.StatusBadRequest, 700001, "date 参数是必需的")
		return
	}

	// 查询日报
	summary, err := h.summaryRepo.FindByDate(date)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 700002, "查询日报失败: "+err.Error())
		return
	}

	if summary == nil {
		response.Error(c, http.StatusNotFound, 700003, "未找到该日期的日报")
		return
	}

	response.Success(c, summary)
}
