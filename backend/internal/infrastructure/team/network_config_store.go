package team

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

// NetworkConfigStore 网卡偏好存储
type NetworkConfigStore struct {
	mu       sync.RWMutex
	filePath string
	config   *domainTeam.NetworkConfig
}

// NewNetworkConfigStore 创建网卡配置存储
func NewNetworkConfigStore() (*NetworkConfigStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	filePath := filepath.Join(homeDir, ".cocursor", "team", "network.json")

	store := &NetworkConfigStore{
		filePath: filePath,
	}

	// 尝试加载现有配置
	config, err := store.load()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	store.config = config

	return store, nil
}

// load 从文件加载
func (s *NetworkConfigStore) load() (*domainTeam.NetworkConfig, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	var config domainTeam.NetworkConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse network config file: %w", err)
	}

	return &config, nil
}

// save 保存到文件
func (s *NetworkConfigStore) save() error {
	if s.config == nil {
		return nil
	}

	// 确保目录存在
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal network config: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write network config file: %w", err)
	}

	return nil
}

// Get 获取当前配置
func (s *NetworkConfigStore) Get() *domainTeam.NetworkConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config == nil {
		return nil
	}

	// 返回副本
	configCopy := *s.config
	return &configCopy
}

// Set 设置配置
func (s *NetworkConfigStore) Set(preferredInterface, preferredIP string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = &domainTeam.NetworkConfig{
		PreferredInterface: preferredInterface,
		PreferredIP:        preferredIP,
		LastUpdated:        time.Now(),
	}

	return s.save()
}

// Clear 清除配置
func (s *NetworkConfigStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = nil

	// 删除文件
	if err := os.Remove(s.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete network config file: %w", err)
	}

	return nil
}

// GetPreferredInterface 获取首选网卡
func (s *NetworkConfigStore) GetPreferredInterface() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config == nil {
		return ""
	}
	return s.config.PreferredInterface
}

// GetPreferredIP 获取首选 IP
func (s *NetworkConfigStore) GetPreferredIP() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config == nil {
		return ""
	}
	return s.config.PreferredIP
}
