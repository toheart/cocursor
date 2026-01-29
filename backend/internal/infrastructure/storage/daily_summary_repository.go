package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/google/uuid"
)

// DailySummaryRepository 每日总结仓储接口
type DailySummaryRepository interface {
	Save(summary *domainCursor.DailySummary) error
	FindByDate(date string) (*domainCursor.DailySummary, error)
	FindDatesByRange(startDate, endDate string) (map[string]bool, error)
	FindByDateRange(startDate, endDate string) ([]*domainCursor.DailySummary, error)
}

// dailySummaryRepository 每日总结仓储实现
type dailySummaryRepository struct {
	db *sql.DB
}

// NewDailySummaryRepository 创建每日总结仓储实例（接受数据库连接作为参数）
func NewDailySummaryRepository(db *sql.DB) DailySummaryRepository {
	return &dailySummaryRepository{
		db: db,
	}
}

// Save 保存每日总结
func (r *dailySummaryRepository) Save(summary *domainCursor.DailySummary) error {
	// 如果 ID 为空，生成新的 UUID
	if summary.ID == "" {
		summary.ID = uuid.New().String()
	}

	// 序列化 work_categories 为 JSON
	workCategoriesJSON, err := json.Marshal(summary.WorkCategories)
	if err != nil {
		return fmt.Errorf("failed to marshal work_categories: %w", err)
	}

	// 序列化 projects 为 JSON
	var projectsJSON string
	if len(summary.Projects) > 0 {
		projectsBytes, err := json.Marshal(summary.Projects)
		if err == nil {
			projectsJSON = string(projectsBytes)
		}
	}

	// 序列化新字段为 JSON
	var codeChangesJSON, timeDistributionJSON, efficiencyMetricsJSON string
	if summary.CodeChanges != nil {
		codeChangesBytes, err := json.Marshal(summary.CodeChanges)
		if err == nil {
			codeChangesJSON = string(codeChangesBytes)
		}
	}
	if summary.TimeDistribution != nil {
		timeDistBytes, err := json.Marshal(summary.TimeDistribution)
		if err == nil {
			timeDistributionJSON = string(timeDistBytes)
		}
	}
	if summary.EfficiencyMetrics != nil {
		effMetricsBytes, err := json.Marshal(summary.EfficiencyMetrics)
		if err == nil {
			efficiencyMetricsJSON = string(effMetricsBytes)
		}
	}

	// 使用 INSERT OR REPLACE 实现 upsert
	query := `
		INSERT OR REPLACE INTO daily_summaries 
		(id, date, summary, language, work_categories, total_sessions, projects, code_changes, time_distribution, efficiency_metrics, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	createdAt := summary.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := summary.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = now
	}

	_, err = r.db.Exec(query,
		summary.ID,
		summary.Date,
		summary.Summary,
		summary.Language,
		string(workCategoriesJSON),
		summary.TotalSessions,
		projectsJSON,
		codeChangesJSON,
		timeDistributionJSON,
		efficiencyMetricsJSON,
		createdAt.Unix(),
		updatedAt.Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to save daily summary: %w", err)
	}

	return nil
}

// FindByDate 按日期查询每日总结
func (r *dailySummaryRepository) FindByDate(date string) (*domainCursor.DailySummary, error) {
	query := `
		SELECT id, date, summary, language, work_categories, total_sessions, projects, code_changes, time_distribution, efficiency_metrics, created_at, updated_at
		FROM daily_summaries
		WHERE date = ?`

	var summary domainCursor.DailySummary
	var workCategoriesJSON string
	var projectsJSON, codeChangesJSON, timeDistributionJSON, efficiencyMetricsJSON sql.NullString
	var createdAt, updatedAt int64

	err := r.db.QueryRow(query, date).Scan(
		&summary.ID,
		&summary.Date,
		&summary.Summary,
		&summary.Language,
		&workCategoriesJSON,
		&summary.TotalSessions,
		&projectsJSON,
		&codeChangesJSON,
		&timeDistributionJSON,
		&efficiencyMetricsJSON,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 未找到，返回 nil 而不是错误
		}
		return nil, fmt.Errorf("failed to query daily summary: %w", err)
	}

	// 反序列化 work_categories
	if err := json.Unmarshal([]byte(workCategoriesJSON), &summary.WorkCategories); err != nil {
		return nil, fmt.Errorf("failed to unmarshal work_categories: %w", err)
	}

	// 反序列化 projects（忽略反序列化错误，字段可为空）
	if projectsJSON.Valid && projectsJSON.String != "" {
		_ = json.Unmarshal([]byte(projectsJSON.String), &summary.Projects)
	}

	// 反序列化新字段（如果存在，忽略反序列化错误，字段可为空）
	if codeChangesJSON.Valid && codeChangesJSON.String != "" {
		_ = json.Unmarshal([]byte(codeChangesJSON.String), &summary.CodeChanges)
	}
	if timeDistributionJSON.Valid && timeDistributionJSON.String != "" {
		_ = json.Unmarshal([]byte(timeDistributionJSON.String), &summary.TimeDistribution)
	}
	if efficiencyMetricsJSON.Valid && efficiencyMetricsJSON.String != "" {
		_ = json.Unmarshal([]byte(efficiencyMetricsJSON.String), &summary.EfficiencyMetrics)
	}

	// 转换时间戳
	summary.CreatedAt = time.Unix(createdAt, 0)
	summary.UpdatedAt = time.Unix(updatedAt, 0)

	return &summary, nil
}

// FindDatesByRange 查询日期范围内有日报的日期
// 返回 map[date]bool，true 表示该日期有日报
func (r *dailySummaryRepository) FindDatesByRange(startDate, endDate string) (map[string]bool, error) {
	query := `
		SELECT date
		FROM daily_summaries
		WHERE date >= ? AND date <= ?`

	rows, err := r.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily summaries by date range: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			continue
		}
		result[date] = true
	}

	return result, nil
}

// FindByDateRange 查询日期范围内的所有日报（完整内容）
// 用于周报聚合等场景
func (r *dailySummaryRepository) FindByDateRange(startDate, endDate string) ([]*domainCursor.DailySummary, error) {
	query := `
		SELECT id, date, summary, language, work_categories, total_sessions, projects, code_changes, time_distribution, efficiency_metrics, created_at, updated_at
		FROM daily_summaries
		WHERE date >= ? AND date <= ?
		ORDER BY date ASC`

	rows, err := r.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily summaries by date range: %w", err)
	}
	defer rows.Close()

	var summaries []*domainCursor.DailySummary
	for rows.Next() {
		var summary domainCursor.DailySummary
		var workCategoriesJSON string
		var projectsJSON, codeChangesJSON, timeDistributionJSON, efficiencyMetricsJSON sql.NullString
		var createdAt, updatedAt int64

		if err := rows.Scan(
			&summary.ID,
			&summary.Date,
			&summary.Summary,
			&summary.Language,
			&workCategoriesJSON,
			&summary.TotalSessions,
			&projectsJSON,
			&codeChangesJSON,
			&timeDistributionJSON,
			&efficiencyMetricsJSON,
			&createdAt,
			&updatedAt,
		); err != nil {
			continue
		}

		// 反序列化字段
		if workCategoriesJSON != "" {
			json.Unmarshal([]byte(workCategoriesJSON), &summary.WorkCategories)
		}
		if projectsJSON.Valid && projectsJSON.String != "" {
			json.Unmarshal([]byte(projectsJSON.String), &summary.Projects)
		}
		if codeChangesJSON.Valid && codeChangesJSON.String != "" {
			json.Unmarshal([]byte(codeChangesJSON.String), &summary.CodeChanges)
		}
		if timeDistributionJSON.Valid && timeDistributionJSON.String != "" {
			json.Unmarshal([]byte(timeDistributionJSON.String), &summary.TimeDistribution)
		}
		if efficiencyMetricsJSON.Valid && efficiencyMetricsJSON.String != "" {
			json.Unmarshal([]byte(efficiencyMetricsJSON.String), &summary.EfficiencyMetrics)
		}

		summary.CreatedAt = time.Unix(createdAt, 0)
		summary.UpdatedAt = time.Unix(updatedAt, 0)

		summaries = append(summaries, &summary)
	}

	return summaries, nil
}
