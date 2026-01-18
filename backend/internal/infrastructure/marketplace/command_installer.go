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
func (c *CommandInstaller) InstallCommand(pluginID string, commandID string) error {
	if commandID == "" {
		return nil // Command 是可选的，如果没有则跳过
	}

	// 读取 Command 文件
	content, err := c.pluginLoader.ReadCommandFile(pluginID)
	if err != nil {
		return fmt.Errorf("failed to read command file: %w", err)
	}

	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// 构建目标文件路径
	commandsDir := filepath.Join(homeDir, ".cursor", "commands")
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
func (c *CommandInstaller) UninstallCommand(commandID string) error {
	if commandID == "" {
		return nil // 如果没有 command ID，跳过
	}

	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// 构建文件路径
	commandPath := filepath.Join(homeDir, ".cursor", "commands", commandID+".md")

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
