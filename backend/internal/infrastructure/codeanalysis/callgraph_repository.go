package codeanalysis

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/cocursor/backend/internal/domain/codeanalysis"
	"github.com/cocursor/backend/internal/infrastructure/log"
	_ "modernc.org/sqlite"
)

// CallGraphRepository 调用图存储仓库实现
type CallGraphRepository struct {
	logger *slog.Logger
}

// NewCallGraphRepository 创建调用图存储仓库
func NewCallGraphRepository() *CallGraphRepository {
	return &CallGraphRepository{
		logger: log.NewModuleLogger("codeanalysis", "callgraph_repository"),
	}
}

// Init 初始化数据库表结构
func (r *CallGraphRepository) Init(_ context.Context, dbPath string) error {
	// 确保父目录存在
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// 创建表
	schema := `
		-- 函数节点表
		CREATE TABLE IF NOT EXISTS func_nodes (
			id INTEGER PRIMARY KEY,
			full_name TEXT UNIQUE NOT NULL,
			canonical_name TEXT NOT NULL DEFAULT '',
			package TEXT NOT NULL,
			func_name TEXT NOT NULL,
			file_path TEXT,
			line_start INTEGER DEFAULT 0,
			line_end INTEGER DEFAULT 0,
			is_exported BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		-- 调用边表
		CREATE TABLE IF NOT EXISTS func_edges (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			caller_id INTEGER NOT NULL,
			callee_id INTEGER NOT NULL,
			call_site_file TEXT,
			call_site_line INTEGER DEFAULT 0,
			FOREIGN KEY (caller_id) REFERENCES func_nodes(id),
			FOREIGN KEY (callee_id) REFERENCES func_nodes(id)
		);

		-- 元数据表
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		);

		-- 创建索引
		CREATE INDEX IF NOT EXISTS idx_func_nodes_package ON func_nodes(package);
		CREATE INDEX IF NOT EXISTS idx_func_nodes_file ON func_nodes(file_path);
		CREATE INDEX IF NOT EXISTS idx_func_nodes_canonical ON func_nodes(canonical_name);
		CREATE INDEX IF NOT EXISTS idx_func_edges_caller ON func_edges(caller_id);
		CREATE INDEX IF NOT EXISTS idx_func_edges_callee ON func_edges(callee_id);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_func_edges_unique ON func_edges(caller_id, callee_id, call_site_line);
	`

	_, err = db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// SaveFuncNode 保存函数节点
func (r *CallGraphRepository) SaveFuncNode(_ context.Context, dbPath string, node *codeanalysis.FuncNode) (int64, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	result, err := db.Exec(`
		INSERT INTO func_nodes (id, full_name, canonical_name, package, func_name, file_path, line_start, line_end, is_exported)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(full_name) DO UPDATE SET
			canonical_name = excluded.canonical_name,
			package = excluded.package,
			func_name = excluded.func_name,
			file_path = excluded.file_path,
			line_start = excluded.line_start,
			line_end = excluded.line_end,
			is_exported = excluded.is_exported
	`, node.ID, node.FullName, node.CanonicalName, node.Package, node.FuncName, node.FilePath, node.LineStart, node.LineEnd, node.IsExported)
	if err != nil {
		return 0, err
	}

	id, _ := result.LastInsertId()
	return id, nil
}

// SaveFuncNodes 批量保存函数节点
func (r *CallGraphRepository) SaveFuncNodes(_ context.Context, dbPath string, nodes []*codeanalysis.FuncNode) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO func_nodes (id, full_name, canonical_name, package, func_name, file_path, line_start, line_end, is_exported)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(full_name) DO UPDATE SET
			canonical_name = excluded.canonical_name,
			package = excluded.package,
			func_name = excluded.func_name,
			file_path = excluded.file_path,
			line_start = excluded.line_start,
			line_end = excluded.line_end,
			is_exported = excluded.is_exported
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, node := range nodes {
		_, err = stmt.Exec(node.ID, node.FullName, node.CanonicalName, node.Package, node.FuncName, node.FilePath, node.LineStart, node.LineEnd, node.IsExported)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SaveFuncEdge 保存调用边
func (r *CallGraphRepository) SaveFuncEdge(_ context.Context, dbPath string, edge *codeanalysis.FuncEdge) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT OR IGNORE INTO func_edges (caller_id, callee_id, call_site_file, call_site_line)
		VALUES (?, ?, ?, ?)
	`, edge.CallerID, edge.CalleeID, edge.CallSiteFile, edge.CallSiteLine)

	return err
}

// SaveFuncEdges 批量保存调用边
func (r *CallGraphRepository) SaveFuncEdges(_ context.Context, dbPath string, edges []*codeanalysis.FuncEdge) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO func_edges (caller_id, callee_id, call_site_file, call_site_line)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, edge := range edges {
		_, err = stmt.Exec(edge.CallerID, edge.CalleeID, edge.CallSiteFile, edge.CallSiteLine)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SaveMetadata 保存元数据
func (r *CallGraphRepository) SaveMetadata(_ context.Context, dbPath string, key string, value string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO metadata (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)

	return err
}

// GetMetadata 获取元数据
func (r *CallGraphRepository) GetMetadata(_ context.Context, dbPath string, key string) (string, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close()

	var value string
	err = db.QueryRow("SELECT value FROM metadata WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", err
	}

	return value, nil
}

// GetFuncNodeByFullName 根据完整函数名获取节点
func (r *CallGraphRepository) GetFuncNodeByFullName(_ context.Context, dbPath string, fullName string) (*codeanalysis.FuncNode, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	node := &codeanalysis.FuncNode{}
	err = db.QueryRow(`
		SELECT id, full_name, canonical_name, package, func_name, file_path, line_start, line_end, is_exported
		FROM func_nodes WHERE full_name = ?
	`, fullName).Scan(&node.ID, &node.FullName, &node.CanonicalName, &node.Package, &node.FuncName, &node.FilePath, &node.LineStart, &node.LineEnd, &node.IsExported)

	if err != nil {
		return nil, err
	}

	return node, nil
}

// GetFuncNodesByFullNames 批量获取函数节点
func (r *CallGraphRepository) GetFuncNodesByFullNames(_ context.Context, dbPath string, fullNames []string) ([]*codeanalysis.FuncNode, error) {
	if len(fullNames) == 0 {
		return nil, nil
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// 构建 IN 查询
	placeholders := make([]string, len(fullNames))
	args := make([]interface{}, len(fullNames))
	for i, name := range fullNames {
		placeholders[i] = "?"
		args[i] = name
	}

	// 同时按 full_name 和 canonical_name 匹配，确保 SSA 格式和 Diff 格式都能命中
	query := fmt.Sprintf(`
		SELECT id, full_name, canonical_name, package, func_name, file_path, line_start, line_end, is_exported
		FROM func_nodes WHERE full_name IN (%s) OR canonical_name IN (%s)
	`, strings.Join(placeholders, ","), strings.Join(placeholders, ","))

	// 参数需要重复一次（分别给 full_name IN 和 canonical_name IN）
	doubleArgs := make([]interface{}, 0, len(args)*2)
	doubleArgs = append(doubleArgs, args...)
	doubleArgs = append(doubleArgs, args...)

	rows, err := db.Query(query, doubleArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[int64]bool)
	var nodes []*codeanalysis.FuncNode
	for rows.Next() {
		node := &codeanalysis.FuncNode{}
		err = rows.Scan(&node.ID, &node.FullName, &node.CanonicalName, &node.Package, &node.FuncName, &node.FilePath, &node.LineStart, &node.LineEnd, &node.IsExported)
		if err != nil {
			return nil, err
		}
		// 去重（full_name 和 canonical_name 可能匹配到同一行）
		if !seen[node.ID] {
			seen[node.ID] = true
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// GetFuncNodeByFile 获取文件中的所有函数节点
func (r *CallGraphRepository) GetFuncNodeByFile(_ context.Context, dbPath string, filePath string) ([]*codeanalysis.FuncNode, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, full_name, canonical_name, package, func_name, file_path, line_start, line_end, is_exported
		FROM func_nodes WHERE file_path = ?
	`, filePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*codeanalysis.FuncNode
	for rows.Next() {
		node := &codeanalysis.FuncNode{}
		err = rows.Scan(&node.ID, &node.FullName, &node.CanonicalName, &node.Package, &node.FuncName, &node.FilePath, &node.LineStart, &node.LineEnd, &node.IsExported)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetCallers 获取函数的所有直接调用者
func (r *CallGraphRepository) GetCallers(_ context.Context, dbPath string, funcID int64) ([]*codeanalysis.FuncNode, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT fn.id, fn.full_name, fn.canonical_name, fn.package, fn.func_name, fn.file_path, fn.line_start, fn.line_end, fn.is_exported
		FROM func_nodes fn
		JOIN func_edges fe ON fe.caller_id = fn.id
		WHERE fe.callee_id = ?
	`, funcID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*codeanalysis.FuncNode
	for rows.Next() {
		node := &codeanalysis.FuncNode{}
		err = rows.Scan(&node.ID, &node.FullName, &node.CanonicalName, &node.Package, &node.FuncName, &node.FilePath, &node.LineStart, &node.LineEnd, &node.IsExported)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetCallersWithDepth 递归获取函数的调用者（带深度限制）
func (r *CallGraphRepository) GetCallersWithDepth(_ context.Context, dbPath string, funcIDs []int64, maxDepth int) ([]codeanalysis.CallerInfo, error) {
	if len(funcIDs) == 0 {
		return nil, nil
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// 构建 ID 列表
	idStrings := make([]string, len(funcIDs))
	for i, id := range funcIDs {
		idStrings[i] = fmt.Sprintf("%d", id)
	}
	idList := strings.Join(idStrings, ",")

	// 使用递归 CTE 查询调用链
	query := fmt.Sprintf(`
		WITH RECURSIVE call_chain AS (
			-- 基础：起始函数的直接调用者
			SELECT 
				fn.id,
				fn.full_name,
				fn.func_name,
				fn.package,
				fn.file_path,
				fe.call_site_line as line,
				1 as depth
			FROM func_nodes fn
			JOIN func_edges fe ON fe.caller_id = fn.id
			WHERE fe.callee_id IN (%s)
			
			UNION ALL
			
			-- 递归：查找调用者的调用者
			SELECT 
				fn.id,
				fn.full_name,
				fn.func_name,
				fn.package,
				fn.file_path,
				fe.call_site_line as line,
				cc.depth + 1 as depth
			FROM func_nodes fn
			JOIN func_edges fe ON fe.caller_id = fn.id
			JOIN call_chain cc ON fe.callee_id = cc.id
			WHERE cc.depth < ?
		)
		SELECT DISTINCT full_name, func_name, package, file_path, line, depth
		FROM call_chain
		ORDER BY depth, full_name
	`, idList)

	rows, err := db.Query(query, maxDepth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var callers []codeanalysis.CallerInfo
	for rows.Next() {
		var caller codeanalysis.CallerInfo
		err = rows.Scan(&caller.Function, &caller.DisplayName, &caller.Package, &caller.File, &caller.Line, &caller.Depth)
		if err != nil {
			return nil, err
		}
		callers = append(callers, caller)
	}

	return callers, nil
}

// GetFuncCount 获取函数数量
func (r *CallGraphRepository) GetFuncCount(_ context.Context, dbPath string) (int, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM func_nodes").Scan(&count)
	return count, err
}

// GetEdgeCount 获取调用边数量
func (r *CallGraphRepository) GetEdgeCount(_ context.Context, dbPath string) (int, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM func_edges").Scan(&count)
	return count, err
}

// SearchFunctions 多维搜索函数节点
// 支持按文件+行号、canonical_name、包名+函数名、函数短名模糊搜索
func (r *CallGraphRepository) SearchFunctions(_ context.Context, dbPath string, filePath string, line int, fullName string, pkg string, funcName string, limit int) ([]*codeanalysis.FuncNode, error) {
	if limit <= 0 {
		limit = 20
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var query string
	var args []interface{}

	switch {
	case filePath != "" && line > 0:
		// 按文件路径 + 行号精确定位
		query = `
			SELECT id, full_name, canonical_name, package, func_name, file_path, line_start, line_end, is_exported
			FROM func_nodes
			WHERE file_path = ? AND line_start <= ? AND line_end >= ?
			LIMIT ?`
		args = []interface{}{filePath, line, line, limit}

	case filePath != "":
		// 按文件路径搜索（返回该文件所有函数）
		query = `
			SELECT id, full_name, canonical_name, package, func_name, file_path, line_start, line_end, is_exported
			FROM func_nodes
			WHERE file_path = ?
			ORDER BY line_start
			LIMIT ?`
		args = []interface{}{filePath, limit}

	case fullName != "":
		// 按 canonical_name 精确匹配
		query = `
			SELECT id, full_name, canonical_name, package, func_name, file_path, line_start, line_end, is_exported
			FROM func_nodes
			WHERE canonical_name = ? OR full_name = ?
			LIMIT ?`
		args = []interface{}{fullName, fullName, limit}

	case pkg != "" && funcName != "":
		// 按包名 + 函数短名精确匹配
		query = `
			SELECT id, full_name, canonical_name, package, func_name, file_path, line_start, line_end, is_exported
			FROM func_nodes
			WHERE package = ? AND func_name = ?
			LIMIT ?`
		args = []interface{}{pkg, funcName, limit}

	case funcName != "":
		// 按函数短名模糊搜索
		query = `
			SELECT id, full_name, canonical_name, package, func_name, file_path, line_start, line_end, is_exported
			FROM func_nodes
			WHERE func_name LIKE ?
			ORDER BY is_exported DESC, func_name
			LIMIT ?`
		args = []interface{}{"%" + funcName + "%", limit}

	default:
		return nil, fmt.Errorf("at least one search parameter is required")
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*codeanalysis.FuncNode
	for rows.Next() {
		node := &codeanalysis.FuncNode{}
		err = rows.Scan(&node.ID, &node.FullName, &node.CanonicalName, &node.Package, &node.FuncName, &node.FilePath, &node.LineStart, &node.LineEnd, &node.IsExported)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetCalleesWithDepth 递归获取函数调用的下游函数（带深度限制）
func (r *CallGraphRepository) GetCalleesWithDepth(_ context.Context, dbPath string, funcIDs []int64, maxDepth int) ([]codeanalysis.CallerInfo, error) {
	if len(funcIDs) == 0 {
		return nil, nil
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// 构建 ID 列表
	idStrings := make([]string, len(funcIDs))
	for i, id := range funcIDs {
		idStrings[i] = fmt.Sprintf("%d", id)
	}
	idList := strings.Join(idStrings, ",")

	// 使用递归 CTE 查询下游调用链
	query := fmt.Sprintf(`
		WITH RECURSIVE call_chain AS (
			-- 基础：起始函数直接调用的函数
			SELECT 
				fn.id,
				fn.full_name,
				fn.func_name,
				fn.package,
				fn.file_path,
				fe.call_site_line as line,
				1 as depth
			FROM func_nodes fn
			JOIN func_edges fe ON fe.callee_id = fn.id
			WHERE fe.caller_id IN (%s)
			
			UNION ALL
			
			-- 递归：查找被调用者的被调用者
			SELECT 
				fn.id,
				fn.full_name,
				fn.func_name,
				fn.package,
				fn.file_path,
				fe.call_site_line as line,
				cc.depth + 1 as depth
			FROM func_nodes fn
			JOIN func_edges fe ON fe.callee_id = fn.id
			JOIN call_chain cc ON fe.caller_id = cc.id
			WHERE cc.depth < ?
		)
		SELECT DISTINCT full_name, func_name, package, file_path, line, depth
		FROM call_chain
		ORDER BY depth, full_name
	`, idList)

	rows, err := db.Query(query, maxDepth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var callees []codeanalysis.CallerInfo
	for rows.Next() {
		var callee codeanalysis.CallerInfo
		err = rows.Scan(&callee.Function, &callee.DisplayName, &callee.Package, &callee.File, &callee.Line, &callee.Depth)
		if err != nil {
			return nil, err
		}
		callees = append(callees, callee)
	}

	return callees, nil
}

// 确保实现接口
var _ codeanalysis.CallGraphRepository = (*CallGraphRepository)(nil)
