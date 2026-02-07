package team

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/cocursor/backend/internal/infrastructure/config"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

// MemberStore 成员列表存储（Leader 使用）
type MemberStore struct {
	mu       sync.RWMutex
	teamID   string
	filePath string
	members  map[string]*domainTeam.TeamMember // memberID -> Member
}

// membersFile 成员列表文件结构
type membersFile struct {
	TeamID  string                  `json:"team_id"`
	Members []domainTeam.TeamMember `json:"members"`
}

// NewMemberStore 创建成员存储
func NewMemberStore(teamID string) (*MemberStore, error) {
	filePath := filepath.Join(config.GetDataDir(), "team", teamID, "members.json")

	store := &MemberStore{
		teamID:   teamID,
		filePath: filePath,
		members:  make(map[string]*domainTeam.TeamMember),
	}

	// 尝试加载现有数据
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

// load 从文件加载
func (s *MemberStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var file membersFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse members file: %w", err)
	}

	s.members = make(map[string]*domainTeam.TeamMember)
	for i := range file.Members {
		member := file.Members[i]
		// 启动时重置非 Leader 成员的在线状态，因为此时尚无 WebSocket 连接
		if !member.IsLeader {
			member.IsOnline = false
		}
		s.members[member.ID] = &member
	}

	return nil
}

// save 保存到文件
func (s *MemberStore) save() error {
	// 确保目录存在
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 转换为列表
	file := membersFile{
		TeamID: s.teamID,
	}
	for _, member := range s.members {
		file.Members = append(file.Members, *member)
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal members: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write members file: %w", err)
	}

	return nil
}

// Get 获取指定成员
func (s *MemberStore) Get(memberID string) (*domainTeam.TeamMember, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	member, exists := s.members[memberID]
	if !exists {
		return nil, domainTeam.ErrNotTeamMember
	}

	// 返回副本
	memberCopy := *member
	return &memberCopy, nil
}

// List 获取所有成员列表
// 排序：Leader 在前，其余按加入时间排序
func (s *MemberStore) List() []*domainTeam.TeamMember {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var members []*domainTeam.TeamMember
	for _, member := range s.members {
		memberCopy := *member
		members = append(members, &memberCopy)
	}

	// 排序
	sort.Slice(members, func(i, j int) bool {
		// Leader 优先
		if members[i].IsLeader != members[j].IsLeader {
			return members[i].IsLeader
		}
		// 按加入时间排序
		return members[i].JoinedAt.Before(members[j].JoinedAt)
	})

	return members
}

// Add 添加成员
func (s *MemberStore) Add(member *domainTeam.TeamMember) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已存在
	if _, exists := s.members[member.ID]; exists {
		return domainTeam.ErrAlreadyTeamMember
	}

	memberCopy := *member
	s.members[member.ID] = &memberCopy

	return s.save()
}

// Update 更新成员
func (s *MemberStore) Update(member *domainTeam.TeamMember) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.members[member.ID]; !exists {
		return domainTeam.ErrNotTeamMember
	}

	memberCopy := *member
	s.members[member.ID] = &memberCopy

	return s.save()
}

// Remove 移除成员
func (s *MemberStore) Remove(memberID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.members[memberID]; !exists {
		return nil // 不存在也不报错
	}

	delete(s.members, memberID)
	return s.save()
}

// SetOnline 设置成员在线状态
func (s *MemberStore) SetOnline(memberID string, online bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	member, exists := s.members[memberID]
	if !exists {
		return nil // 不存在静默返回
	}

	member.IsOnline = online
	return s.save()
}

// UpdateEndpoint 更新成员端点
func (s *MemberStore) UpdateEndpoint(memberID, endpoint string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	member, exists := s.members[memberID]
	if !exists {
		return domainTeam.ErrNotTeamMember
	}

	member.Endpoint = endpoint
	return s.save()
}

// Exists 检查成员是否存在
func (s *MemberStore) Exists(memberID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.members[memberID]
	return exists
}

// Count 获取成员数量
func (s *MemberStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.members)
}

// OnlineCount 获取在线成员数量
func (s *MemberStore) OnlineCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, member := range s.members {
		if member.IsOnline {
			count++
		}
	}
	return count
}

// AddLeader 添加 Leader 作为成员
func (s *MemberStore) AddLeader(id, name, endpoint string) error {
	return s.Add(&domainTeam.TeamMember{
		ID:       id,
		Name:     name,
		Endpoint: endpoint,
		IsLeader: true,
		IsOnline: true,
		JoinedAt: time.Now(),
	})
}

// Clear 清空所有成员
func (s *MemberStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.members = make(map[string]*domainTeam.TeamMember)
	return s.save()
}
