package cursor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDailyAcceptanceStats_CalculateAcceptanceRate(t *testing.T) {
	tests := []struct {
		name                    string
		stats                   DailyAcceptanceStats
		expectedTabRate         float64
		expectedComposerRate    float64
	}{
		{
			name: "正常计算接受率",
			stats: DailyAcceptanceStats{
				TabSuggestedLines:      100,
				TabAcceptedLines:        50,
				ComposerSuggestedLines:  200,
				ComposerAcceptedLines:   150,
			},
			expectedTabRate:      50.0,
			expectedComposerRate: 75.0,
		},
		{
			name: "建议行数为0但接受行数>0时返回-1表示N/A",
			stats: DailyAcceptanceStats{
				TabSuggestedLines:      0,
				TabAcceptedLines:        10,
				ComposerSuggestedLines:  0,
				ComposerAcceptedLines:   20,
			},
			expectedTabRate:      0.0, // Tab 建议0接受10，Tab仍然返回0（只有Composer有特殊处理）
			expectedComposerRate: -1.0, // Composer 建议0接受20，返回-1表示N/A
		},
		{
			name: "完全接受",
			stats: DailyAcceptanceStats{
				TabSuggestedLines:      100,
				TabAcceptedLines:        100,
				ComposerSuggestedLines:  50,
				ComposerAcceptedLines:   50,
			},
			expectedTabRate:      100.0,
			expectedComposerRate: 100.0,
		},
		{
			name: "完全拒绝",
			stats: DailyAcceptanceStats{
				TabSuggestedLines:      100,
				TabAcceptedLines:        0,
				ComposerSuggestedLines:  50,
				ComposerAcceptedLines:   0,
			},
			expectedTabRate:      0.0,
			expectedComposerRate: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.stats.CalculateAcceptanceRate()
			assert.InDelta(t, tt.expectedTabRate, tt.stats.TabAcceptanceRate, 0.01, "Tab 接受率应该正确计算")
			assert.InDelta(t, tt.expectedComposerRate, tt.stats.ComposerAcceptanceRate, 0.01, "Composer 接受率应该正确计算")
		})
	}
}

func TestDailyAcceptanceStats_GetOverallAcceptanceRate(t *testing.T) {
	tests := []struct {
		name         string
		stats        DailyAcceptanceStats
		expectedRate float64
	}{
		{
			name: "正常计算整体接受率",
			stats: DailyAcceptanceStats{
				TabSuggestedLines:      100,
				TabAcceptedLines:       50,
				ComposerSuggestedLines: 100,
				ComposerAcceptedLines:  50,
			},
			expectedRate: 50.0,
		},
		{
			name: "只有Tab有数据",
			stats: DailyAcceptanceStats{
				TabSuggestedLines:      100,
				TabAcceptedLines:       60,
				ComposerSuggestedLines: 0,
				ComposerAcceptedLines:  0,
			},
			expectedRate: 60.0,
		},
		{
			name: "只有Composer有数据",
			stats: DailyAcceptanceStats{
				TabSuggestedLines:      0,
				TabAcceptedLines:       0,
				ComposerSuggestedLines: 100,
				ComposerAcceptedLines:  80,
			},
			expectedRate: 80.0,
		},
		{
			name: "Composer建议为0但接受>0时只使用Tab",
			stats: DailyAcceptanceStats{
				TabSuggestedLines:      100,
				TabAcceptedLines:       70,
				ComposerSuggestedLines: 0,
				ComposerAcceptedLines:  50,
			},
			expectedRate: 70.0, // 只使用Tab的接受率
		},
		{
			name: "都没有有效数据返回-1",
			stats: DailyAcceptanceStats{
				TabSuggestedLines:      0,
				TabAcceptedLines:       10,
				ComposerSuggestedLines: 0,
				ComposerAcceptedLines:  20,
			},
			expectedRate: -1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.stats.CalculateAcceptanceRate()
			rate := tt.stats.GetOverallAcceptanceRate()
			assert.InDelta(t, tt.expectedRate, rate, 0.01, "整体接受率应该正确计算")
		})
	}
}

// TestCalculateHealthStatus 测试健康状态计算逻辑
func TestCalculateHealthStatus(t *testing.T) {
	tests := []struct {
		name                string
		entropy             float64
		contextUsagePercent float64
		expectedStatus      HealthStatus
		expectWarning       bool
	}{
		{
			name:                "健康状态-低熵值低上下文",
			entropy:             20,
			contextUsagePercent: 30,
			expectedStatus:      HealthStatusHealthy,
			expectWarning:       false,
		},
		{
			name:                "健康状态-边界值",
			entropy:             39.9,
			contextUsagePercent: 59.9,
			expectedStatus:      HealthStatusHealthy,
			expectWarning:       false,
		},
		{
			name:                "警告状态-熵值中等",
			entropy:             50,
			contextUsagePercent: 30,
			expectedStatus:      HealthStatusWarning,
			expectWarning:       true,
		},
		{
			name:                "警告状态-上下文中等",
			entropy:             30,
			contextUsagePercent: 70,
			expectedStatus:      HealthStatusWarning,
			expectWarning:       true,
		},
		{
			name:                "警告状态-边界值熵值40",
			entropy:             40,
			contextUsagePercent: 50,
			expectedStatus:      HealthStatusWarning,
			expectWarning:       true,
		},
		{
			name:                "警告状态-边界值上下文60",
			entropy:             30,
			contextUsagePercent: 60,
			expectedStatus:      HealthStatusWarning,
			expectWarning:       true,
		},
		{
			name:                "危险状态-熵值过高",
			entropy:             75,
			contextUsagePercent: 50,
			expectedStatus:      HealthStatusCritical,
			expectWarning:       true,
		},
		{
			name:                "危险状态-上下文过高",
			entropy:             30,
			contextUsagePercent: 85,
			expectedStatus:      HealthStatusCritical,
			expectWarning:       true,
		},
		{
			name:                "危险状态-边界值熵值70.1",
			entropy:             70.1,
			contextUsagePercent: 50,
			expectedStatus:      HealthStatusCritical,
			expectWarning:       true,
		},
		{
			name:                "危险状态-边界值上下文80.1",
			entropy:             30,
			contextUsagePercent: 80.1,
			expectedStatus:      HealthStatusCritical,
			expectWarning:       true,
		},
		{
			name:                "危险状态-两者都高",
			entropy:             80,
			contextUsagePercent: 90,
			expectedStatus:      HealthStatusCritical,
			expectWarning:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, warning := CalculateHealthStatus(tt.entropy, tt.contextUsagePercent)
			assert.Equal(t, tt.expectedStatus, status, "健康状态应该正确")
			if tt.expectWarning {
				assert.NotEmpty(t, warning, "应该有警告信息")
			} else {
				assert.Empty(t, warning, "不应该有警告信息")
			}
		})
	}
}

// TestCalculateActiveLevel 测试活跃等级计算逻辑
func TestCalculateActiveLevel(t *testing.T) {
	tests := []struct {
		name          string
		isArchived    bool
		isVisible     bool
		isFocused     bool
		expectedLevel int
	}{
		{
			name:          "聚焦状态",
			isArchived:    false,
			isVisible:     true,
			isFocused:     true,
			expectedLevel: ActiveLevelFocused,
		},
		{
			name:          "聚焦状态-即使已归档",
			isArchived:    true,
			isVisible:     true,
			isFocused:     true,
			expectedLevel: ActiveLevelFocused,
		},
		{
			name:          "打开状态",
			isArchived:    false,
			isVisible:     true,
			isFocused:     false,
			expectedLevel: ActiveLevelOpen,
		},
		{
			name:          "打开状态-可见优先于归档",
			isArchived:    true,
			isVisible:     true,
			isFocused:     false,
			expectedLevel: ActiveLevelOpen,
		},
		{
			name:          "归档状态",
			isArchived:    true,
			isVisible:     false,
			isFocused:     false,
			expectedLevel: ActiveLevelArchived,
		},
		{
			name:          "关闭状态",
			isArchived:    false,
			isVisible:     false,
			isFocused:     false,
			expectedLevel: ActiveLevelClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := CalculateActiveLevel(tt.isArchived, tt.isVisible, tt.isFocused)
			assert.Equal(t, tt.expectedLevel, level, "活跃等级应该正确")
		})
	}
}

// TestCalculateEntropy 测试熵值计算逻辑
func TestCalculateEntropy(t *testing.T) {
	tests := []struct {
		name                string
		linesAdded          int
		linesRemoved        int
		filesChanged        int
		contextUsagePercent float64
		minExpected         float64
		maxExpected         float64
	}{
		{
			name:                "零值输入",
			linesAdded:          0,
			linesRemoved:        0,
			filesChanged:        0,
			contextUsagePercent: 0,
			minExpected:         0,
			maxExpected:         0,
		},
		{
			name:                "仅有代码变更",
			linesAdded:          100,
			linesRemoved:        50,
			filesChanged:        0,
			contextUsagePercent: 0,
			minExpected:         10,
			maxExpected:         15,
		},
		{
			name:                "仅有文件变更",
			linesAdded:          0,
			linesRemoved:        0,
			filesChanged:        5,
			contextUsagePercent: 0,
			minExpected:         10,
			maxExpected:         20,
		},
		{
			name:                "仅有上下文使用",
			linesAdded:          0,
			linesRemoved:        0,
			filesChanged:        0,
			contextUsagePercent: 50,
			minExpected:         10,
			maxExpected:         20,
		},
		{
			name:                "综合较低活动",
			linesAdded:          50,
			linesRemoved:        30,
			filesChanged:        2,
			contextUsagePercent: 20,
			minExpected:         10,
			maxExpected:         25,
		},
		{
			name:                "综合中等活动",
			linesAdded:          200,
			linesRemoved:        100,
			filesChanged:        5,
			contextUsagePercent: 50,
			minExpected:         35,
			maxExpected:         60,
		},
		{
			name:                "综合高活动",
			linesAdded:          400,
			linesRemoved:        200,
			filesChanged:        10,
			contextUsagePercent: 80,
			minExpected:         75,
			maxExpected:         100,
		},
		{
			name:                "超出阈值-代码行数封顶",
			linesAdded:          1000,
			linesRemoved:        500,
			filesChanged:        20,
			contextUsagePercent: 100,
			minExpected:         95,
			maxExpected:         100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entropy := CalculateEntropy(tt.linesAdded, tt.linesRemoved, tt.filesChanged, tt.contextUsagePercent)
			assert.GreaterOrEqual(t, entropy, tt.minExpected, "熵值应该 >= 最小期望值")
			assert.LessOrEqual(t, entropy, tt.maxExpected, "熵值应该 <= 最大期望值")
		})
	}
}

// TestActiveLevelConstants 测试活跃等级常量值
func TestActiveLevelConstants(t *testing.T) {
	// 确保常量值按预期递增
	assert.Equal(t, 0, ActiveLevelFocused, "聚焦等级应为 0")
	assert.Equal(t, 1, ActiveLevelOpen, "打开等级应为 1")
	assert.Equal(t, 2, ActiveLevelClosed, "关闭等级应为 2")
	assert.Equal(t, 3, ActiveLevelArchived, "归档等级应为 3")

	// 确保排序顺序：Focused < Open < Closed < Archived
	assert.Less(t, ActiveLevelFocused, ActiveLevelOpen, "聚焦 < 打开")
	assert.Less(t, ActiveLevelOpen, ActiveLevelClosed, "打开 < 关闭")
	assert.Less(t, ActiveLevelClosed, ActiveLevelArchived, "关闭 < 归档")
}

// TestHealthStatusConstants 测试健康状态常量值
func TestHealthStatusConstants(t *testing.T) {
	assert.Equal(t, HealthStatus("healthy"), HealthStatusHealthy)
	assert.Equal(t, HealthStatus("warning"), HealthStatusWarning)
	assert.Equal(t, HealthStatus("critical"), HealthStatusCritical)
}
