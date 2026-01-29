package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/google/uuid"
)

// WeeklySummaryRepository 每周总结仓储接口
type WeeklySummaryRepository interface {
	Save(summary *domainCursor.WeeklySummary) error
	FindByWeekStart(weekStart string) (*domainCursor.WeeklySummary, error)
	FindByWeekRange(startDate, endDate string) ([]*domainCursor.WeeklySummary, error)
}

// weeklySummaryRepository 每周总结仓储实现
type weeklySummaryRepository struct {
	db *sql.DB
}

// NewWeeklySummaryRepository 创建每周总结仓储实例
func NewWeeklySummaryRepository(db *sql.DB) WeeklySummaryRepository {
	return &weeklySummaryRepository{
		db: db,
	}
}

// Save 保存每周总结（支持幂等更新）
func (r *weeklySummaryRepository) Save(summary *domainCursor.WeeklySummary) error {
	// 如果 ID 为空，生成新的 UUID
	if summary.ID == "" {
		summary.ID = uuid.New().String()
	}

	// 序列化 work_categories 为 JSON
	var categoriesJSON string
	if summary.WorkCategories != nil {
		categoriesBytes, err := json.Marshal(summary.WorkCategories)
		if err != nil {
			return fmt.Errorf("failed to marshal work_categories: %w", err)
		}
		categoriesJSON = string(categoriesBytes)
	}

	// 序列化 projects 为 JSON
	var projectsJSON string
	if len(summary.Projects) > 0 {
		projectsBytes, err := json.Marshal(summary.Projects)
		if err == nil {
			projectsJSON = string(projectsBytes)
		}
	}

	// 序列化 code_changes 为 JSON
	var codeChangesJSON string
	if summary.CodeChanges != nil {
		codeChangesBytes, err := json.Marshal(summary.CodeChanges)
		if err == nil {
			codeChangesJSON = string(codeChangesBytes)
		}
	}

	// 序列化 key_accomplishments 为 JSON
	var keyAccomplishmentsJSON string
	if len(summary.KeyAccomplishments) > 0 {
		keyAccomplishmentsBytes, err := json.Marshal(summary.KeyAccomplishments)
		if err == nil {
			keyAccomplishmentsJSON = string(keyAccomplishmentsBytes)
		}
	}

	// 使用 INSERT OR REPLACE 实现 upsert（基于 UNIQUE(week_start)）
	query := `
		INSERT OR REPLACE INTO weekly_summaries 
		(id, week_start, week_end, summary, language, projects, categories, total_sessions, working_days, code_changes, key_accomplishments, data_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	createdAt := summary.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := summary.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = now
	}

	_, err := r.db.Exec(query,
		summary.ID,
		summary.WeekStart,
		summary.WeekEnd,
		summary.Summary,
		summary.Language,
		projectsJSON,
		categoriesJSON,
		summary.TotalSessions,
		summary.WorkingDays,
		codeChangesJSON,
		keyAccomplishmentsJSON,
		summary.DataHash,
		createdAt.Unix(),
		updatedAt.Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to save weekly summary: %w", err)
	}

	return nil
}

// FindByWeekStart 按周起始日期查询每周总结
func (r *weeklySummaryRepository) FindByWeekStart(weekStart string) (*domainCursor.WeeklySummary, error) {
	query := `
		SELECT id, week_start, week_end, summary, language, projects, categories, total_sessions, working_days, code_changes, key_accomplishments, data_hash, created_at, updated_at
		FROM weekly_summaries
		WHERE week_start = ?`

	var summary domainCursor.WeeklySummary
	var categoriesJSON string
	var projectsJSON, codeChangesJSON, keyAccomplishmentsJSON, dataHash sql.NullString
	var createdAt, updatedAt int64

	err := r.db.QueryRow(query, weekStart).Scan(
		&summary.ID,
		&summary.WeekStart,
		&summary.WeekEnd,
		&summary.Summary,
		&summary.Language,
		&projectsJSON,
		&categoriesJSON,
		&summary.TotalSessions,
		&summary.WorkingDays,
		&codeChangesJSON,
		&keyAccomplishmentsJSON,
		&dataHash,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 未找到，返回 nil 而不是错误
		}
		return nil, fmt.Errorf("failed to query weekly summary: %w", err)
	}

	// 反序列化 work_categories（忽略反序列化错误）
	if categoriesJSON != "" {
		_ = json.Unmarshal([]byte(categoriesJSON), &summary.WorkCategories)
	}

	// 反序列化 projects（忽略反序列化错误）
	if projectsJSON.Valid && projectsJSON.String != "" {
		_ = json.Unmarshal([]byte(projectsJSON.String), &summary.Projects)
	}

	// 反序列化 code_changes（忽略反序列化错误）
	if codeChangesJSON.Valid && codeChangesJSON.String != "" {
		_ = json.Unmarshal([]byte(codeChangesJSON.String), &summary.CodeChanges)
	}

	// 反序列化 key_accomplishments（忽略反序列化错误）
	if keyAccomplishmentsJSON.Valid && keyAccomplishmentsJSON.String != "" {
		_ = json.Unmarshal([]byte(keyAccomplishmentsJSON.String), &summary.KeyAccomplishments)
	}

	// 设置 data_hash
	if dataHash.Valid {
		summary.DataHash = dataHash.String
	}

	// 转换时间戳
	summary.CreatedAt = time.Unix(createdAt, 0)
	summary.UpdatedAt = time.Unix(updatedAt, 0)

	return &summary, nil
}

// FindByWeekRange 按周范围查询多个周报
func (r *weeklySummaryRepository) FindByWeekRange(startDate, endDate string) ([]*domainCursor.WeeklySummary, error) {
	query := `
		SELECT id, week_start, week_end, summary, language, projects, categories, total_sessions, working_days, code_changes, key_accomplishments, data_hash, created_at, updated_at
		FROM weekly_summaries
		WHERE week_start >= ? AND week_start <= ?
		ORDER BY week_start DESC`

	rows, err := r.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query weekly summaries: %w", err)
	}
	defer rows.Close()

	var summaries []*domainCursor.WeeklySummary
	for rows.Next() {
		var summary domainCursor.WeeklySummary
		var categoriesJSON string
		var projectsJSON, codeChangesJSON, keyAccomplishmentsJSON, dataHash sql.NullString
		var createdAt, updatedAt int64

		if err := rows.Scan(
			&summary.ID,
			&summary.WeekStart,
			&summary.WeekEnd,
			&summary.Summary,
			&summary.Language,
			&projectsJSON,
			&categoriesJSON,
			&summary.TotalSessions,
			&summary.WorkingDays,
			&codeChangesJSON,
			&keyAccomplishmentsJSON,
			&dataHash,
			&createdAt,
			&updatedAt,
		); err != nil {
			continue
		}

		// 反序列化字段
		if categoriesJSON != "" {
			json.Unmarshal([]byte(categoriesJSON), &summary.WorkCategories)
		}
		if projectsJSON.Valid && projectsJSON.String != "" {
			json.Unmarshal([]byte(projectsJSON.String), &summary.Projects)
		}
		if codeChangesJSON.Valid && codeChangesJSON.String != "" {
			json.Unmarshal([]byte(codeChangesJSON.String), &summary.CodeChanges)
		}
		if keyAccomplishmentsJSON.Valid && keyAccomplishmentsJSON.String != "" {
			json.Unmarshal([]byte(keyAccomplishmentsJSON.String), &summary.KeyAccomplishments)
		}
		if dataHash.Valid {
			summary.DataHash = dataHash.String
		}

		summary.CreatedAt = time.Unix(createdAt, 0)
		summary.UpdatedAt = time.Unix(updatedAt, 0)

		summaries = append(summaries, &summary)
	}

	return summaries, nil
}
