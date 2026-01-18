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
}

// dailySummaryRepository 每日总结仓储实现
type dailySummaryRepository struct {
	db *sql.DB
}

// NewDailySummaryRepository 创建每日总结仓储实例
func NewDailySummaryRepository() (DailySummaryRepository, error) {
	// 确保数据库已初始化
	if err := InitDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	db, err := OpenDB()
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &dailySummaryRepository{
		db: db,
	}, nil
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

	// Projects 信息已经包含在 summary 文本中（Markdown 格式）
	// 如果需要单独存储和查询 projects，可以创建关联表

	// 使用 INSERT OR REPLACE 实现 upsert
	query := `
		INSERT OR REPLACE INTO daily_summaries 
		(id, date, summary, language, work_categories, total_sessions, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

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
		SELECT id, date, summary, language, work_categories, total_sessions, created_at, updated_at
		FROM daily_summaries
		WHERE date = ?`

	var summary domainCursor.DailySummary
	var workCategoriesJSON string
	var createdAt, updatedAt int64

	err := r.db.QueryRow(query, date).Scan(
		&summary.ID,
		&summary.Date,
		&summary.Summary,
		&summary.Language,
		&workCategoriesJSON,
		&summary.TotalSessions,
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

	// 转换时间戳
	summary.CreatedAt = time.Unix(createdAt, 0)
	summary.UpdatedAt = time.Unix(updatedAt, 0)

	// Projects 信息存储在 summary 文本中（Markdown 格式）
	// 如果需要单独存储和查询，可以创建关联表
	// 这里简化处理，projects 信息从 summary 文本中解析

	return &summary, nil
}
