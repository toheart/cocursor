package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/cocursor/backend/internal/domain/todo"
	"github.com/google/uuid"
)

// TodoRepository 待办事项仓储接口
type TodoRepository interface {
	Save(item *todo.TodoItem) error
	FindByID(id string) (*todo.TodoItem, error)
	FindAll() ([]*todo.TodoItem, error)
	Delete(id string) error
	DeleteCompleted() (int64, error)
	FindCompletedByDateRange(startTime, endTime int64) ([]*todo.TodoItem, error)
}

// todoRepository 待办事项 SQLite 仓储实现
type todoRepository struct {
	db *sql.DB
}

// NewTodoRepository 创建待办事项仓储实例
func NewTodoRepository(db *sql.DB) TodoRepository {
	// 确保表存在
	if err := initTodoTable(db); err != nil {
		// 初始化失败时记录错误但不阻止创建
		fmt.Printf("failed to init todo table: %v\n", err)
	}
	return &todoRepository{db: db}
}

// initTodoTable 初始化待办事项表
func initTodoTable(db *sql.DB) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS todos (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		completed INTEGER DEFAULT 0,
		created_at INTEGER NOT NULL,
		completed_at INTEGER
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create todos table: %w", err)
	}

	// 创建索引
	createIndexSQL := `
	CREATE INDEX IF NOT EXISTS idx_todos_completed_at ON todos(completed_at);
	CREATE INDEX IF NOT EXISTS idx_todos_created_at ON todos(created_at);
	`

	if _, err := db.Exec(createIndexSQL); err != nil {
		return fmt.Errorf("failed to create todos indexes: %w", err)
	}

	return nil
}

// Save 保存待办事项
func (r *todoRepository) Save(item *todo.TodoItem) error {
	// 如果 ID 为空，生成新的 UUID
	if item.ID == "" {
		item.ID = uuid.New().String()
	}

	// 处理完成时间
	var completedAt sql.NullInt64
	if item.CompletedAt != nil {
		completedAt = sql.NullInt64{
			Int64: item.CompletedAt.UnixMilli(),
			Valid: true,
		}
	}

	// 使用 INSERT OR REPLACE 实现 upsert
	query := `
		INSERT OR REPLACE INTO todos 
		(id, content, completed, created_at, completed_at)
		VALUES (?, ?, ?, ?, ?)`

	completed := 0
	if item.Completed {
		completed = 1
	}

	_, err := r.db.Exec(query,
		item.ID,
		item.Content,
		completed,
		item.CreatedAt.UnixMilli(),
		completedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save todo: %w", err)
	}

	return nil
}

// FindByID 根据 ID 查找待办事项
func (r *todoRepository) FindByID(id string) (*todo.TodoItem, error) {
	query := `
		SELECT id, content, completed, created_at, completed_at
		FROM todos
		WHERE id = ?`

	var item todo.TodoItem
	var completed int
	var createdAt int64
	var completedAt sql.NullInt64

	err := r.db.QueryRow(query, id).Scan(
		&item.ID,
		&item.Content,
		&completed,
		&createdAt,
		&completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query todo: %w", err)
	}

	item.Completed = completed == 1
	item.CreatedAt = time.UnixMilli(createdAt)
	if completedAt.Valid {
		t := time.UnixMilli(completedAt.Int64)
		item.CompletedAt = &t
	}

	return &item, nil
}

// FindAll 获取所有待办事项
func (r *todoRepository) FindAll() ([]*todo.TodoItem, error) {
	query := `
		SELECT id, content, completed, created_at, completed_at
		FROM todos
		ORDER BY created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query todos: %w", err)
	}
	defer rows.Close()

	var items []*todo.TodoItem
	for rows.Next() {
		var item todo.TodoItem
		var completed int
		var createdAt int64
		var completedAt sql.NullInt64

		if err := rows.Scan(
			&item.ID,
			&item.Content,
			&completed,
			&createdAt,
			&completedAt,
		); err != nil {
			continue
		}

		item.Completed = completed == 1
		item.CreatedAt = time.UnixMilli(createdAt)
		if completedAt.Valid {
			t := time.UnixMilli(completedAt.Int64)
			item.CompletedAt = &t
		}

		items = append(items, &item)
	}

	return items, nil
}

// Delete 删除待办事项
func (r *todoRepository) Delete(id string) error {
	query := `DELETE FROM todos WHERE id = ?`
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}
	return nil
}

// DeleteCompleted 删除所有已完成的待办事项
func (r *todoRepository) DeleteCompleted() (int64, error) {
	query := `DELETE FROM todos WHERE completed = 1`
	result, err := r.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete completed todos: %w", err)
	}
	return result.RowsAffected()
}

// FindCompletedByDateRange 查询日期范围内完成的待办
func (r *todoRepository) FindCompletedByDateRange(startTime, endTime int64) ([]*todo.TodoItem, error) {
	query := `
		SELECT id, content, completed, created_at, completed_at
		FROM todos
		WHERE completed = 1 
		  AND completed_at >= ? 
		  AND completed_at < ?
		ORDER BY completed_at ASC`

	rows, err := r.db.Query(query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query completed todos by date range: %w", err)
	}
	defer rows.Close()

	var items []*todo.TodoItem
	for rows.Next() {
		var item todo.TodoItem
		var completed int
		var createdAt int64
		var completedAt sql.NullInt64

		if err := rows.Scan(
			&item.ID,
			&item.Content,
			&completed,
			&createdAt,
			&completedAt,
		); err != nil {
			continue
		}

		item.Completed = completed == 1
		item.CreatedAt = time.UnixMilli(createdAt)
		if completedAt.Valid {
			t := time.UnixMilli(completedAt.Int64)
			item.CompletedAt = &t
		}

		items = append(items, &item)
	}

	return items, nil
}

// 编译时检查接口实现
var _ TodoRepository = (*todoRepository)(nil)
