package handler

import (
	"net/http"
	"os"

	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// SettingsHandler 设置处理器
type SettingsHandler struct{}

// NewSettingsHandler 创建设置处理器
func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{}
}

// GetPathStatus 获取 Cursor 路径配置状态
// GET /api/v1/settings/cursor-paths
func (h *SettingsHandler) GetPathStatus(c *gin.Context) {
	pathResolver := infraCursor.NewPathResolver()
	status := pathResolver.GetPathStatus()

	// 如果有错误，返回需要配置的提示
	if !status.UserDataDirOK || !status.ProjectsDirOK {
		response.Success(c, gin.H{
			"status":         status,
			"needs_config":   true,
			"config_methods": getConfigMethods(status.IsWSL),
		})
		return
	}

	response.Success(c, gin.H{
		"status":       status,
		"needs_config": false,
	})
}

// ValidatePath 验证路径是否有效
// POST /api/v1/settings/cursor-paths/validate
func (h *SettingsHandler) ValidatePath(c *gin.Context) {
	var req struct {
		PathType string `json:"path_type" binding:"required"` // "user_data_dir" 或 "projects_dir"
		Path     string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid request: "+err.Error())
		return
	}

	// 检查路径是否存在
	info, err := os.Stat(req.Path)
	if err != nil {
		response.Success(c, gin.H{
			"valid":   false,
			"error":   "路径不存在",
			"details": err.Error(),
		})
		return
	}

	if !info.IsDir() {
		response.Success(c, gin.H{
			"valid": false,
			"error": "路径不是目录",
		})
		return
	}

	// 根据路径类型验证特定文件/目录是否存在
	switch req.PathType {
	case "user_data_dir":
		// 检查 globalStorage 目录是否存在
		globalStoragePath := req.Path + "/globalStorage"
		if _, err := os.Stat(globalStoragePath); err != nil {
			response.Success(c, gin.H{
				"valid":   false,
				"error":   "目录结构不正确",
				"details": "未找到 globalStorage 子目录，请确保路径指向 Cursor/User 目录",
			})
			return
		}
	case "projects_dir":
		// projects 目录结构验证可以更宽松
		// 只要是目录就行
	}

	response.Success(c, gin.H{
		"valid": true,
	})
}

// SetPaths 设置 Cursor 路径（通过环境变量方式）
// POST /api/v1/settings/cursor-paths
// 注意：这个接口只返回配置说明，实际配置需要用户手动设置环境变量或修改启动脚本
func (h *SettingsHandler) SetPaths(c *gin.Context) {
	var req struct {
		UserDataDir string `json:"user_data_dir"`
		ProjectsDir string `json:"projects_dir"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid request: "+err.Error())
		return
	}

	// 验证路径
	var errors []string
	if req.UserDataDir != "" {
		if _, err := os.Stat(req.UserDataDir); err != nil {
			errors = append(errors, "user_data_dir 路径不存在: "+err.Error())
		}
	}
	if req.ProjectsDir != "" {
		if _, err := os.Stat(req.ProjectsDir); err != nil {
			errors = append(errors, "projects_dir 路径不存在: "+err.Error())
		}
	}

	if len(errors) > 0 {
		response.Error(c, http.StatusBadRequest, 400, "路径验证失败: "+errors[0])
		return
	}

	// 生成配置说明
	configInstructions := generateConfigInstructions(req.UserDataDir, req.ProjectsDir)

	response.Success(c, gin.H{
		"message":      "请按照以下说明配置环境变量，然后重启服务",
		"instructions": configInstructions,
		"env_vars": gin.H{
			"CURSOR_USER_DATA_DIR": req.UserDataDir,
			"CURSOR_PROJECTS_DIR":  req.ProjectsDir,
		},
	})
}

// getConfigMethods 获取配置方法说明
func getConfigMethods(isWSL bool) []gin.H {
	methods := []gin.H{
		{
			"method": "环境变量",
			"description": "在启动服务前设置环境变量",
			"example_linux": "export CURSOR_USER_DATA_DIR=/path/to/Cursor/User\nexport CURSOR_PROJECTS_DIR=/path/to/.cursor/projects",
			"example_windows": "set CURSOR_USER_DATA_DIR=C:\\Users\\xxx\\AppData\\Roaming\\Cursor\\User\nset CURSOR_PROJECTS_DIR=C:\\Users\\xxx\\.cursor\\projects",
		},
		{
			"method": "启动脚本",
			"description": "在启动脚本中添加环境变量设置",
		},
	}

	if isWSL {
		methods = append(methods, gin.H{
			"method":      "WSL 配置",
			"description": "在 WSL 环境中，路径需要使用 /mnt/c/... 格式访问 Windows 文件系统",
			"example": gin.H{
				"user_data_dir": "/mnt/c/Users/<你的Windows用户名>/AppData/Roaming/Cursor/User",
				"projects_dir":  "/mnt/c/Users/<你的Windows用户名>/.cursor/projects",
			},
			"tip": "将 <你的Windows用户名> 替换为你的 Windows 用户名（可通过 ls /mnt/c/Users 查看）",
		})
	}

	return methods
}

// generateConfigInstructions 生成配置说明
func generateConfigInstructions(userDataDir, projectsDir string) []gin.H {
	var instructions []gin.H

	// Bash/Zsh 配置
	bashConfig := ""
	if userDataDir != "" {
		bashConfig += "export CURSOR_USER_DATA_DIR=\"" + userDataDir + "\"\n"
	}
	if projectsDir != "" {
		bashConfig += "export CURSOR_PROJECTS_DIR=\"" + projectsDir + "\"\n"
	}
	if bashConfig != "" {
		instructions = append(instructions, gin.H{
			"shell":   "bash/zsh",
			"file":    "~/.bashrc 或 ~/.zshrc",
			"content": bashConfig,
			"steps": []string{
				"1. 编辑配置文件：nano ~/.bashrc (或 ~/.zshrc)",
				"2. 添加上述环境变量",
				"3. 重新加载配置：source ~/.bashrc",
				"4. 重启 cocursor 服务",
			},
		})
	}

	// systemd 服务配置
	systemdConfig := "[Service]\n"
	if userDataDir != "" {
		systemdConfig += "Environment=\"CURSOR_USER_DATA_DIR=" + userDataDir + "\"\n"
	}
	if projectsDir != "" {
		systemdConfig += "Environment=\"CURSOR_PROJECTS_DIR=" + projectsDir + "\"\n"
	}
	instructions = append(instructions, gin.H{
		"shell":   "systemd",
		"file":    "/etc/systemd/system/cocursor.service",
		"content": systemdConfig,
		"steps": []string{
			"1. 编辑服务文件",
			"2. 在 [Service] 部分添加 Environment 配置",
			"3. 重新加载配置：systemctl daemon-reload",
			"4. 重启服务：systemctl restart cocursor",
		},
	})

	return instructions
}
