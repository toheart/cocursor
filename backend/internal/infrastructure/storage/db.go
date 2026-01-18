package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// GetDBPath 获取 cocursor 数据库路径
// Windows: %USERPROFILE%\.cocursor\cocursor.db
// macOS/Linux: ~/.cocursor/cocursor.db
func GetDBPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	dbPath := filepath.Join(homeDir, ".cocursor", "cocursor.db")
	return dbPath, nil
}

// OpenDB 打开数据库连接
func OpenDB() (*sql.DB, error) {
	dbPath, err := GetDBPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get database path: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// 打开数据库连接
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// InitDatabase 初始化数据库和表结构
func InitDatabase() error {
	db, err := OpenDB()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// 创建 daily_summaries 表
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS daily_summaries (
		id TEXT PRIMARY KEY,
		date TEXT NOT NULL UNIQUE,
		summary TEXT NOT NULL,
		language TEXT NOT NULL,
		work_categories TEXT NOT NULL,
		total_sessions INTEGER NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create daily_summaries table: %w", err)
	}

	// 创建索引
	createIndexSQL := `
	CREATE INDEX IF NOT EXISTS idx_daily_summaries_date ON daily_summaries(date);`

	if _, err := db.Exec(createIndexSQL); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// 创建 openspec_workflows 表
	createOpenSpecTableSQL := `
	CREATE TABLE IF NOT EXISTS openspec_workflows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id TEXT NOT NULL,
		project_path TEXT NOT NULL,
		change_id TEXT NOT NULL,
		stage TEXT NOT NULL,
		status TEXT NOT NULL,
		started_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		metadata TEXT,
		summary TEXT,
		UNIQUE(workspace_id, change_id)
	);`

	if _, err := db.Exec(createOpenSpecTableSQL); err != nil {
		return fmt.Errorf("failed to create openspec_workflows table: %w", err)
	}

	// 创建索引
	createOpenSpecIndexSQL := `
	CREATE INDEX IF NOT EXISTS idx_openspec_workspace_change ON openspec_workflows(workspace_id, change_id);
	CREATE INDEX IF NOT EXISTS idx_openspec_status ON openspec_workflows(status);`

	if _, err := db.Exec(createOpenSpecIndexSQL); err != nil {
		return fmt.Errorf("failed to create openspec_workflows indexes: %w", err)
	}

	return nil
}
