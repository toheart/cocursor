package cursor

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DBReader 数据库读取器，使用只读副本避免文件锁
type DBReader struct{}

// NewDBReader 创建数据库读取器实例
func NewDBReader() *DBReader {
	return &DBReader{}
}

// ReadValueFromTable 从数据库表中读取值
// dbPath: 数据库文件路径
// key: 要查询的键
// 返回: 对应的值（BLOB 数据）
func (r *DBReader) ReadValueFromTable(dbPath string, key string) ([]byte, error) {
	// 检查源文件是否存在
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("database file not found: %w", err)
	}

	// 创建临时文件路径
	tmpDir := os.TempDir()
	tmpFileName := fmt.Sprintf("cocursor_tmp_%s.db", filepath.Base(dbPath))
	tmpPath := filepath.Join(tmpDir, tmpFileName)

	// 复制数据库文件到临时目录
	if err := r.copyFile(dbPath, tmpPath); err != nil {
		return nil, fmt.Errorf("failed to copy database file: %w", err)
	}

	// 确保清理临时文件
	defer func() {
		os.Remove(tmpPath)
	}()

	// 打开临时数据库文件
	db, err := sql.Open("sqlite", tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// 查询数据
	var value []byte
	query := "SELECT value FROM ItemTable WHERE key = ?"
	err = db.QueryRow(query, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	return value, nil
}

// ReadValueFromWorkspaceDB 从工作区数据库读取值
// workspaceDBPath: 工作区数据库文件路径
// key: 要查询的键
// 返回: 对应的值（BLOB 数据）
func (r *DBReader) ReadValueFromWorkspaceDB(workspaceDBPath string, key string) ([]byte, error) {
	// 工作区数据库也使用相同的只读副本逻辑
	return r.ReadValueFromTable(workspaceDBPath, key)
}

// ReadWorkspaceData 通过 WorkspaceID 和 Key 读取工作区数据
// workspaceID: 工作区 ID（哈希值）
// key: 要查询的键，如 "composer.composerData"
// 返回: 对应的值（BLOB 数据）
func (r *DBReader) ReadWorkspaceData(workspaceID string, key string) ([]byte, error) {
	// 使用 PathResolver 获取数据库路径
	pathResolver := NewPathResolver()
	dbPath, err := pathResolver.GetWorkspaceDBPath(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace DB path: %w", err)
	}

	// 调用现有的 ReadValueFromWorkspaceDB 方法
	return r.ReadValueFromWorkspaceDB(dbPath, key)
}

// ReadKeysWithPrefixFromWorkspaceDB 从工作区数据库读取具有指定前缀的所有键
// workspaceDBPath: 工作区数据库文件路径
// prefix: 键的前缀
// 返回: 键列表
func (r *DBReader) ReadKeysWithPrefixFromWorkspaceDB(workspaceDBPath string, prefix string) ([]string, error) {
	return r.QueryAllKeys(workspaceDBPath, prefix+"%")
}

// QueryAllKeys 查询所有键（支持模糊匹配）
// dbPath: 数据库文件路径
// pattern: 键的模式（支持 % 通配符），如 "aiService.%" 或 "%token%"
// 返回: 键列表
func (r *DBReader) QueryAllKeys(dbPath string, pattern string) ([]string, error) {
	// 检查源文件是否存在
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("database file not found: %w", err)
	}

	// 创建临时文件路径
	tmpDir := os.TempDir()
	tmpFileName := fmt.Sprintf("cocursor_tmp_%s.db", filepath.Base(dbPath))
	tmpPath := filepath.Join(tmpDir, tmpFileName)

	// 复制数据库文件到临时目录
	if err := r.copyFile(dbPath, tmpPath); err != nil {
		return nil, fmt.Errorf("failed to copy database file: %w", err)
	}

	// 确保清理临时文件
	defer func() {
		os.Remove(tmpPath)
	}()

	// 打开临时数据库文件
	db, err := sql.Open("sqlite", tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// 查询所有匹配的键
	var keys []string
	query := "SELECT key FROM ItemTable WHERE key LIKE ? ORDER BY key"
	rows, err := db.Query(query, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			continue
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// copyFile 复制文件
func (r *DBReader) copyFile(src, dst string) error {
	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// 复制数据
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// 确保数据写入磁盘
	return dstFile.Sync()
}
