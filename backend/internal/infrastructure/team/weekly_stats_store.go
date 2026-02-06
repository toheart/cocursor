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

// 默认缓存过期时间：1 小时
const DefaultCacheExpiration = time.Hour

// WeeklyStatsStore 周统计缓存存储
type WeeklyStatsStore struct {
	mu       sync.RWMutex
	teamID   string
	filePath string
	cache    *domainTeam.WeeklyStatsCache
}

// NewWeeklyStatsStore 创建周统计缓存存储
func NewWeeklyStatsStore(teamID string) (*WeeklyStatsStore, error) {
	filePath := filepath.Join(config.GetDataDir(), "team", teamID, "weekly_stats_cache.json")

	store := &WeeklyStatsStore{
		teamID:   teamID,
		filePath: filePath,
	}

	// 尝试加载现有缓存
	cache, err := store.load()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if cache == nil {
		// 初始化空缓存
		cache = &domainTeam.WeeklyStatsCache{
			TeamID:    teamID,
			Entries:   []domainTeam.WeeklyStatsCacheEntry{},
			UpdatedAt: time.Now(),
		}
	}
	store.cache = cache

	return store, nil
}

// load 从文件加载缓存
func (s *WeeklyStatsStore) load() (*domainTeam.WeeklyStatsCache, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	var cache domainTeam.WeeklyStatsCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse weekly stats cache file: %w", err)
	}

	return &cache, nil
}

// save 保存缓存到文件
func (s *WeeklyStatsStore) save() error {
	// 确保目录存在
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(s.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal weekly stats cache: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write weekly stats cache file: %w", err)
	}

	return nil
}

// Get 获取成员周统计数据
// 返回缓存条目和是否过期
func (s *WeeklyStatsStore) Get(memberID, weekStart string) (*domainTeam.MemberWeeklyStats, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry := s.cache.GetEntry(memberID, weekStart)
	if entry == nil {
		return nil, false, nil
	}

	// 返回副本
	statsCopy := entry.Stats
	return &statsCopy, entry.IsExpired(), nil
}

// Set 设置成员周统计数据
func (s *WeeklyStatsStore) Set(stats *domainTeam.MemberWeeklyStats) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	entry := domainTeam.WeeklyStatsCacheEntry{
		MemberID:  stats.MemberID,
		WeekStart: stats.WeekStart,
		Stats:     *stats,
		CachedAt:  now,
		ExpireAt:  now.Add(DefaultCacheExpiration),
	}

	s.cache.SetEntry(entry)

	return s.save()
}

// SetWithExpiration 设置成员周统计数据（自定义过期时间）
func (s *WeeklyStatsStore) SetWithExpiration(stats *domainTeam.MemberWeeklyStats, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	entry := domainTeam.WeeklyStatsCacheEntry{
		MemberID:  stats.MemberID,
		WeekStart: stats.WeekStart,
		Stats:     *stats,
		CachedAt:  now,
		ExpireAt:  now.Add(expiration),
	}

	s.cache.SetEntry(entry)

	return s.save()
}

// GetAll 获取所有成员的周统计数据（指定周）
func (s *WeeklyStatsStore) GetAll(weekStart string) ([]domainTeam.MemberWeeklyStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []domainTeam.MemberWeeklyStats
	for _, entry := range s.cache.Entries {
		if entry.WeekStart == weekStart {
			result = append(result, entry.Stats)
		}
	}
	return result, nil
}

// GetExpiredMembers 获取缓存过期的成员 ID 列表
func (s *WeeklyStatsStore) GetExpiredMembers(weekStart string, memberIDs []string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var expired []string
	memberSet := make(map[string]bool)
	for _, id := range memberIDs {
		memberSet[id] = true
	}

	for _, entry := range s.cache.Entries {
		if entry.WeekStart == weekStart && memberSet[entry.MemberID] {
			if entry.IsExpired() {
				expired = append(expired, entry.MemberID)
			}
			delete(memberSet, entry.MemberID)
		}
	}

	// 未缓存的成员也视为"过期"
	for id := range memberSet {
		expired = append(expired, id)
	}

	return expired
}

// CleanExpired 清理过期条目
func (s *WeeklyStatsStore) CleanExpired() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := s.cache.CleanExpired()
	if removed > 0 {
		if err := s.save(); err != nil {
			return 0, err
		}
	}
	return removed, nil
}

// Delete 删除指定成员的缓存
func (s *WeeklyStatsStore) Delete(memberID, weekStart string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var newEntries []domainTeam.WeeklyStatsCacheEntry
	for _, entry := range s.cache.Entries {
		if entry.MemberID != memberID || entry.WeekStart != weekStart {
			newEntries = append(newEntries, entry)
		}
	}

	if len(newEntries) != len(s.cache.Entries) {
		s.cache.Entries = newEntries
		s.cache.UpdatedAt = time.Now()
		return s.save()
	}

	return nil
}

// Clear 清空所有缓存
func (s *WeeklyStatsStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache.Entries = []domainTeam.WeeklyStatsCacheEntry{}
	s.cache.UpdatedAt = time.Now()
	return s.save()
}
