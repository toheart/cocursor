package cursor

import (
	"math"

	"github.com/cocursor/backend/internal/domain/cursor"
)

// EntropyService 会话熵计算服务
type EntropyService struct{}

// NewEntropyService 创建熵计算服务实例
func NewEntropyService() *EntropyService {
	return &EntropyService{}
}

// CalculateSessionEntropy 计算会话熵值
// 公式：Score = (ContextUsage * 0.4) + (FileCount * 2.0) + (log(TotalLines) * 1.5)
// 参数：
//   - data: ComposerData 会话数据
//
// 返回：熵值（float64）
func (s *EntropyService) CalculateSessionEntropy(data cursor.ComposerData) float64 {
	// 上下文使用率（0-100）
	contextUsage := data.ContextUsagePercent

	// 文件数量
	fileCount := float64(data.FilesChangedCount)

	// 总代码行数（添加 + 删除）
	totalLines := float64(data.GetTotalLinesChanged())
	if totalLines < 1 {
		totalLines = 1 // 避免 log(0)
	}

	// 计算对数（使用自然对数）
	logTotalLines := math.Log(totalLines)

	// 应用权重计算熵值
	// ContextUsage * 0.4 + FileCount * 2.0 + log(TotalLines) * 1.5
	entropy := (contextUsage * 0.4) + (fileCount * 2.0) + (logTotalLines * 1.5)

	return entropy
}

// GetHealthStatus 获取会话健康状态
// 参数：
//   - entropy: 熵值
//
// 返回：健康状态和警告信息
func (s *EntropyService) GetHealthStatus(entropy float64) (status HealthStatus, warning string) {
	switch {
	case entropy < 40:
		return HealthStatusHealthy, ""
	case entropy < 70:
		return HealthStatusSubHealthy, "建议总结当前会话，熵值较高"
	default:
		return HealthStatusDangerous, "会话熵值过高，AI 极易产生幻觉，建议强制开启新会话"
	}
}

// HealthStatus 健康状态
type HealthStatus string

const (
	// HealthStatusHealthy 健康状态（熵 < 40）
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusSubHealthy 亚健康状态（40 <= 熵 < 70）
	HealthStatusSubHealthy HealthStatus = "sub_healthy"
	// HealthStatusDangerous 危险状态（熵 >= 70）
	HealthStatusDangerous HealthStatus = "dangerous"
)

// HealthInfo 健康信息
type HealthInfo struct {
	Entropy float64      `json:"entropy"` // 熵值
	Status  HealthStatus `json:"status"`  // 健康状态
	Warning string       `json:"warning"` // 警告信息（如果有）
}
