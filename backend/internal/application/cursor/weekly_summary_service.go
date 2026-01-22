package cursor

import (
	"fmt"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/cocursor/backend/internal/infrastructure/storage"
)

// WeeklySummaryService 周报服务（统一 MCP 和 HTTP 的业务逻辑）
type WeeklySummaryService struct {
	weeklySummaryRepo storage.WeeklySummaryRepository
	dailySummaryRepo  storage.DailySummaryRepository
}

// NewWeeklySummaryService 创建周报服务实例
func NewWeeklySummaryService(
	weeklySummaryRepo storage.WeeklySummaryRepository,
	dailySummaryRepo storage.DailySummaryRepository,
) *WeeklySummaryService {
	return &WeeklySummaryService{
		weeklySummaryRepo: weeklySummaryRepo,
		dailySummaryRepo:  dailySummaryRepo,
	}
}

// SaveWeeklySummaryInput 保存周报输入
type SaveWeeklySummaryInput struct {
	WeekStart          string                              `json:"week_start"`
	WeekEnd            string                              `json:"week_end"`
	Summary            string                              `json:"summary"`
	Language           string                              `json:"language"`
	Projects           []*domainCursor.WeeklyProjectSummary `json:"projects"`
	Categories         *domainCursor.WorkCategories        `json:"categories"`
	TotalSessions      int                                 `json:"total_sessions"`
	WorkingDays        int                                 `json:"working_days"`
	CodeChanges        *domainCursor.CodeChangeSummary     `json:"code_changes,omitempty"`
	KeyAccomplishments []string                            `json:"key_accomplishments,omitempty"`
}

// SaveWeeklySummaryResult 保存周报结果
type SaveWeeklySummaryResult struct {
	Success   bool   `json:"success"`
	SummaryID string `json:"summary_id,omitempty"`
	Message   string `json:"message"`
}

// SaveWeeklySummary 保存周报（支持幂等更新）
func (s *WeeklySummaryService) SaveWeeklySummary(input *SaveWeeklySummaryInput) (*SaveWeeklySummaryResult, error) {
	// 验证必需字段
	if input.WeekStart == "" {
		return nil, fmt.Errorf("week_start is required")
	}
	if input.WeekEnd == "" {
		return nil, fmt.Errorf("week_end is required")
	}
	if input.Summary == "" {
		return nil, fmt.Errorf("summary is required")
	}
	if input.Language == "" {
		input.Language = "zh"
	}

	// 处理 categories 参数
	categories := input.Categories
	if categories == nil {
		categories = &domainCursor.WorkCategories{}
	}

	// 计算源数据哈希（用于幂等检查）
	dailySummaries, err := s.dailySummaryRepo.FindByDateRange(input.WeekStart, input.WeekEnd)
	if err != nil {
		// 如果查询失败，继续保存但不设置哈希
		dailySummaries = nil
	}
	dataHash := ComputeDataHash(dailySummaries)

	// 构建 WeeklySummary 对象
	summary := &domainCursor.WeeklySummary{
		WeekStart:          input.WeekStart,
		WeekEnd:            input.WeekEnd,
		Summary:            input.Summary,
		Language:           input.Language,
		Projects:           input.Projects,
		WorkCategories:     categories,
		TotalSessions:      input.TotalSessions,
		WorkingDays:        input.WorkingDays,
		CodeChanges:        input.CodeChanges,
		KeyAccomplishments: input.KeyAccomplishments,
		DataHash:           dataHash,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// 保存到数据库（UPSERT，基于 week_start 唯一约束）
	if err := s.weeklySummaryRepo.Save(summary); err != nil {
		return &SaveWeeklySummaryResult{
			Success: false,
			Message: fmt.Sprintf("save failed: %v", err),
		}, nil
	}

	return &SaveWeeklySummaryResult{
		Success:   true,
		SummaryID: summary.ID,
		Message:   "save successful",
	}, nil
}

// GetWeeklySummaryResult 查询周报结果
type GetWeeklySummaryResult struct {
	Summary     *domainCursor.WeeklySummary `json:"summary,omitempty"`
	Found       bool                        `json:"found"`
	NeedsUpdate bool                        `json:"needs_update"`
}

// GetWeeklySummary 查询周报（带幂等检查）
func (s *WeeklySummaryService) GetWeeklySummary(weekStart string) (*GetWeeklySummaryResult, error) {
	if weekStart == "" {
		return nil, fmt.Errorf("week_start is required")
	}

	// 查询周报
	summary, err := s.weeklySummaryRepo.FindByWeekStart(weekStart)
	if err != nil {
		return nil, err
	}

	if summary == nil {
		return &GetWeeklySummaryResult{
			Found:       false,
			NeedsUpdate: true, // 不存在则需要生成
		}, nil
	}

	// 检查源数据是否有变化（幂等检查）
	needsUpdate := false
	if summary.DataHash != "" {
		// 获取当前日报数据并计算哈希
		dailySummaries, err := s.dailySummaryRepo.FindByDateRange(summary.WeekStart, summary.WeekEnd)
		if err == nil {
			currentHash := ComputeDataHash(dailySummaries)
			needsUpdate = currentHash != summary.DataHash
		}
	}

	return &GetWeeklySummaryResult{
		Summary:     summary,
		Found:       true,
		NeedsUpdate: needsUpdate,
	}, nil
}

// GetWeeklySummariesRange 批量查询周报
func (s *WeeklySummaryService) GetWeeklySummariesRange(startDate, endDate string) ([]*domainCursor.WeeklySummary, error) {
	if startDate == "" || endDate == "" {
		return nil, fmt.Errorf("start_date and end_date are required")
	}

	return s.weeklySummaryRepo.FindByWeekRange(startDate, endDate)
}
