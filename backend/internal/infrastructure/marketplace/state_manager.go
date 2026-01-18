package marketplace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	domainMarketplace "github.com/cocursor/backend/internal/domain/marketplace"
)

// StateManager 状态管理器
type StateManager struct {
	statePath string
}

// NewStateManager 创建状态管理器
func NewStateManager() (*StateManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	statePath := filepath.Join(homeDir, ".cocursor", "plugins-state.json")

	return &StateManager{
		statePath: statePath,
	}, nil
}

// ReadState 读取插件状态
func (s *StateManager) ReadState() (*domainMarketplace.PluginState, error) {
	// 如果文件不存在，返回空状态
	if _, err := os.Stat(s.statePath); os.IsNotExist(err) {
		return domainMarketplace.NewPluginState(), nil
	}

	data, err := os.ReadFile(s.statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state domainMarketplace.PluginState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// 如果 InstalledPlugins 为 nil，初始化为空 map
	if state.InstalledPlugins == nil {
		state.InstalledPlugins = make(map[string]domainMarketplace.InstalledPlugin)
	}

	return &state, nil
}

// WriteState 写入插件状态
func (s *StateManager) WriteState(state *domainMarketplace.PluginState) error {
	// 确保目录存在
	dir := filepath.Dir(s.statePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(s.statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// UpdateInstalledPlugin 更新已安装插件状态
func (s *StateManager) UpdateInstalledPlugin(pluginID string, version string) error {
	state, err := s.ReadState()
	if err != nil {
		return err
	}

	state.SetInstalled(pluginID, version)

	return s.WriteState(state)
}

// RemoveInstalledPlugin 移除已安装插件状态
func (s *StateManager) RemoveInstalledPlugin(pluginID string) error {
	state, err := s.ReadState()
	if err != nil {
		return err
	}

	state.RemoveInstalled(pluginID)

	return s.WriteState(state)
}
