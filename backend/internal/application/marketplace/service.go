package marketplace

import (
	"fmt"
	"strings"

	domainMarketplace "github.com/cocursor/backend/internal/domain/marketplace"
	infraMarketplace "github.com/cocursor/backend/internal/infrastructure/marketplace"
)

// PluginService 插件服务
type PluginService struct {
	pluginLoader     *infraMarketplace.PluginLoader
	stateManager     *infraMarketplace.StateManager
	skillInstaller   *infraMarketplace.SkillInstaller
	mcpInstaller     *infraMarketplace.MCPInstaller
	commandInstaller *infraMarketplace.CommandInstaller
}

// NewPluginService 创建插件服务
func NewPluginService(
	pluginLoader *infraMarketplace.PluginLoader,
	stateManager *infraMarketplace.StateManager,
	skillInstaller *infraMarketplace.SkillInstaller,
	mcpInstaller *infraMarketplace.MCPInstaller,
	commandInstaller *infraMarketplace.CommandInstaller,
) *PluginService {
	return &PluginService{
		pluginLoader:     pluginLoader,
		stateManager:     stateManager,
		skillInstaller:   skillInstaller,
		mcpInstaller:     mcpInstaller,
		commandInstaller: commandInstaller,
	}
}

// ListPlugins 列出所有插件
// category: 分类筛选（可选）
// search: 搜索关键词（可选）
// installed: 是否只显示已安装（可选）
func (s *PluginService) ListPlugins(category, search string, installed *bool) ([]*domainMarketplace.Plugin, error) {
	plugins, err := s.pluginLoader.LoadPlugins()
	if err != nil {
		return nil, fmt.Errorf("failed to load plugins: %w", err)
	}

	// 应用筛选
	var filtered []*domainMarketplace.Plugin
	for _, plugin := range plugins {
		// 分类筛选
		if category != "" && plugin.Category != category {
			continue
		}

		// 搜索筛选
		if search != "" {
			searchLower := strings.ToLower(search)
			nameMatch := strings.Contains(strings.ToLower(plugin.Name), searchLower)
			descMatch := strings.Contains(strings.ToLower(plugin.Description), searchLower)
			if !nameMatch && !descMatch {
				continue
			}
		}

		// 已安装筛选
		if installed != nil {
			if *installed && !plugin.Installed {
				continue
			}
			if !*installed && plugin.Installed {
				continue
			}
		}

		filtered = append(filtered, plugin)
	}

	return filtered, nil
}

// GetPlugin 获取插件详情
func (s *PluginService) GetPlugin(id string) (*domainMarketplace.Plugin, error) {
	plugin, err := s.pluginLoader.LoadPlugin(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}
	return plugin, nil
}

// GetInstalledPlugins 获取已安装插件列表
func (s *PluginService) GetInstalledPlugins() ([]*domainMarketplace.Plugin, error) {
	plugins, err := s.pluginLoader.LoadPlugins()
	if err != nil {
		return nil, fmt.Errorf("failed to load plugins: %w", err)
	}

	var installed []*domainMarketplace.Plugin
	for _, plugin := range plugins {
		if plugin.Installed {
			installed = append(installed, plugin)
		}
	}

	return installed, nil
}

// InstallPluginResult 安装插件结果
type InstallPluginResult struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	EnvVars []string `json:"env_vars,omitempty"`
}

// InstallPlugin 安装插件
// workspacePath: 工作区路径（用于更新 AGENTS.md）
func (s *PluginService) InstallPlugin(id, workspacePath string) (*InstallPluginResult, error) {
	// 加载插件
	plugin, err := s.pluginLoader.LoadPlugin(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	// 安装 Skill
	if err := s.skillInstaller.InstallSkill(id, &plugin.Skill, workspacePath); err != nil {
		return nil, fmt.Errorf("failed to install skill: %w", err)
	}

	// 安装 MCP（如果存在）
	if plugin.MCP != nil {
		if err := s.mcpInstaller.InstallMCP(plugin.MCP); err != nil {
			return nil, fmt.Errorf("failed to install MCP: %w", err)
		}
	}

	// 安装 Command（如果存在）
	if plugin.Command != nil {
		if err := s.commandInstaller.InstallCommand(id, plugin.Command.CommandID); err != nil {
			return nil, fmt.Errorf("failed to install command: %w", err)
		}
	}

	// 更新安装状态
	if err := s.stateManager.UpdateInstalledPlugin(id, plugin.Version); err != nil {
		return nil, fmt.Errorf("failed to update plugin state: %w", err)
	}

	// 提取环境变量（从 MCP headers 中）
	var envVars []string
	if plugin.MCP != nil && len(plugin.MCP.Headers) > 0 {
		envVars = s.pluginLoader.ExtractEnvVars(plugin.MCP.Headers)
	}

	return &InstallPluginResult{
		Success: true,
		Message: "Install successful",
		EnvVars: envVars,
	}, nil
}

// UninstallPlugin 卸载插件
// workspacePath: 工作区路径（用于更新 AGENTS.md）
func (s *PluginService) UninstallPlugin(id, workspacePath string) error {
	// 加载插件（获取组件信息）
	plugin, err := s.pluginLoader.LoadPlugin(id)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// 卸载 Skill
	if err := s.skillInstaller.UninstallSkill(plugin.Skill.SkillName, workspacePath); err != nil {
		return fmt.Errorf("failed to uninstall skill: %w", err)
	}

	// 卸载 MCP（如果存在）
	if plugin.MCP != nil {
		if err := s.mcpInstaller.UninstallMCP(plugin.MCP.ServerName); err != nil {
			return fmt.Errorf("failed to uninstall MCP: %w", err)
		}
	}

	// 卸载 Command（如果存在）
	if plugin.Command != nil {
		if err := s.commandInstaller.UninstallCommand(plugin.Command.CommandID); err != nil {
			return fmt.Errorf("failed to uninstall command: %w", err)
		}
	}

	// 更新安装状态
	if err := s.stateManager.RemoveInstalledPlugin(id); err != nil {
		return fmt.Errorf("failed to update plugin state: %w", err)
	}

	return nil
}

// PluginStatus 插件状态
type PluginStatus struct {
	Installed        bool   `json:"installed"`
	InstalledVersion string `json:"installed_version,omitempty"`
	LatestVersion    string `json:"latest_version"`
}

// CheckPluginStatus 检查插件状态
func (s *PluginService) CheckPluginStatus(id string) (*PluginStatus, error) {
	plugin, err := s.pluginLoader.LoadPlugin(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	return &PluginStatus{
		Installed:        plugin.Installed,
		InstalledVersion: plugin.InstalledVersion,
		LatestVersion:    plugin.Version,
	}, nil
}
