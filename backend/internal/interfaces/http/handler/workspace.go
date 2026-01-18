package handler

import (
	"log"
	"net/http"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	appMarketplace "github.com/cocursor/backend/internal/application/marketplace"
	"github.com/gin-gonic/gin"
)

// WorkspaceHandler 工作区处理器
type WorkspaceHandler struct {
	projectManager *appCursor.ProjectManager
	pluginService  *appMarketplace.PluginService
}

// NewWorkspaceHandler 创建 WorkspaceHandler
func NewWorkspaceHandler(projectManager *appCursor.ProjectManager, pluginService *appMarketplace.PluginService) *WorkspaceHandler {
	return &WorkspaceHandler{
		projectManager: projectManager,
		pluginService:  pluginService,
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

	// 同步已安装插件的技能到 AGENTS.md
	go h.syncInstalledSkillsToAgentsMD(req.Path)

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

	// 同步已安装插件的技能到 AGENTS.md
	workspacePath := req.Path
	if workspacePath == "" && req.WorkspaceID != "" {
		// 如果只提供了 workspaceID，尝试从 projectManager 获取路径
		// 这里简化处理，如果 path 为空则跳过同步
	} else if workspacePath != "" {
		go h.syncInstalledSkillsToAgentsMD(workspacePath)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// syncInstalledSkillsToAgentsMD 同步已安装插件的技能到工作区的 AGENTS.md
func (h *WorkspaceHandler) syncInstalledSkillsToAgentsMD(workspacePath string) {
	if workspacePath == "" {
		return
	}

	// 获取所有已安装的插件
	installedPlugins, err := h.pluginService.GetInstalledPlugins()
	if err != nil {
		log.Printf("[WorkspaceHandler] 获取已安装插件失败: %v", err)
		return
	}

	if len(installedPlugins) == 0 {
		return
	}

	// 对于每个已安装的插件，检查其技能是否在 AGENTS.md 中
	// 这里需要通过 skillInstaller 来访问 agentsUpdater
	// 由于架构限制，我们需要在 PluginService 中添加一个同步方法
	// 或者直接在这里调用 skillInstaller 的方法
	// 为了保持架构清晰，我们在 PluginService 中添加同步方法
	if err := h.pluginService.SyncInstalledSkillsToAgentsMD(workspacePath); err != nil {
		log.Printf("[WorkspaceHandler] 同步技能到 AGENTS.md 失败: %v", err)
	} else {
		log.Printf("[WorkspaceHandler] 成功同步 %d 个插件的技能到工作区: %s", len(installedPlugins), workspacePath)
	}
}
