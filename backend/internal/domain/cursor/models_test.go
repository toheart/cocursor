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
