package handler

import (
	"net/http"

	"github.com/cocursor/backend/internal/application/notification"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// NotificationHandler 通知处理器
type NotificationHandler struct {
	service *notification.Service
}

// NewNotificationHandler 创建通知处理器
func NewNotificationHandler(service *notification.Service) *NotificationHandler {
	return &NotificationHandler{service: service}
}

// Create 创建并推送通知
// @Summary 创建通知
// @Tags 通知
// @Accept json
// @Produce json
// @Param body body notification.CreateNotificationDTO true "通知信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.ErrorResponse
// @Router /notifications [post]
func (h *NotificationHandler) Create(c *gin.Context) {
	var dto notification.CreateNotificationDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Error(c, http.StatusBadRequest, 100001, "参数错误")
		return
	}

	result, err := h.service.CreateAndPush(&dto)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 100004, "创建失败")
		return
	}

	response.Success(c, result)
}
