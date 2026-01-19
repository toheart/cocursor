package cursor

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
)

// DataMerger 数据合并器
type DataMerger struct {
	dbReader      *infraCursor.DBReader
	globalDBReader domainCursor.GlobalDBReader
}

// NewDataMerger 创建数据合并器实例
func NewDataMerger(
	dbReader *infraCursor.DBReader,
	globalDBReader domainCursor.GlobalDBReader,
) *DataMerger {
	return &DataMerger{
		dbReader:      dbReader,
		globalDBReader: globalDBReader,
	}
}

// MergeAcceptanceStats 合并接受率统计（累加日期范围内的所有数据）
// 注意：接受率统计存储在全局存储中，不是按工作区存储
// 这里合并的是日期范围内的所有统计数据
func (m *DataMerger) MergeAcceptanceStats(startDate, endDate string) (*domainCursor.DailyAcceptanceStats, []*domainCursor.DailyAcceptanceStats, error) {
	// 解析日期范围
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid start_date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid end_date: %w", err)
	}

	// 收集日期范围内的所有统计数据
	var rawStats []*domainCursor.DailyAcceptanceStats
	merged := &domainCursor.DailyAcceptanceStats{
		Date: startDate,
	}

	// 遍历日期范围
	current := start
	foundCount := 0
	for !current.After(end) {
		dateStr := current.Format("2006-01-02")
		key := fmt.Sprintf("aiCodeTracking.dailyStats.v1.5.%s", dateStr)

		// 使用 GlobalDBReader 读取数据
		value, err := m.globalDBReader.ReadValue(key)
		if err != nil {
			// 如果读取失败（可能是该日期没有数据），跳过该日期
			// 注意：key not found 是正常情况（某些日期可能没有数据）
			current = current.AddDate(0, 0, 1)
			continue
		}

		// 解析数据
		stats, err := domainCursor.ParseAcceptanceStats(string(value), dateStr)
		if err != nil {
			log.Printf("[MergeAcceptanceStats] Failed to parse stats for %s: %v", dateStr, err)
			current = current.AddDate(0, 0, 1)
			continue
		}

		rawStats = append(rawStats, stats)
		foundCount++

		// 累加统计数据
		merged.TabSuggestedLines += stats.TabSuggestedLines
		merged.TabAcceptedLines += stats.TabAcceptedLines
		merged.ComposerSuggestedLines += stats.ComposerSuggestedLines
		merged.ComposerAcceptedLines += stats.ComposerAcceptedLines

		current = current.AddDate(0, 0, 1)
	}

	// 记录找到的数据数量
	if foundCount == 0 {
		log.Printf("[MergeAcceptanceStats] No acceptance stats found for date range %s to %s", startDate, endDate)
	} else {
		log.Printf("[MergeAcceptanceStats] Found %d days of data, merged: TabSuggested=%d, TabAccepted=%d, ComposerSuggested=%d, ComposerAccepted=%d",
			foundCount, merged.TabSuggestedLines, merged.TabAcceptedLines, merged.ComposerSuggestedLines, merged.ComposerAcceptedLines)
	}

	// 重新计算接受率
	merged.CalculateAcceptanceRate()

	return merged, rawStats, nil
}

// MergePrompts 合并 Prompts（按时间排序，不合并）
func (m *DataMerger) MergePrompts(workspaces []*domainCursor.WorkspaceInfo) ([]PromptWithSource, error) {
	pathResolver := infraCursor.NewPathResolver()

	var allPrompts []PromptWithSource

	// 从每个工作区读取 Prompts
	for _, ws := range workspaces {
		workspaceDBPath, err := pathResolver.GetWorkspaceDBPath(ws.WorkspaceID)
		if err != nil {
			continue
		}

		// 读取 prompts
		promptsValue, err := m.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "aiService.prompts")
		if err != nil || len(promptsValue) == 0 {
			continue
		}

		// 解析 prompts
		var prompts []map[string]interface{}
		if err := json.Unmarshal(promptsValue, &prompts); err != nil {
			continue
		}

		// 转换为带来源的 Prompt
		for _, p := range prompts {
			allPrompts = append(allPrompts, PromptWithSource{
				Prompt: p,
				Source: ws.WorkspaceID,
			})
		}
	}

	// 按时间排序（如果有时间戳字段）
	// 注意：prompts 可能没有时间戳，这里先按添加顺序排序
	sort.Slice(allPrompts, func(i, j int) bool {
		// 如果没有时间戳，保持原顺序
		return false
	})

	return allPrompts, nil
}

// MergeGenerations 合并 Generations（按时间排序，不合并）
func (m *DataMerger) MergeGenerations(workspaces []*domainCursor.WorkspaceInfo) ([]GenerationWithSource, error) {
	pathResolver := infraCursor.NewPathResolver()

	var allGenerations []GenerationWithSource

	// 从每个工作区读取 Generations
	for _, ws := range workspaces {
		workspaceDBPath, err := pathResolver.GetWorkspaceDBPath(ws.WorkspaceID)
		if err != nil {
			continue
		}

		// 读取 generations
		generationsValue, err := m.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "aiService.generations")
		if err != nil || len(generationsValue) == 0 {
			continue
		}

		// 解析 generations
		generations, err := domainCursor.ParseGenerationsData(string(generationsValue))
		if err != nil {
			continue
		}

		// 转换为带来源的 Generation
		for i := range generations {
			allGenerations = append(allGenerations, GenerationWithSource{
				GenerationData: &generations[i],
				Source:         ws.WorkspaceID,
			})
		}
	}

	// 按时间戳排序（最新的在前）
	sort.Slice(allGenerations, func(i, j int) bool {
		if allGenerations[i].GenerationData == nil || allGenerations[j].GenerationData == nil {
			return false
		}
		return allGenerations[i].GenerationData.UnixMs > allGenerations[j].GenerationData.UnixMs
	})

	return allGenerations, nil
}

// MergeSessions 合并 Composer Sessions（按时间排序，不合并）
func (m *DataMerger) MergeSessions(workspaces []*domainCursor.WorkspaceInfo) ([]SessionWithSource, error) {
	pathResolver := infraCursor.NewPathResolver()

	var allSessions []SessionWithSource

	// 从每个工作区读取 Composer 数据
	for _, ws := range workspaces {
		workspaceDBPath, err := pathResolver.GetWorkspaceDBPath(ws.WorkspaceID)
		if err != nil {
			continue
		}

		// 读取 composer data
		composerValue, err := m.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
		if err != nil || len(composerValue) == 0 {
			continue
		}

		// 解析 composer data
		composers, err := domainCursor.ParseComposerData(string(composerValue))
		if err != nil {
			continue
		}

		// 转换为带来源的 Session
		for i := range composers {
			allSessions = append(allSessions, SessionWithSource{
				ComposerData: &composers[i],
				Source:       ws.WorkspaceID,
			})
		}
	}

	// 按最后更新时间排序（最新的在前）
	sort.Slice(allSessions, func(i, j int) bool {
		if allSessions[i].ComposerData == nil || allSessions[j].ComposerData == nil {
			return false
		}
		return allSessions[i].ComposerData.LastUpdatedAt > allSessions[j].ComposerData.LastUpdatedAt
	})

	return allSessions, nil
}

// PromptWithSource 带来源的 Prompt
type PromptWithSource struct {
	Prompt map[string]interface{} `json:"prompt"`
	Source string                  `json:"source"` // 工作区 ID
}

// GenerationWithSource 带来源的 Generation
type GenerationWithSource struct {
	*domainCursor.GenerationData
	Source string `json:"source"` // 工作区 ID
}

// SessionWithSource 带来源的 Session
type SessionWithSource struct {
	*domainCursor.ComposerData
	Source string `json:"source"` // 工作区 ID
}
