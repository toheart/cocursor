package cursor

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// ParseComposerData 解析 composer.composerData 的 JSON 字符串
// rawJson: 从数据库读取的原始 JSON 字符串
// 返回: ComposerData 切片和错误
func ParseComposerData(rawJson string) ([]ComposerData, error) {
	if rawJson == "" {
		return nil, fmt.Errorf("raw JSON is empty")
	}

	var data ComposerDataList
	if err := json.Unmarshal([]byte(rawJson), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return data.AllComposers, nil
}

// ParseGenerationsData 解析 aiService.generations 的 JSON 字符串
// rawJson: 从数据库读取的原始 JSON 字符串
// 返回: GenerationData 切片和错误
func ParseGenerationsData(rawJson string) ([]GenerationData, error) {
	if rawJson == "" {
		return nil, fmt.Errorf("raw JSON is empty")
	}

	var generations []GenerationData
	if err := json.Unmarshal([]byte(rawJson), &generations); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return generations, nil
}

// GetActiveComposer 获取当前活跃的 Composer（最近更新的）
// composers: ComposerData 切片
// 返回: 最活跃的 Composer，如果没有则返回 nil
func GetActiveComposer(composers []ComposerData) *ComposerData {
	if len(composers) == 0 {
		return nil
	}

	// 过滤掉已归档的会话
	activeComposers := make([]ComposerData, 0)
	for _, c := range composers {
		if !c.IsArchived {
			activeComposers = append(activeComposers, c)
		}
	}

	if len(activeComposers) == 0 {
		return nil
	}

	// 按最后更新时间排序，返回最新的
	sort.Slice(activeComposers, func(i, j int) bool {
		return activeComposers[i].LastUpdatedAt > activeComposers[j].LastUpdatedAt
	})

	return &activeComposers[0]
}

// GetLatestGenerationTime 获取最近一次 AI 生成的时间
// generations: GenerationData 切片
// 返回: 最近一次生成的时间，如果没有则返回零值
func GetLatestGenerationTime(generations []GenerationData) time.Time {
	if len(generations) == 0 {
		return time.Time{}
	}

	// 按时间戳排序
	sort.Slice(generations, func(i, j int) bool {
		return generations[i].UnixMs > generations[j].UnixMs
	})

	// 返回最新的时间戳
	latest := generations[0]
	return time.Unix(0, latest.UnixMs*int64(time.Millisecond))
}

// GetTimeSinceLastGeneration 计算最近一次 AI 生成与当前时间的差值（分钟）
// generations: GenerationData 切片
// 返回: 时间差（分钟），如果没有生成记录则返回 -1
func GetTimeSinceLastGeneration(generations []GenerationData) float64 {
	latestTime := GetLatestGenerationTime(generations)
	if latestTime.IsZero() {
		return -1
	}

	now := time.Now()
	duration := now.Sub(latestTime)
	return duration.Minutes()
}

// ParseAcceptanceStats 解析 aiCodeTracking.dailyStats 的 JSON 字符串
// rawJson: 从数据库读取的原始 JSON 字符串
// date: 日期字符串 YYYY-MM-DD
// 返回: DailyAcceptanceStats 和错误
func ParseAcceptanceStats(rawJson string, date string) (*DailyAcceptanceStats, error) {
	if rawJson == "" {
		return nil, fmt.Errorf("raw JSON is empty")
	}

	var stats DailyAcceptanceStats
	if err := json.Unmarshal([]byte(rawJson), &stats); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// 设置日期
	stats.Date = date

	// 计算接受率
	stats.CalculateAcceptanceRate()

	return &stats, nil
}
