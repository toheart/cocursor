package team

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

// TeamStore 团队列表存储
type TeamStore struct {
	mu       sync.RWMutex
	filePath string
	teams    map[string]*domainTeam.Team // teamID -> Team
}

// teamsFile 团队列表文件结构
type teamsFile struct {
	Teams []domainTeam.Team `json:"teams"`
}

// NewTeamStore 创建团队存储
func NewTeamStore() (*TeamStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	filePath := filepath.Join(homeDir, ".cocursor", "team", "teams.json")

	store := &TeamStore{
		filePath: filePath,
		teams:    make(map[string]*domainTeam.Team),
	}

	// 尝试加载现有数据
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

// load 从文件加载
func (s *TeamStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var file teamsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse teams file: %w", err)
	}

	s.teams = make(map[string]*domainTeam.Team)
	for i := range file.Teams {
		team := file.Teams[i]
		s.teams[team.ID] = &team
	}

	return nil
}

// save 保存到文件
func (s *TeamStore) save() error {
	// 确保目录存在
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 转换为列表
	var file teamsFile
	for _, team := range s.teams {
		file.Teams = append(file.Teams, *team)
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal teams: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write teams file: %w", err)
	}

	return nil
}

// Get 获取指定团队
func (s *TeamStore) Get(teamID string) (*domainTeam.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	team, exists := s.teams[teamID]
	if !exists {
		return nil, domainTeam.ErrTeamNotFound
	}

	// 返回副本
	teamCopy := *team
	return &teamCopy, nil
}

// List 获取所有团队列表
// 排序：Leader 的团队在前，其余按加入时间倒序
func (s *TeamStore) List() []*domainTeam.Team {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var teams []*domainTeam.Team
	for _, team := range s.teams {
		teamCopy := *team
		teams = append(teams, &teamCopy)
	}

	// 排序
	sort.Slice(teams, func(i, j int) bool {
		// Leader 的团队优先
		if teams[i].IsLeader != teams[j].IsLeader {
			return teams[i].IsLeader
		}
		// 按加入时间倒序
		return teams[i].JoinedAt.After(teams[j].JoinedAt)
	})

	return teams
}

// Add 添加团队
func (s *TeamStore) Add(team *domainTeam.Team) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已存在
	if _, exists := s.teams[team.ID]; exists {
		return domainTeam.ErrAlreadyTeamMember
	}

	teamCopy := *team
	s.teams[team.ID] = &teamCopy

	return s.save()
}

// Update 更新团队
func (s *TeamStore) Update(team *domainTeam.Team) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.teams[team.ID]; !exists {
		return domainTeam.ErrTeamNotFound
	}

	teamCopy := *team
	s.teams[team.ID] = &teamCopy

	return s.save()
}

// Remove 移除团队
func (s *TeamStore) Remove(teamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.teams[teamID]; !exists {
		return nil // 不存在也不报错
	}

	delete(s.teams, teamID)

	// 同时删除团队目录
	homeDir, _ := os.UserHomeDir()
	teamDir := filepath.Join(homeDir, ".cocursor", "team", teamID)
	os.RemoveAll(teamDir) // 忽略错误

	return s.save()
}

// GetLeaderTeam 获取作为 Leader 的团队（只能有一个）
func (s *TeamStore) GetLeaderTeam() *domainTeam.Team {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, team := range s.teams {
		if team.IsLeader {
			teamCopy := *team
			return &teamCopy
		}
	}
	return nil
}

// HasLeaderTeam 是否有作为 Leader 的团队
func (s *TeamStore) HasLeaderTeam() bool {
	return s.GetLeaderTeam() != nil
}

// UpdateLastSync 更新最后同步时间
func (s *TeamStore) UpdateLastSync(teamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, exists := s.teams[teamID]
	if !exists {
		return domainTeam.ErrTeamNotFound
	}

	team.LastSyncAt = time.Now()
	return s.save()
}

// UpdateLeaderOnline 更新 Leader 在线状态
func (s *TeamStore) UpdateLeaderOnline(teamID string, online bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, exists := s.teams[teamID]
	if !exists {
		return domainTeam.ErrTeamNotFound
	}

	team.LeaderOnline = online
	return s.save()
}

// Count 获取团队数量
func (s *TeamStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.teams)
}
