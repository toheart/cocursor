package handler

import (
	"log"
	"net/http"

	"github.com/cocursor/backend/internal/application/cursor"
	"github.com/gin-gonic/gin"
)

// WorkspaceHandler 工作区处理器
type WorkspaceHandler struct {
	projectManager *cursor.ProjectManager
}

// NewWorkspaceHandler 创建 WorkspaceHandler
func NewWorkspaceHandler(projectManager *cursor.ProjectManager) *WorkspaceHandler {
	return &WorkspaceHandler{
		projectManager: projectManager,
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

	state, err := h.projectManager.RegisterWorkspace(req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RegisterResponse{
		WorkspaceID: state.WorkspaceID,
		Path:        state.Path,
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
		log.Printf("[WorkspaceHandler.Focus] 解析请求失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[WorkspaceHandler.Focus] 收到请求: workspaceID=%s, path=%s", req.WorkspaceID, req.Path)

	if req.WorkspaceID == "" && req.Path == "" {
		log.Printf("[WorkspaceHandler.Focus] 参数错误: workspaceID 和 path 都为空")
		c.JSON(http.StatusBadRequest, gin.H{"error": "either workspaceID or path must be provided"})
		return
	}

	err := h.projectManager.UpdateWorkspaceFocus(req.WorkspaceID, req.Path)
	if err != nil {
		log.Printf("[WorkspaceHandler.Focus] UpdateWorkspaceFocus 失败: workspaceID=%s, path=%s, error=%v", req.WorkspaceID, req.Path, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[WorkspaceHandler.Focus] 成功: workspaceID=%s, path=%s", req.WorkspaceID, req.Path)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
