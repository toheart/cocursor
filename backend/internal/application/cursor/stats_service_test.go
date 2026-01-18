package cursor

import (
	"testing"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/stretchr/testify/assert"
)

func TestStatsService_CalculateSessionEntropy(t *testing.T) {
	service := NewStatsService()

	tests := []struct {
		name     string
		data     domainCursor.ComposerData
		expected float64
	}{
		{
			name: "高复杂度会话",
			data: domainCursor.ComposerData{
				ContextUsagePercent: 65.186,
				FilesChangedCount:   39,
				TotalLinesAdded:     1704,
				TotalLinesRemoved:   116,
			},
			expected: 0, // 我们只验证计算逻辑，不验证具体值
		},
		{
			name: "低复杂度会话",
			data: domainCursor.ComposerData{
				ContextUsagePercent: 7.094,
				FilesChangedCount:   1,
				TotalLinesAdded:     20,
				TotalLinesRemoved:   5,
			},
			expected: 0,
		},
		{
			name: "零变更行数",
			data: domainCursor.ComposerData{
				ContextUsagePercent: 10.0,
				FilesChangedCount:   2,
				TotalLinesAdded:     0,
				TotalLinesRemoved:   0,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entropy := service.CalculateSessionEntropy(tt.data)
			// 验证熵值大于等于0（应该是正数）
			assert.GreaterOrEqual(t, entropy, 0.0, "熵值应该大于等于0")
			// 验证高复杂度会话的熵值大于低复杂度会话
			if tt.name == "高复杂度会话" {
				lowComplexity := domainCursor.ComposerData{
					ContextUsagePercent: 7.094,
					FilesChangedCount:   1,
					TotalLinesAdded:     20,
					TotalLinesRemoved:   5,
				}
				lowEntropy := service.CalculateSessionEntropy(lowComplexity)
				assert.Greater(t, entropy, lowEntropy, "高复杂度会话的熵值应该大于低复杂度会话")
			}
		})
	}
}

func TestStatsService_GetHealthStatus(t *testing.T) {
	service := NewStatsService()

	tests := []struct {
		name           string
		entropy        float64
		expectedStatus HealthStatus
		hasWarning     bool
	}{
		{
			name:           "健康状态",
			entropy:        30.0,
			expectedStatus: HealthStatusHealthy,
			hasWarning:     false,
		},
		{
			name:           "亚健康状态",
			entropy:        50.0,
			expectedStatus: HealthStatusSubHealthy,
			hasWarning:     true,
		},
		{
			name:           "危险状态",
			entropy:        80.0,
			expectedStatus: HealthStatusDangerous,
			hasWarning:     true,
		},
		{
			name:           "边界值-健康",
			entropy:        39.9,
			expectedStatus: HealthStatusHealthy,
			hasWarning:     false,
		},
		{
			name:           "边界值-亚健康",
			entropy:        40.0,
			expectedStatus: HealthStatusSubHealthy,
			hasWarning:     true,
		},
		{
			name:           "边界值-危险",
			entropy:        70.0,
			expectedStatus: HealthStatusDangerous,
			hasWarning:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, warning := service.GetHealthStatus(tt.entropy)
			assert.Equal(t, tt.expectedStatus, status, "健康状态应该正确")
			if tt.hasWarning {
				assert.NotEmpty(t, warning, "应该有警告信息")
			} else {
				assert.Empty(t, warning, "不应该有警告信息")
			}
		})
	}
}
