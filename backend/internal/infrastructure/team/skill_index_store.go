package team

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cocursor/backend/internal/infrastructure/config"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

// SkillIndexStore 技能目录存储
type SkillIndexStore struct {
	mu       sync.RWMutex
	teamID   string
	filePath string
	index    *domainTeam.TeamSkillIndex
}

// NewSkillIndexStore 创建技能目录存储
func NewSkillIndexStore(teamID string) (*SkillIndexStore, error) {
	filePath := filepath.Join(config.GetDataDir(), "team", teamID, "skill-index.json")

	store := &SkillIndexStore{
		teamID:   teamID,
		filePath: filePath,
	}

	// 尝试加载现有数据
	index, err := store.load()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if index != nil {
		store.index = index
	} else {
		store.index = &domainTeam.TeamSkillIndex{
			TeamID:    teamID,
			UpdatedAt: time.Now(),
			Skills:    []domainTeam.TeamSkillEntry{},
		}
	}

	return store, nil
}

// load 从文件加载
func (s *SkillIndexStore) load() (*domainTeam.TeamSkillIndex, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	var index domainTeam.TeamSkillIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse skill index file: %w", err)
	}

	return &index, nil
}

// save 保存到文件
func (s *SkillIndexStore) save() error {
	// 确保目录存在
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(s.index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal skill index: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write skill index file: %w", err)
	}

	return nil
}

// Get 获取完整的技能目录
func (s *SkillIndexStore) Get() *domainTeam.TeamSkillIndex {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回深拷贝
	indexCopy := domainTeam.TeamSkillIndex{
		TeamID:    s.index.TeamID,
		UpdatedAt: s.index.UpdatedAt,
		Skills:    make([]domainTeam.TeamSkillEntry, len(s.index.Skills)),
	}
	copy(indexCopy.Skills, s.index.Skills)

	return &indexCopy
}

// GetSkill 获取指定技能
func (s *SkillIndexStore) GetSkill(pluginID string) *domainTeam.TeamSkillEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry := s.index.FindSkill(pluginID)
	if entry == nil {
		return nil
	}

	// 返回副本
	entryCopy := *entry
	return &entryCopy
}

// List 获取技能列表
func (s *SkillIndexStore) List() []domainTeam.TeamSkillEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skills := make([]domainTeam.TeamSkillEntry, len(s.index.Skills))
	copy(skills, s.index.Skills)
	return skills
}

// AddOrUpdate 添加或更新技能
func (s *SkillIndexStore) AddOrUpdate(entry domainTeam.TeamSkillEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.index.AddOrUpdateSkill(entry)
	return s.save()
}

// Remove 移除技能
func (s *SkillIndexStore) Remove(pluginID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.index.RemoveSkill(pluginID)
	return s.save()
}

// Replace 替换整个技能目录（用于同步）
func (s *SkillIndexStore) Replace(index *domainTeam.TeamSkillIndex) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.index = &domainTeam.TeamSkillIndex{
		TeamID:    s.teamID,
		UpdatedAt: index.UpdatedAt,
		Skills:    make([]domainTeam.TeamSkillEntry, len(index.Skills)),
	}
	copy(s.index.Skills, index.Skills)

	return s.save()
}

// Count 获取技能数量
func (s *SkillIndexStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.index.Skills)
}

// GetLastUpdated 获取最后更新时间
func (s *SkillIndexStore) GetLastUpdated() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.index.UpdatedAt
}

// Clear 清空技能目录
func (s *SkillIndexStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.index.Skills = []domainTeam.TeamSkillEntry{}
	s.index.UpdatedAt = time.Now()

	return s.save()
}

// FindByAuthor 按作者查找技能
func (s *SkillIndexStore) FindByAuthor(authorID string) []domainTeam.TeamSkillEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var skills []domainTeam.TeamSkillEntry
	for _, skill := range s.index.Skills {
		if skill.AuthorID == authorID {
			skills = append(skills, skill)
		}
	}
	return skills
}
