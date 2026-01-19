package cursor

import (
	"database/sql"
	"fmt"
	"os"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	_ "modernc.org/sqlite"
)

// globalDBReader Global 数据库读取器实现
type globalDBReader struct {
	pathResolver *PathResolver
	dbPath       string
}

// NewGlobalDBReader 创建 Global 数据库读取器实例
// pathResolver: 路径解析器，用于获取 Global 数据库路径
func NewGlobalDBReader(pathResolver *PathResolver) (domainCursor.GlobalDBReader, error) {
	// 获取 Global 数据库路径
	dbPath, err := pathResolver.GetGlobalStoragePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get global storage path: %w", err)
	}

	// 检查文件是否存在
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("global storage database not found: %w", err)
	}

	return &globalDBReader{
		pathResolver: pathResolver,
		dbPath:       dbPath,
	}, nil
}

// ReadValue 从 Global 数据库读取指定键的值
func (r *globalDBReader) ReadValue(key string) ([]byte, error) {
	// 构建只读连接字符串
	// modernc.org/sqlite 支持 file: URI 格式，使用 mode=ro 确保只读模式
	// 格式: file:///absolute/path?mode=ro 或 file:relative/path?mode=ro
	// 对于绝对路径，需要使用 file:/// 前缀（三个斜杠）
	connStr := fmt.Sprintf("file:%s?mode=ro", r.dbPath)

	// 打开只读数据库连接
	db, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open global database: %w", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping global database: %w", err)
	}

	// 查询数据
	var value []byte
	query := "SELECT value FROM ItemTable WHERE key = ?"
	err = db.QueryRow(query, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to query global database: %w", err)
	}

	return value, nil
}
