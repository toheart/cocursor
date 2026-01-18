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
			name: "建议行数为0时接受率为0",
			stats: DailyAcceptanceStats{
				TabSuggestedLines:      0,
				TabAcceptedLines:        10,
				ComposerSuggestedLines:  0,
				ComposerAcceptedLines:   20,
			},
			expectedTabRate:      0.0,
			expectedComposerRate: 0.0,
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
