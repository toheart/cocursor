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

// DailySummaryIndexStore 日报索引存储
type DailySummaryIndexStore struct {
	mu       sync.RWMutex
	teamID   string
	filePath string
}

// NewDailySummaryIndexStore 创建日报索引存储
func NewDailySummaryIndexStore(teamID string) (*DailySummaryIndexStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	filePath := filepath.Join(homeDir, ".cocursor", "team", teamID, "daily-summaries-index.json")

	store := &DailySummaryIndexStore{
		teamID:   teamID,
		filePath: filePath,
	}

	return store, nil
}

// Load 加载日报索引
func (s *DailySummaryIndexStore) Load() (*domainTeam.TeamDailySummaryIndex, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 返回空索引
			return &domainTeam.TeamDailySummaryIndex{
				TeamID:    s.teamID,
				UpdatedAt: time.Now(),
				Summaries: []domainTeam.TeamDailySummary{},
			}, nil
		}
		return nil, err
	}

	var index domainTeam.TeamDailySummaryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse daily summary index file: %w", err)
	}

	return &index, nil
}

// Save 保存日报索引
func (s *DailySummaryIndexStore) Save(index *domainTeam.TeamDailySummaryIndex) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确保目录存在
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 清理过期的日报（保留最近 30 天）
	s.cleanOldSummaries(index, 30)

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal daily summary index: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write daily summary index file: %w", err)
	}

	return nil
}

// cleanOldSummaries 清理过期的日报
func (s *DailySummaryIndexStore) cleanOldSummaries(index *domainTeam.TeamDailySummaryIndex, retentionDays int) {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays).Format("2006-01-02")

	var newSummaries []domainTeam.TeamDailySummary
	for _, summary := range index.Summaries {
		if summary.Date >= cutoffDate {
			newSummaries = append(newSummaries, summary)
		}
	}

	index.Summaries = newSummaries
}

// GetSummariesByDate 按日期获取日报列表
func (s *DailySummaryIndexStore) GetSummariesByDate(date string) ([]domainTeam.TeamDailySummary, error) {
	index, err := s.Load()
	if err != nil {
		return nil, err
	}

	return index.GetSummariesByDate(date), nil
}

// AddOrUpdateSummary 添加或更新日报
func (s *DailySummaryIndexStore) AddOrUpdateSummary(entry domainTeam.TeamDailySummary) error {
	index, err := s.Load()
	if err != nil {
		return err
	}

	index.AddOrUpdateSummary(entry)
	return s.Save(index)
}

// GetRecentDates 获取最近有日报的日期列表
func (s *DailySummaryIndexStore) GetRecentDates(limit int) ([]string, error) {
	index, err := s.Load()
	if err != nil {
		return nil, err
	}

	dateSet := make(map[string]bool)
	for _, summary := range index.Summaries {
		dateSet[summary.Date] = true
	}

	var dates []string
	for date := range dateSet {
		dates = append(dates, date)
	}

	// 按日期降序排序
	for i := 0; i < len(dates)-1; i++ {
		for j := i + 1; j < len(dates); j++ {
			if dates[i] < dates[j] {
				dates[i], dates[j] = dates[j], dates[i]
			}
		}
	}

	if limit > 0 && len(dates) > limit {
		dates = dates[:limit]
	}

	return dates, nil
}

// GetMemberSummaryDates 获取成员有日报的日期列表
func (s *DailySummaryIndexStore) GetMemberSummaryDates(memberID string) ([]string, error) {
	index, err := s.Load()
	if err != nil {
		return nil, err
	}

	var dates []string
	for _, summary := range index.Summaries {
		if summary.MemberID == memberID {
			dates = append(dates, summary.Date)
		}
	}

	return dates, nil
}
