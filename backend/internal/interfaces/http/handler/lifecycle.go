package handler

import (
	"net/http"

	"github.com/cocursor/backend/internal/application/lifecycle"
	domainLifecycle "github.com/cocursor/backend/internal/domain/lifecycle"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// LifecycleHandler 生命周期管理 API 处理器
type LifecycleHandler struct {
	windowManager *lifecycle.WindowManager
}

// NewLifecycleHandler 创建生命周期处理器
func NewLifecycleHandler(windowManager *lifecycle.WindowManager) *LifecycleHandler {
	return &LifecycleHandler{
		windowManager: windowManager,
	}
}

// HeartbeatRequest 心跳请求
type HeartbeatRequest struct {
	WindowID    string `json:"window_id" binding:"required"`
	ProjectPath string `json:"project_path,omitempty"`
}

// Heartbeat 处理窗口心跳
// @Summary 窗口心跳
// @Description VSCode 窗口定期发送心跳以维持活跃状态
// @Tags 生命周期
// @Accept json
// @Produce json
// @Param request body HeartbeatRequest true "心跳请求"
// @Success 200 {object} response.Response{data=domainLifecycle.HeartbeatResponse}
// @Router /api/v1/heartbeat [post]
func (h *LifecycleHandler) Heartbeat(c *gin.Context) {
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	resp := h.windowManager.Heartbeat(req.WindowID, req.ProjectPath)
	response.Success(c, resp)
}

// GetStatus 获取生命周期状态
// @Summary 获取生命周期状态
// @Description 获取当前活跃窗口列表和空闲状态（用于调试）
// @Tags 生命周期
// @Produce json
// @Success 200 {object} response.Response{data=domainLifecycle.LifecycleStatus}
// @Router /api/v1/lifecycle/status [get]
func (h *LifecycleHandler) GetStatus(c *gin.Context) {
	status := h.windowManager.GetStatus()
	response.Success(c, status)
}

// 确保编译时检查未使用的导入
var _ = domainLifecycle.HeartbeatResponse{}
