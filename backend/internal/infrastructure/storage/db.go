package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

// OpenDB 打开数据库连接（保留用于向后兼容，新代码应使用 ProvideDB）
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

	// 启用 WAL 模式（Write-Ahead Logging）
	// WAL 模式允许多个读操作和一个写操作同时进行，提供更好的并发性能
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// 配置连接池
	configureDB(db)

	return db, nil
}

// ProvideDB 提供数据库连接（用于依赖注入）
// 确保数据库已初始化，配置连接池，并返回共享的数据库连接
func ProvideDB() (*sql.DB, error) {
	// 确保数据库已初始化
	if err := InitDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// 打开数据库连接
	db, err := OpenDB()
	if err != nil {
		return nil, err
	}

	return db, nil
}

// configureDB 配置数据库连接池参数
func configureDB(db *sql.DB) {
	// 启用 WAL 模式后，SQLite 可以支持多个读连接和一个写连接
	// 设置最大打开连接数为 5（1 个写连接 + 4 个读连接）
	db.SetMaxOpenConns(5)
	// 设置最大空闲连接数
	db.SetMaxIdleConns(2)
	// 设置连接最大生存时间（5 分钟）
	db.SetConnMaxLifetime(5 * time.Minute)
	// 设置连接最大空闲时间（10 分钟）
	db.SetConnMaxIdleTime(10 * time.Minute)
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
		code_changes TEXT,
		time_distribution TEXT,
		efficiency_metrics TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create daily_summaries table: %w", err)
	}

	// 迁移：为已存在的表添加新字段（向后兼容）
	// 注意：SQLite 不支持 IF NOT EXISTS 在 ALTER TABLE ADD COLUMN 中
	// 如果字段已存在，会返回错误，我们忽略这些错误
	migrationSQLs := []string{
		"ALTER TABLE daily_summaries ADD COLUMN code_changes TEXT",
		"ALTER TABLE daily_summaries ADD COLUMN time_distribution TEXT",
		"ALTER TABLE daily_summaries ADD COLUMN efficiency_metrics TEXT",
		"ALTER TABLE daily_summaries ADD COLUMN projects TEXT",
		"ALTER TABLE workspace_sessions ADD COLUMN token_count INTEGER DEFAULT 0",
	}

	// 执行迁移（忽略错误，因为字段可能已存在）
	for _, sql := range migrationSQLs {
		_, _ = db.Exec(sql) // 忽略错误，字段可能已存在
	}

	// 创建索引
	createIndexSQL := `
	CREATE INDEX IF NOT EXISTS idx_daily_summaries_date ON daily_summaries(date);`

	if _, err := db.Exec(createIndexSQL); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// 创建 workspace_sessions 表
	createWorkspaceSessionsTableSQL := `
	CREATE TABLE IF NOT EXISTS workspace_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id TEXT NOT NULL,
		composer_id TEXT NOT NULL,
		name TEXT,
		type TEXT,
		created_at INTEGER NOT NULL,
		last_updated_at INTEGER NOT NULL,
		unified_mode TEXT,
		subtitle TEXT,
		total_lines_added INTEGER DEFAULT 0,
		total_lines_removed INTEGER DEFAULT 0,
		files_changed_count INTEGER DEFAULT 0,
		context_usage_percent REAL DEFAULT 0,
		is_archived INTEGER DEFAULT 0,
		created_on_branch TEXT,
		token_count INTEGER DEFAULT 0,
		cached_at INTEGER NOT NULL,
		UNIQUE(workspace_id, composer_id)
	);`

	if _, err := db.Exec(createWorkspaceSessionsTableSQL); err != nil {
		return fmt.Errorf("failed to create workspace_sessions table: %w", err)
	}

	// 创建 workspace_sessions 索引
	createWorkspaceSessionsIndexSQL := `
	CREATE INDEX IF NOT EXISTS idx_workspace_sessions_workspace ON workspace_sessions(workspace_id);
	CREATE INDEX IF NOT EXISTS idx_workspace_sessions_dates ON workspace_sessions(workspace_id, created_at, last_updated_at);
	CREATE INDEX IF NOT EXISTS idx_workspace_sessions_name ON workspace_sessions(name);
	CREATE INDEX IF NOT EXISTS idx_workspace_sessions_archived ON workspace_sessions(workspace_id, is_archived);`

	if _, err := db.Exec(createWorkspaceSessionsIndexSQL); err != nil {
		return fmt.Errorf("failed to create workspace_sessions indexes: %w", err)
	}

	// 创建 workspace_file_metadata 表
	createWorkspaceFileMetadataTableSQL := `
	CREATE TABLE IF NOT EXISTS workspace_file_metadata (
		workspace_id TEXT PRIMARY KEY,
		db_path TEXT NOT NULL,
		file_mtime INTEGER NOT NULL,
		file_size INTEGER NOT NULL,
		last_scan_time INTEGER NOT NULL,
		last_sync_time INTEGER,
		sessions_count INTEGER DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);`

	if _, err := db.Exec(createWorkspaceFileMetadataTableSQL); err != nil {
		return fmt.Errorf("failed to create workspace_file_metadata table: %w", err)
	}

	// 创建 rag_knowledge_chunks 表
	createRAGKnowledgeChunksTableSQL := `
	CREATE TABLE IF NOT EXISTS rag_knowledge_chunks (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		chunk_index INTEGER NOT NULL,
		project_id TEXT NOT NULL,
		project_name TEXT,
		workspace_id TEXT,
		
		user_query TEXT NOT NULL,
		ai_response_core TEXT NOT NULL,
		vector_text TEXT NOT NULL,
		
		tools_used TEXT,
		files_modified TEXT,
		code_languages TEXT,
		has_code INTEGER DEFAULT 0,
		
		summary TEXT,
		main_topic TEXT,
		tags TEXT,
		enrichment_status TEXT DEFAULT 'pending',
		enrichment_error TEXT,
		
		timestamp INTEGER NOT NULL,
		content_hash TEXT NOT NULL,
		file_path TEXT NOT NULL,
		indexed_at INTEGER NOT NULL,
		
		UNIQUE(session_id, chunk_index)
	);`

	if _, err := db.Exec(createRAGKnowledgeChunksTableSQL); err != nil {
		return fmt.Errorf("failed to create rag_knowledge_chunks table: %w", err)
	}

	// 创建 rag_index_status 表
	createRAGIndexStatusTableSQL := `
	CREATE TABLE IF NOT EXISTS rag_index_status (
		file_path TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		project_id TEXT NOT NULL,
		content_hash TEXT NOT NULL,
		chunk_count INTEGER DEFAULT 0,
		file_mtime INTEGER NOT NULL,
		last_indexed_at INTEGER NOT NULL,
		status TEXT DEFAULT 'indexed'
	);`

	if _, err := db.Exec(createRAGIndexStatusTableSQL); err != nil {
		return fmt.Errorf("failed to create rag_index_status table: %w", err)
	}

	// 创建 rag_enrichment_queue 表
	createRAGEnrichmentQueueTableSQL := `
	CREATE TABLE IF NOT EXISTS rag_enrichment_queue (
		chunk_id TEXT PRIMARY KEY,
		priority INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending',
		retry_count INTEGER DEFAULT 0,
		max_retries INTEGER DEFAULT 3,
		created_at INTEGER NOT NULL,
		next_retry_at INTEGER,
		last_error TEXT
	);`

	if _, err := db.Exec(createRAGEnrichmentQueueTableSQL); err != nil {
		return fmt.Errorf("failed to create rag_enrichment_queue table: %w", err)
	}

	// 创建新 RAG 表的索引
	createNewRAGIndexesSQL := `
	CREATE INDEX IF NOT EXISTS idx_rag_chunks_session ON rag_knowledge_chunks(session_id);
	CREATE INDEX IF NOT EXISTS idx_rag_chunks_project ON rag_knowledge_chunks(project_id);
	CREATE INDEX IF NOT EXISTS idx_rag_chunks_enrichment ON rag_knowledge_chunks(enrichment_status);
	CREATE INDEX IF NOT EXISTS idx_rag_chunks_timestamp ON rag_knowledge_chunks(timestamp);
	CREATE INDEX IF NOT EXISTS idx_rag_index_status_session ON rag_index_status(session_id);
	CREATE INDEX IF NOT EXISTS idx_rag_enrichment_status ON rag_enrichment_queue(status);
	CREATE INDEX IF NOT EXISTS idx_rag_enrichment_retry ON rag_enrichment_queue(status, next_retry_at);
	`

	if _, err := db.Exec(createNewRAGIndexesSQL); err != nil {
		return fmt.Errorf("failed to create new RAG indexes: %w", err)
	}

	return nil
}
