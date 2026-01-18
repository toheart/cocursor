package marketplace

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	domainMarketplace "github.com/cocursor/backend/internal/domain/marketplace"
)

//go:embed plugins
var pluginsFS embed.FS

// PluginLoader 插件加载器
type PluginLoader struct {
	stateManager *StateManager
}

// NewPluginLoader 创建插件加载器
func NewPluginLoader(stateManager *StateManager) *PluginLoader {
	return &PluginLoader{
		stateManager: stateManager,
	}
}

// LoadPlugins 加载所有插件
func (l *PluginLoader) LoadPlugins() ([]*domainMarketplace.Plugin, error) {
	state, err := l.stateManager.ReadState()
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %w", err)
	}

	entries, err := pluginsFS.ReadDir("plugins")
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins directory: %w", err)
	}

	var plugins []*domainMarketplace.Plugin

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginID := entry.Name()
		plugin, err := l.LoadPlugin(pluginID)
		if err != nil {
			// 跳过无法加载的插件，记录错误但继续处理其他插件
			continue
		}

		// 合并已安装状态
		if state.IsInstalled(pluginID) {
			plugin.Installed = true
			plugin.InstalledVersion = state.GetInstalledVersion(pluginID)
		}

		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

// LoadPlugin 加载单个插件
func (l *PluginLoader) LoadPlugin(pluginID string) (*domainMarketplace.Plugin, error) {
	// embed 文件系统使用正斜杠作为路径分隔符
	pluginJSONPath := "plugins/" + pluginID + "/plugin.json"

	data, err := pluginsFS.ReadFile(pluginJSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin config: %w", err)
	}

	var plugin domainMarketplace.Plugin
	if err := json.Unmarshal(data, &plugin); err != nil {
		return nil, fmt.Errorf("failed to parse plugin config: %w", err)
	}

	// 验证插件数据
	if err := plugin.Validate(); err != nil {
		return nil, fmt.Errorf("plugin validation failed: %w", err)
	}

	return &plugin, nil
}

// ReadSkillFiles 读取 Skill 文件
// 返回文件映射：相对路径 -> 文件内容
func (l *PluginLoader) ReadSkillFiles(pluginID string) (map[string][]byte, error) {
	// embed 文件系统使用正斜杠作为路径分隔符
	skillDir := "plugins/" + pluginID + "/skill"
	files := make(map[string][]byte)

	err := fs.WalkDir(pluginsFS, skillDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		data, err := pluginsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// 计算相对路径（相对于 skill 目录）
		relPath, err := filepath.Rel(skillDir, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		// 统一使用正斜杠作为路径分隔符（跨平台兼容）
		relPath = strings.ReplaceAll(relPath, "\\", "/")

		files[relPath] = data
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk skill directory: %w", err)
	}

	return files, nil
}

// ReadCommandFile 读取 Command 文件
// pluginID: 插件 ID
// commandID: 命令 ID（文件名，不含 .md）
func (l *PluginLoader) ReadCommandFile(pluginID string, commandID string) ([]byte, error) {
	// embed 文件系统使用正斜杠作为路径分隔符
	commandPath := "plugins/" + pluginID + "/command/" + commandID + ".md"

	data, err := pluginsFS.ReadFile(commandPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read command file: %w", err)
	}

	return data, nil
}

// ReadCommandFiles 读取所有 Command 文件
// 返回文件映射：commandID -> 文件内容
func (l *PluginLoader) ReadCommandFiles(pluginID string) (map[string][]byte, error) {
	commandDir := "plugins/" + pluginID + "/command"
	files := make(map[string][]byte)

	err := fs.WalkDir(pluginsFS, commandDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		data, err := pluginsFS.ReadFile(path)
		if err != nil {
			return err
		}

		// 提取文件名（不含 .md）
		fileName := filepath.Base(path)
		commandID := strings.TrimSuffix(fileName, ".md")
		files[commandID] = data
		return nil
	})

	if err != nil {
		// 如果目录不存在，返回空映射而不是错误（Command 是可选的）
		if strings.Contains(err.Error(), "does not exist") {
			return files, nil
		}
		return nil, fmt.Errorf("failed to walk command directory: %w", err)
	}

	return files, nil
}

// ExtractEnvVars 从 MCP headers 中提取环境变量
// 返回环境变量名称列表
func (l *PluginLoader) ExtractEnvVars(headers map[string]string) []string {
	var envVars []string
	seen := make(map[string]bool)

	for _, value := range headers {
		// 查找 ${env:VAR_NAME} 格式
		start := 0
		for {
			idx := strings.Index(value[start:], "${env:")
			if idx == -1 {
				break
			}
			idx += start

			// 找到变量名开始位置
			varStart := idx + 6 // "${env:" 的长度
			varEnd := strings.Index(value[varStart:], "}")
			if varEnd == -1 {
				break
			}
			varEnd += varStart

			varName := value[varStart:varEnd]
			if !seen[varName] {
				envVars = append(envVars, varName)
				seen[varName] = true
			}

			start = varEnd + 1
		}
	}

	return envVars
}
