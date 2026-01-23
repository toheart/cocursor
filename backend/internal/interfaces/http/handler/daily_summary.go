package handler

import (
	"net/http"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	"github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// DailySummaryHandler 日报处理器
type DailySummaryHandler struct {
	summaryRepo         storage.DailySummaryRepository
	dailySummaryService *appCursor.DailySummaryService
	sessionService      *appCursor.SessionService
}

// NewDailySummaryHandler 创建日报处理器
func NewDailySummaryHandler(
	summaryRepo storage.DailySummaryRepository,
	dailySummaryService *appCursor.DailySummaryService,
	sessionService *appCursor.SessionService,
) *DailySummaryHandler {
	return &DailySummaryHandler{
		summaryRepo:         summaryRepo,
		dailySummaryService: dailySummaryService,
		sessionService:      sessionService,
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

// GetDailySessions 获取每日会话列表
// @Summary 获取每日会话列表
// @Tags 日报
// @Accept json
// @Produce json
// @Param date query string false "日期，格式 YYYY-MM-DD，默认今天"
// @Success 200 {object} response.Response{data=appCursor.DailySessionsResult}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /sessions/daily [get]
func (h *DailySummaryHandler) GetDailySessions(c *gin.Context) {
	date := c.Query("date")

	result, err := h.dailySummaryService.GetDailySessions(date)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 700001, "获取每日会话失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// GetDailyConversations 获取每日对话内容
// @Summary 获取每日对话内容
// @Tags 日报
// @Accept json
// @Produce json
// @Param date query string false "日期，格式 YYYY-MM-DD，默认今天"
// @Success 200 {object} response.Response{data=appCursor.DailyConversationsResult}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /sessions/conversations [get]
func (h *DailySummaryHandler) GetDailyConversations(c *gin.Context) {
	date := c.Query("date")

	result, err := h.dailySummaryService.GetDailyConversations(date)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 700001, "获取每日对话失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// SaveDailySummaryRequest 保存日报请求
type SaveDailySummaryRequest struct {
	Date              string                                 `json:"date" binding:"required"`
	Summary           string                                 `json:"summary" binding:"required"`
	Language          string                                 `json:"language"`
	Projects          interface{}                            `json:"projects"`
	Categories        interface{}                            `json:"categories"`
	TotalSessions     int                                    `json:"total_sessions"`
	CodeChanges       interface{}                            `json:"code_changes"`
	TimeDistribution  interface{}                            `json:"time_distribution"`
	EfficiencyMetrics interface{}                            `json:"efficiency_metrics"`
}

// SaveDailySummary 保存日报
// @Summary 保存日报
// @Tags 日报
// @Accept json
// @Produce json
// @Param body body SaveDailySummaryRequest true "日报内容"
// @Success 200 {object} response.Response{data=appCursor.SaveDailySummaryResult}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /daily-summary [post]
func (h *DailySummaryHandler) SaveDailySummary(c *gin.Context) {
	var req SaveDailySummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 700001, "请求参数错误: "+err.Error())
		return
	}

	input := &appCursor.SaveDailySummaryInput{
		Date:          req.Date,
		Summary:       req.Summary,
		Language:      req.Language,
		TotalSessions: req.TotalSessions,
	}

	result, err := h.dailySummaryService.SaveDailySummary(input)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 700002, "保存日报失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// GetDailySummariesRange 批量查询日报
// @Summary 批量查询日报
// @Tags 日报
// @Accept json
// @Produce json
// @Param start_date query string true "开始日期，格式 YYYY-MM-DD"
// @Param end_date query string true "结束日期，格式 YYYY-MM-DD"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /daily-summary/range [get]
func (h *DailySummaryHandler) GetDailySummariesRange(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		response.Error(c, http.StatusBadRequest, 700001, "start_date 和 end_date 参数是必需的")
		return
	}

	summaries, err := h.dailySummaryService.GetDailySummariesRange(startDate, endDate)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 700002, "查询日报失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"summaries": summaries,
		"count":     len(summaries),
	})
}

// SessionContentResponse 会话内容响应
type SessionContentResponse struct {
	SessionID     string                `json:"session_id"`     // 会话 ID
	Name          string                `json:"name"`           // 会话名称
	ProjectName   string                `json:"project_name"`   // 项目名称
	Messages      []*SessionTextMessage `json:"messages"`       // 消息列表
	TotalMessages int                   `json:"total_messages"` // 总消息数
}

// SessionTextMessage 会话文本消息
type SessionTextMessage struct {
	Type      string `json:"type"`      // 消息类型：user 或 ai
	Text      string `json:"text"`      // 消息内容
	Timestamp int64  `json:"timestamp"` // 时间戳
}

// GetSessionContent 获取会话纯文本内容
// @Summary 获取会话纯文本内容
// @Tags 会话
// @Accept json
// @Produce json
// @Param sessionId path string true "会话 ID"
// @Success 200 {object} response.Response{data=SessionContentResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /sessions/{sessionId}/content [get]
func (h *DailySummaryHandler) GetSessionContent(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		response.Error(c, http.StatusBadRequest, 700001, "session id is required")
		return
	}

	// 获取会话纯文本内容
	messages, err := h.sessionService.GetSessionTextContent(sessionID)
	if err != nil {
		response.Error(c, http.StatusNotFound, 700004, "session not found or cannot read content: "+err.Error())
		return
	}

	// 转换消息格式
	var textMessages []*SessionTextMessage
	for _, msg := range messages {
		msgType := string(msg.Type)
		textMessages = append(textMessages, &SessionTextMessage{
			Type:      msgType,
			Text:      msg.Text,
			Timestamp: msg.Timestamp,
		})
	}

	// 获取会话元信息（名称等）
	sessionDetail, _ := h.sessionService.GetSessionDetail(sessionID, 1)
	sessionName := ""
	projectName := ""
	if sessionDetail != nil && sessionDetail.Session != nil {
		sessionName = sessionDetail.Session.Name
	}

	response.Success(c, SessionContentResponse{
		SessionID:     sessionID,
		Name:          sessionName,
		ProjectName:   projectName,
		Messages:      textMessages,
		TotalMessages: len(textMessages),
	})
}
