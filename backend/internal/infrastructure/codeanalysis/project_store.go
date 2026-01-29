package codeanalysis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cocursor/backend/internal/domain/codeanalysis"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"gopkg.in/yaml.v3"
)

// ProjectStore 项目配置存储实现
type ProjectStore struct {
	configPath string
	logger     *slog.Logger
	mu         sync.RWMutex
	projects   map[string]*codeanalysis.Project
}

// projectsConfig 项目配置文件结构
type projectsConfig struct {
	Projects []*codeanalysis.Project `yaml:"projects"`
}

// NewProjectStore 创建项目配置存储
func NewProjectStore() (*ProjectStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".cocursor", "analysis")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "projects.yaml")

	store := &ProjectStore{
		configPath: configPath,
		logger:     log.NewModuleLogger("codeanalysis", "project_store"),
		projects:   make(map[string]*codeanalysis.Project),
	}

	// 加载现有配置
	if err := store.load(); err != nil {
		store.logger.Warn("failed to load projects config, starting fresh", "error", err)
	}

	return store, nil
}

// load 从文件加载配置
func (s *ProjectStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在是正常的
		}
		return err
	}

	var config projectsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	s.projects = make(map[string]*codeanalysis.Project)
	for _, p := range config.Projects {
		s.projects[p.ID] = p
	}

	return nil
}

// save 保存配置到文件
func (s *ProjectStore) save() error {
	config := projectsConfig{
		Projects: make([]*codeanalysis.Project, 0, len(s.projects)),
	}

	for _, p := range s.projects {
		config.Projects = append(config.Projects, p)
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}

	return os.WriteFile(s.configPath, data, 0644)
}

// GetByID 根据 ID 获取项目配置
func (s *ProjectStore) GetByID(_ context.Context, id string) (*codeanalysis.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if p, ok := s.projects[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("project not found: %s", id)
}

// GetByPath 根据本地路径获取项目配置
func (s *ProjectStore) GetByPath(_ context.Context, path string) (*codeanalysis.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 规范化路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	for _, p := range s.projects {
		for _, localPath := range p.LocalPaths {
			if localPath == absPath {
				return p, nil
			}
		}
	}
	return nil, fmt.Errorf("project not found for path: %s", path)
}

// GetByRemoteURL 根据远程 URL 获取项目配置
func (s *ProjectStore) GetByRemoteURL(_ context.Context, remoteURL string) (*codeanalysis.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	normalizedURL := normalizeRemoteURL(remoteURL)
	for _, p := range s.projects {
		if p.RemoteURL == normalizedURL {
			return p, nil
		}
	}
	return nil, fmt.Errorf("project not found for remote URL: %s", remoteURL)
}

// List 获取所有项目配置
func (s *ProjectStore) List(_ context.Context) ([]*codeanalysis.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*codeanalysis.Project, 0, len(s.projects))
	for _, p := range s.projects {
		result = append(result, p)
	}
	return result, nil
}

// Save 保存项目配置
func (s *ProjectStore) Save(_ context.Context, project *codeanalysis.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确保有 ID
	if project.ID == "" {
		project.ID = generateProjectID(project.RemoteURL)
	}

	// 设置时间戳
	now := time.Now()
	if project.CreatedAt.IsZero() {
		project.CreatedAt = now
	}
	project.UpdatedAt = now

	// 设置默认值
	if project.Algorithm == "" {
		project.Algorithm = codeanalysis.AlgorithmRTA
	}
	if project.Exclude == nil {
		project.Exclude = []string{"vendor/", "*_test.go"}
	}

	s.projects[project.ID] = project

	return s.save()
}

// Delete 删除项目配置
func (s *ProjectStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.projects[id]; !ok {
		return fmt.Errorf("project not found: %s", id)
	}

	delete(s.projects, id)
	return s.save()
}

// AddLocalPath 为项目添加本地路径
func (s *ProjectStore) AddLocalPath(_ context.Context, id string, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.projects[id]
	if !ok {
		return fmt.Errorf("project not found: %s", id)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// 检查是否已存在
	for _, existing := range p.LocalPaths {
		if existing == absPath {
			return nil
		}
	}

	p.LocalPaths = append(p.LocalPaths, absPath)
	p.UpdatedAt = time.Now()

	return s.save()
}

// RemoveLocalPath 移除项目的本地路径
func (s *ProjectStore) RemoveLocalPath(_ context.Context, id string, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.projects[id]
	if !ok {
		return fmt.Errorf("project not found: %s", id)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	newPaths := make([]string, 0, len(p.LocalPaths))
	for _, existing := range p.LocalPaths {
		if existing != absPath {
			newPaths = append(newPaths, existing)
		}
	}

	p.LocalPaths = newPaths
	p.UpdatedAt = time.Now()

	return s.save()
}

// generateProjectID 根据 remote URL 生成项目 ID
func generateProjectID(remoteURL string) string {
	normalized := normalizeRemoteURL(remoteURL)
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:8]) // 使用前 8 字节（16 字符）
}

// normalizeRemoteURL 规范化远程 URL
func normalizeRemoteURL(url string) string {
	normalized := strings.ToLower(url)
	normalized = strings.TrimSuffix(normalized, ".git")

	// 移除协议前缀
	normalized = strings.TrimPrefix(normalized, "https://")
	normalized = strings.TrimPrefix(normalized, "http://")
	normalized = strings.TrimPrefix(normalized, "ssh://")

	// 处理 git@ 格式
	if strings.HasPrefix(normalized, "git@") {
		normalized = strings.TrimPrefix(normalized, "git@")
		normalized = strings.Replace(normalized, ":", "/", 1)
	}

	return normalized
}

// GetProjectID 根据 remote URL 获取项目 ID（公开方法）
func GetProjectID(remoteURL string) string {
	return generateProjectID(remoteURL)
}

// 确保实现接口
var _ codeanalysis.ProjectRepository = (*ProjectStore)(nil)
