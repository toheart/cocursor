package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cocursor/backend/internal/domain/todo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// setupTestDB 创建临时测试数据库
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "todo_test_*")
	require.NoError(t, err)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	// 启用 WAL 模式
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	require.NoError(t, err)

	// 清理函数
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestTodoRepository_Save(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTodoRepository(db)

	// 测试创建新待办
	item := &todo.TodoItem{
		Content:   "测试待办",
		Completed: false,
		CreatedAt: time.Now(),
	}

	err := repo.Save(item)
	require.NoError(t, err)
	assert.NotEmpty(t, item.ID, "保存后应自动生成 ID")

	// 测试更新待办
	item.Content = "更新后的待办"
	item.MarkComplete()

	err = repo.Save(item)
	require.NoError(t, err)

	// 验证更新
	found, err := repo.FindByID(item.ID)
	require.NoError(t, err)
	assert.Equal(t, "更新后的待办", found.Content)
	assert.True(t, found.Completed)
	assert.NotNil(t, found.CompletedAt)
}

func TestTodoRepository_FindByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTodoRepository(db)

	// 创建待办
	item := &todo.TodoItem{
		ID:        "test-id-123",
		Content:   "测试待办",
		Completed: false,
		CreatedAt: time.Now(),
	}
	err := repo.Save(item)
	require.NoError(t, err)

	// 测试查找存在的待办
	found, err := repo.FindByID("test-id-123")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, item.Content, found.Content)

	// 测试查找不存在的待办
	notFound, err := repo.FindByID("not-exist")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestTodoRepository_FindAll(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTodoRepository(db)

	// 创建多个待办
	items := []*todo.TodoItem{
		{ID: "1", Content: "待办1", Completed: false, CreatedAt: time.Now().Add(-2 * time.Hour)},
		{ID: "2", Content: "待办2", Completed: true, CreatedAt: time.Now().Add(-1 * time.Hour)},
		{ID: "3", Content: "待办3", Completed: false, CreatedAt: time.Now()},
	}

	for _, item := range items {
		err := repo.Save(item)
		require.NoError(t, err)
	}

	// 测试获取全部
	all, err := repo.FindAll()
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// 验证按创建时间倒序
	assert.Equal(t, "待办3", all[0].Content)
	assert.Equal(t, "待办2", all[1].Content)
	assert.Equal(t, "待办1", all[2].Content)
}

func TestTodoRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTodoRepository(db)

	// 创建待办
	item := &todo.TodoItem{
		ID:        "to-delete",
		Content:   "将被删除",
		Completed: false,
		CreatedAt: time.Now(),
	}
	err := repo.Save(item)
	require.NoError(t, err)

	// 删除
	err = repo.Delete("to-delete")
	require.NoError(t, err)

	// 验证已删除
	found, err := repo.FindByID("to-delete")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestTodoRepository_DeleteCompleted(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTodoRepository(db)

	// 创建混合状态的待办
	items := []*todo.TodoItem{
		{ID: "1", Content: "未完成1", Completed: false, CreatedAt: time.Now()},
		{ID: "2", Content: "已完成1", Completed: true, CreatedAt: time.Now()},
		{ID: "3", Content: "已完成2", Completed: true, CreatedAt: time.Now()},
		{ID: "4", Content: "未完成2", Completed: false, CreatedAt: time.Now()},
	}

	for _, item := range items {
		err := repo.Save(item)
		require.NoError(t, err)
	}

	// 删除已完成的
	count, err := repo.DeleteCompleted()
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// 验证剩余
	all, err := repo.FindAll()
	require.NoError(t, err)
	assert.Len(t, all, 2)

	for _, item := range all {
		assert.False(t, item.Completed, "剩余的应该都是未完成的")
	}
}

func TestTodoRepository_FindCompletedByDateRange(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTodoRepository(db)

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	// 创建不同完成时间的待办
	item1 := &todo.TodoItem{
		ID:        "1",
		Content:   "昨天完成",
		Completed: true,
		CreatedAt: yesterday,
	}
	item1.CompletedAt = &yesterday

	item2 := &todo.TodoItem{
		ID:        "2",
		Content:   "今天完成",
		Completed: true,
		CreatedAt: now,
	}
	completedNow := now
	item2.CompletedAt = &completedNow

	item3 := &todo.TodoItem{
		ID:        "3",
		Content:   "未完成",
		Completed: false,
		CreatedAt: now,
	}

	for _, item := range []*todo.TodoItem{item1, item2, item3} {
		err := repo.Save(item)
		require.NoError(t, err)
	}

	// 查询今天完成的（从今天 0 点到明天 0 点）
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowStart := todayStart.Add(24 * time.Hour)

	completed, err := repo.FindCompletedByDateRange(
		todayStart.UnixMilli(),
		tomorrowStart.UnixMilli(),
	)
	require.NoError(t, err)
	assert.Len(t, completed, 1)
	assert.Equal(t, "今天完成", completed[0].Content)

	// 查询所有时间范围
	allCompleted, err := repo.FindCompletedByDateRange(
		yesterday.Add(-time.Hour).UnixMilli(),
		tomorrow.UnixMilli(),
	)
	require.NoError(t, err)
	assert.Len(t, allCompleted, 2)
}
