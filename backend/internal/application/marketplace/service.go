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

// ListPluginsOptions 列出插件的选项
type ListPluginsOptions struct {
	Category  string // 分类筛选
	Search    string // 搜索关键词
	Installed *bool  // 是否只显示已安装
	Lang      string // 语言 (zh-CN 或 en)
	Source    string // 来源筛选 (builtin/project/team_global/team_project)
	TeamID    string // 团队 ID 筛选
}

// ListPlugins 列出所有插件
// category: 分类筛选（可选）
// search: 搜索关键词（可选）
// installed: 是否只显示已安装（可选）
// lang: 语言（可选），支持 "zh-CN" 和 "en"，默认为 "zh-CN"
func (s *PluginService) ListPlugins(category, search string, installed *bool, lang ...string) ([]*domainMarketplace.Plugin, error) {
	language := ""
	if len(lang) > 0 {
		language = lang[0]
	}
	return s.ListPluginsWithOptions(ListPluginsOptions{
		Category:  category,
		Search:    search,
		Installed: installed,
		Lang:      language,
	})
}

// ListPluginsWithOptions 使用选项列出所有插件
func (s *PluginService) ListPluginsWithOptions(opts ListPluginsOptions) ([]*domainMarketplace.Plugin, error) {
	plugins, err := s.pluginLoader.LoadPlugins()
	if err != nil {
		return nil, fmt.Errorf("failed to load plugins: %w", err)
	}

	// 确定语言，默认为中文
	language := "zh-CN"
	if opts.Lang != "" {
		language = opts.Lang
	}

	// 应用筛选
	var filtered []*domainMarketplace.Plugin
	for _, plugin := range plugins {
		// 来源筛选
		if opts.Source != "" && string(plugin.Source) != opts.Source {
			continue
		}

		// 团队 ID 筛选
		if opts.TeamID != "" && plugin.TeamID != opts.TeamID {
			continue
		}

		// 分类筛选
		if opts.Category != "" && plugin.Category != opts.Category {
			continue
		}

		// 搜索筛选（使用本地化后的名称和描述）
		if opts.Search != "" {
			localizedPlugin := plugin.Localize(language)
			searchLower := strings.ToLower(opts.Search)
			nameMatch := strings.Contains(strings.ToLower(localizedPlugin.Name), searchLower)
			descMatch := strings.Contains(strings.ToLower(localizedPlugin.Description), searchLower)
			if !nameMatch && !descMatch {
				continue
			}
		}

		// 已安装筛选
		if opts.Installed != nil {
			if *opts.Installed && !plugin.Installed {
				continue
			}
			if !*opts.Installed && plugin.Installed {
				continue
			}
		}

		filtered = append(filtered, plugin)
	}

	// 对所有返回的插件进行本地化
	var result []*domainMarketplace.Plugin
	for _, plugin := range filtered {
		result = append(result, plugin.Localize(language))
	}

	return result, nil
}

// GetPlugin 获取插件详情
// lang: 语言（可选），支持 "zh-CN" 和 "en"，默认为 "zh-CN"
func (s *PluginService) GetPlugin(id string, lang ...string) (*domainMarketplace.Plugin, error) {
	plugin, err := s.pluginLoader.LoadPlugin(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	// 确定语言，默认为中文
	language := "zh-CN"
	if len(lang) > 0 && lang[0] != "" {
		language = lang[0]
	}

	// 返回本地化的插件信息
	return plugin.Localize(language), nil
}

// GetInstalledPlugins 获取已安装插件列表
// lang: 语言（可选），支持 "zh-CN" 和 "en"，默认为 "zh-CN"
func (s *PluginService) GetInstalledPlugins(lang ...string) ([]*domainMarketplace.Plugin, error) {
	plugins, err := s.pluginLoader.LoadPlugins()
	if err != nil {
		return nil, fmt.Errorf("failed to load plugins: %w", err)
	}

	// 确定语言，默认为中文
	language := "zh-CN"
	if len(lang) > 0 && lang[0] != "" {
		language = lang[0]
	}

	var installed []*domainMarketplace.Plugin
	for _, plugin := range plugins {
		if plugin.Installed {
			installed = append(installed, plugin.Localize(language))
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
		// 安装所有 Command
		for _, cmd := range plugin.Command.Commands {
			if err := s.commandInstaller.InstallCommand(id, cmd.CommandID, workspacePath); err != nil {
				return nil, fmt.Errorf("failed to install command %s: %w", cmd.CommandID, err)
			}
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
		// 卸载所有 Command
		for _, cmd := range plugin.Command.Commands {
			if err := s.commandInstaller.UninstallCommand(cmd.CommandID, workspacePath); err != nil {
				// 记录错误但继续卸载其他 Command
				continue
			}
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

// SyncInstalledSkillsToAgentsMD 同步已安装插件的技能到工作区的 AGENTS.md
// 当打开新工作区时，确保所有已安装插件的技能都在该工作区的 AGENTS.md 中
func (s *PluginService) SyncInstalledSkillsToAgentsMD(workspacePath string) error {
	if workspacePath == "" {
		return fmt.Errorf("workspace path is required")
	}

	// 获取所有已安装的插件
	installedPlugins, err := s.GetInstalledPlugins()
	if err != nil {
		return fmt.Errorf("failed to get installed plugins: %w", err)
	}

	if len(installedPlugins) == 0 {
		return nil
	}

	// 对于每个已安装的插件，检查其技能是否在 AGENTS.md 中
	for _, plugin := range installedPlugins {
		// 读取插件的 SKILL.md 文件
		skillFiles, err := s.pluginLoader.ReadSkillFiles(plugin.ID)
		if err != nil {
			// 如果无法读取技能文件，跳过该插件
			continue
		}

		// 查找 SKILL.md 文件
		skillMDContent, ok := skillFiles["SKILL.md"]
		if !ok {
			// 如果没有 SKILL.md，跳过
			continue
		}

		// 解析 frontmatter 获取技能元数据
		// 需要通过 skillInstaller 访问 agentsUpdater
		// 由于 skillInstaller 是私有的，我们需要添加一个公开方法
		// 或者在这里直接使用 skillInstaller 的内部方法
		// 为了保持架构清晰，我们在 SkillInstaller 中添加一个同步方法
		if err := s.skillInstaller.SyncSkillToAgentsMD(plugin.ID, plugin.Skill.SkillName, workspacePath, skillMDContent); err != nil {
			// 记录错误但继续处理其他插件
			continue
		}
	}

	return nil
}
