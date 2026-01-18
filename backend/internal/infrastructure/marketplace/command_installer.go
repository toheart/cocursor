package marketplace

import (
	"fmt"
	"os"
	"path/filepath"
)

// CommandInstaller Command 安装器
type CommandInstaller struct {
	pluginLoader *PluginLoader
}

// NewCommandInstaller 创建 Command 安装器
func NewCommandInstaller(pluginLoader *PluginLoader) *CommandInstaller {
	return &CommandInstaller{
		pluginLoader: pluginLoader,
	}
}

// InstallCommand 安装 Command 文件
// pluginID: 插件 ID
// commandID: 命令 ID（文件名，不含 .md）
// workspacePath: 工作区路径（必需）
func (c *CommandInstaller) InstallCommand(pluginID string, commandID string, workspacePath string) error {
	if commandID == "" {
		return nil // Command 是可选的，如果没有则跳过
	}

	if workspacePath == "" {
		return fmt.Errorf("workspace path is required")
	}

	// 读取 Command 文件
	content, err := c.pluginLoader.ReadCommandFile(pluginID, commandID)
	if err != nil {
		return fmt.Errorf("failed to read command file: %w", err)
	}

	// 项目级：<workspace>/.cursor/commands/
	commandsDir := filepath.Join(workspacePath, ".cursor", "commands")
	targetPath := filepath.Join(commandsDir, commandID+".md")

	// 确保目录存在
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}

	// 写入文件（覆盖已存在的文件）
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write command file: %w", err)
	}

	return nil
}

// UninstallCommand 卸载 Command 文件
// commandID: 命令 ID
// workspacePath: 工作区路径（必需）
func (c *CommandInstaller) UninstallCommand(commandID string, workspacePath string) error {
	if commandID == "" {
		return nil // 如果没有 command ID，跳过
	}

	if workspacePath == "" {
		return fmt.Errorf("workspace path is required")
	}

	// 项目级：<workspace>/.cursor/commands/
	commandPath := filepath.Join(workspacePath, ".cursor", "commands", commandID+".md")

	// 检查文件是否存在
	if _, err := os.Stat(commandPath); os.IsNotExist(err) {
		// 文件不存在，认为已经卸载
		return nil
	}

	// 删除文件
	if err := os.Remove(commandPath); err != nil {
		return fmt.Errorf("failed to remove command file: %w", err)
	}

	return nil
}
