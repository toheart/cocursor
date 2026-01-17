package handler

import (
	"net/http"

	"github.com/cocursor/backend/internal/application/workspace"
	"github.com/gin-gonic/gin"
)

// WorkspaceHandler 工作区处理器
type WorkspaceHandler struct {
	manager *workspace.Manager
}

// NewWorkspaceHandler 创建 WorkspaceHandler
func NewWorkspaceHandler(manager *workspace.Manager) *WorkspaceHandler {
	return &WorkspaceHandler{
		manager: manager,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Path string `json:"path" binding:"required"`
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	WorkspaceID string `json:"workspaceID"`
	Path        string `json:"path"`
}

// Register 处理工作区注册请求
// POST /api/v1/workspace/register
func (h *WorkspaceHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ws, err := h.manager.Register(req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RegisterResponse{
		WorkspaceID: ws.ID,
		Path:        ws.Path,
	})
}

// FocusRequest 焦点切换请求
type FocusRequest struct {
	WorkspaceID string `json:"workspaceID"`
	Path        string `json:"path"`
}

// Focus 处理工作区焦点切换请求
// POST /api/v1/workspace/focus
func (h *WorkspaceHandler) Focus(c *gin.Context) {
	var req FocusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.WorkspaceID == "" && req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "either workspaceID or path must be provided"})
		return
	}

	err := h.manager.UpdateFocus(req.WorkspaceID, req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
