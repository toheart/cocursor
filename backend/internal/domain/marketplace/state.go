package marketplace

import "time"

// PluginState 插件状态
type PluginState struct {
	InstalledPlugins map[string]InstalledPlugin `json:"installed_plugins"` // 已安装插件映射
}

// InstalledPlugin 已安装插件信息
type InstalledPlugin struct {
	Version     string    `json:"version"`      // 已安装的版本
	InstalledAt time.Time `json:"installed_at"` // 安装时间
}

// NewPluginState 创建新的插件状态
func NewPluginState() *PluginState {
	return &PluginState{
		InstalledPlugins: make(map[string]InstalledPlugin),
	}
}

// IsInstalled 检查插件是否已安装
func (s *PluginState) IsInstalled(pluginID string) bool {
	_, exists := s.InstalledPlugins[pluginID]
	return exists
}

// GetInstalledVersion 获取已安装的版本
func (s *PluginState) GetInstalledVersion(pluginID string) string {
	if plugin, exists := s.InstalledPlugins[pluginID]; exists {
		return plugin.Version
	}
	return ""
}

// SetInstalled 设置插件为已安装
func (s *PluginState) SetInstalled(pluginID string, version string) {
	s.InstalledPlugins[pluginID] = InstalledPlugin{
		Version:     version,
		InstalledAt: time.Now(),
	}
}

// RemoveInstalled 移除已安装状态
func (s *PluginState) RemoveInstalled(pluginID string) {
	delete(s.InstalledPlugins, pluginID)
}
