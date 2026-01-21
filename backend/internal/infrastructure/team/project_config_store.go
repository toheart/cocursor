package team

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

// ProjectConfigStore 项目配置存储
type ProjectConfigStore struct {
	mu       sync.RWMutex
	teamID   string
	filePath string
	config   *domainTeam.TeamProjectConfig
}

// NewProjectConfigStore 创建项目配置存储
func NewProjectConfigStore(teamID string) (*ProjectConfigStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	filePath := filepath.Join(homeDir, ".cocursor", "team", teamID, "project_config.json")

	store := &ProjectConfigStore{
		teamID:   teamID,
		filePath: filePath,
	}

	// 尝试加载现有配置
	config, err := store.load()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if config == nil {
		// 初始化空配置
		config = &domainTeam.TeamProjectConfig{
			TeamID:    teamID,
			Projects:  []domainTeam.ProjectMatcher{},
			UpdatedAt: time.Now(),
		}
	}
	store.config = config

	return store, nil
}

// load 从文件加载配置
func (s *ProjectConfigStore) load() (*domainTeam.TeamProjectConfig, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	var config domainTeam.TeamProjectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse project config file: %w", err)
	}

	return &config, nil
}

// save 保存配置到文件
func (s *ProjectConfigStore) save(config *domainTeam.TeamProjectConfig) error {
	// 确保目录存在
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project config: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write project config file: %w", err)
	}

	return nil
}

// Load 获取当前配置
func (s *ProjectConfigStore) Load() (*domainTeam.TeamProjectConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config == nil {
		return nil, fmt.Errorf("project config not initialized")
	}

	// 返回副本
	configCopy := *s.config
	configCopy.Projects = make([]domainTeam.ProjectMatcher, len(s.config.Projects))
	copy(configCopy.Projects, s.config.Projects)
	return &configCopy, nil
}

// Save 保存配置
func (s *ProjectConfigStore) Save(config *domainTeam.TeamProjectConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config.TeamID = s.teamID
	config.UpdatedAt = time.Now()

	if err := s.save(config); err != nil {
		return err
	}

	s.config = config
	return nil
}

// AddProject 添加项目
func (s *ProjectConfigStore) AddProject(name, repoURL string) (*domainTeam.ProjectMatcher, error) {
	if name == "" {
		return nil, fmt.Errorf("project name is required")
	}
	if repoURL == "" {
		return nil, fmt.Errorf("repo URL is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已存在
	for _, p := range s.config.Projects {
		if p.RepoURL == repoURL {
			return nil, fmt.Errorf("project with repo URL %s already exists", repoURL)
		}
	}

	project := domainTeam.ProjectMatcher{
		ID:      uuid.New().String(),
		Name:    name,
		RepoURL: repoURL,
	}

	s.config.Projects = append(s.config.Projects, project)
	s.config.UpdatedAt = time.Now()

	if err := s.save(s.config); err != nil {
		// 回滚
		s.config.Projects = s.config.Projects[:len(s.config.Projects)-1]
		return nil, err
	}

	return &project, nil
}

// RemoveProject 移除项目
func (s *ProjectConfigStore) RemoveProject(projectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, p := range s.config.Projects {
		if p.ID == projectID {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("project not found: %s", projectID)
	}

	// 保存旧项目用于回滚
	oldProject := s.config.Projects[idx]
	s.config.Projects = append(s.config.Projects[:idx], s.config.Projects[idx+1:]...)
	s.config.UpdatedAt = time.Now()

	if err := s.save(s.config); err != nil {
		// 回滚
		s.config.Projects = append(s.config.Projects[:idx], append([]domainTeam.ProjectMatcher{oldProject}, s.config.Projects[idx:]...)...)
		return err
	}

	return nil
}

// UpdateProject 更新项目
func (s *ProjectConfigStore) UpdateProject(projectID, name, repoURL string) (*domainTeam.ProjectMatcher, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, p := range s.config.Projects {
		if p.ID == projectID {
			idx = i
			break
		}
	}

	if idx == -1 {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	// 检查 repoURL 是否与其他项目冲突
	for i, p := range s.config.Projects {
		if i != idx && p.RepoURL == repoURL {
			return nil, fmt.Errorf("project with repo URL %s already exists", repoURL)
		}
	}

	oldProject := s.config.Projects[idx]
	s.config.Projects[idx].Name = name
	s.config.Projects[idx].RepoURL = repoURL
	s.config.UpdatedAt = time.Now()

	if err := s.save(s.config); err != nil {
		// 回滚
		s.config.Projects[idx] = oldProject
		return nil, err
	}

	project := s.config.Projects[idx]
	return &project, nil
}

// GetRepoURLs 获取所有项目的 RepoURL 列表
func (s *ProjectConfigStore) GetRepoURLs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	urls := make([]string, len(s.config.Projects))
	for i, p := range s.config.Projects {
		urls[i] = p.RepoURL
	}
	return urls
}
