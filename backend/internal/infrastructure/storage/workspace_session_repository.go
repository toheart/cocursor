package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// WorkspaceSession 工作区会话实体
type WorkspaceSession struct {
	ID                  int64
	WorkspaceID         string
	ComposerID          string
	Name                string
	Type                string
	CreatedAt           int64 // 毫秒时间戳
	LastUpdatedAt       int64 // 毫秒时间戳
	UnifiedMode         string
	Subtitle            string
	TotalLinesAdded     int
	TotalLinesRemoved   int
	FilesChangedCount   int
	ContextUsagePercent float64
	IsArchived          bool
	CreatedOnBranch     string
	TokenCount          int   // 会话的 Token 数量
	CachedAt            int64 // 毫秒时间戳
	// 运行时状态字段
	IsVisible   bool   // 是否可见（面板打开）
	IsFocused   bool   // 是否聚焦（当前活跃）
	ActiveLevel int    // 活跃等级：0=聚焦, 1=打开, 2=关闭, 3=归档
	PanelID     string // 对应的面板 ID
}

// WorkspaceFileMetadata 工作区文件元数据实体
type WorkspaceFileMetadata struct {
	WorkspaceID  string
	DBPath       string
	FileMtime    int64 // Unix 时间戳
	FileSize     int64
	LastScanTime int64 // Unix 时间戳
	LastSyncTime *int64 // Unix 时间戳，可为空
	SessionsCount int
	CreatedAt    int64 // Unix 时间戳
	UpdatedAt    int64 // Unix 时间戳
}

// DailyTokenUsage 每日 Token 使用统计
type DailyTokenUsage struct {
	Date       string // 日期 YYYY-MM-DD
	TokenCount int    // 当日 Token 总数
}

// RuntimeStateUpdate 运行时状态更新参数
type RuntimeStateUpdate struct {
	ComposerID  string
	IsVisible   bool
	IsFocused   bool
	ActiveLevel int
	PanelID     string
}

// WorkspaceSessionRepository 工作区会话仓储接口
type WorkspaceSessionRepository interface {
	// Save 保存或更新会话（upsert）
	Save(session *WorkspaceSession) error
	// FindByWorkspaceID 查询指定工作区的所有会话
	FindByWorkspaceID(workspaceID string) ([]*WorkspaceSession, error)
	// FindByWorkspaceIDAndComposerID 查询指定工作区和会话 ID 的会话
	FindByWorkspaceIDAndComposerID(workspaceID, composerID string) (*WorkspaceSession, error)
	// FindByWorkspacesAndDateRange 查询多个工作区在日期范围内的会话
	FindByWorkspacesAndDateRange(workspaceIDs []string, startDate, endDate string) ([]*WorkspaceSession, error)
	// FindByWorkspaces 查询多个工作区的所有会话（用于会话列表）
	FindByWorkspaces(workspaceIDs []string, search string, limit, offset int) ([]*WorkspaceSession, int, error)
	// GetCachedComposerIDs 获取已缓存的 composer_id 列表
	GetCachedComposerIDs(workspaceID string) ([]string, error)
	// GetDailyTokenUsage 按日期聚合 Token 使用量
	GetDailyTokenUsage(workspaceIDs []string, startDate, endDate string) ([]*DailyTokenUsage, error)
	// UpdateRuntimeState 批量更新运行时状态
	UpdateRuntimeState(workspaceID string, updates []*RuntimeStateUpdate) error
	// ResetRuntimeState 重置工作区所有会话的运行时状态（用于扫描前清理）
	ResetRuntimeState(workspaceID string) error
	// FindActiveByWorkspaceID 查询指定工作区的活跃会话（按 active_level 排序）
	FindActiveByWorkspaceID(workspaceID string) ([]*WorkspaceSession, error)
}

// WorkspaceFileMetadataRepository 工作区文件元数据仓储接口
type WorkspaceFileMetadataRepository interface {
	// Save 保存或更新元数据（upsert）
	Save(metadata *WorkspaceFileMetadata) error
	// FindByWorkspaceID 查询指定工作区的元数据
	FindByWorkspaceID(workspaceID string) (*WorkspaceFileMetadata, error)
	// FindAllWorkspaceIDs 查询所有已缓存的工作区 ID
	FindAllWorkspaceIDs() ([]string, error)
}

// workspaceSessionRepository 工作区会话仓储实现
type workspaceSessionRepository struct {
	db *sql.DB
}

// workspaceFileMetadataRepository 工作区文件元数据仓储实现
type workspaceFileMetadataRepository struct {
	db *sql.DB
}

// NewWorkspaceSessionRepository 创建工作区会话仓储实例（接受数据库连接作为参数）
func NewWorkspaceSessionRepository(db *sql.DB) WorkspaceSessionRepository {
	return &workspaceSessionRepository{
		db: db,
	}
}

// NewWorkspaceFileMetadataRepository 创建工作区文件元数据仓储实例（接受数据库连接作为参数）
func NewWorkspaceFileMetadataRepository(db *sql.DB) WorkspaceFileMetadataRepository {
	return &workspaceFileMetadataRepository{
		db: db,
	}
}

// Save 保存或更新会话（upsert）
func (r *workspaceSessionRepository) Save(session *WorkspaceSession) error {
	query := `
		INSERT OR REPLACE INTO workspace_sessions 
		(workspace_id, composer_id, name, type, created_at, last_updated_at, unified_mode, subtitle,
		 total_lines_added, total_lines_removed, files_changed_count, context_usage_percent,
		 is_archived, created_on_branch, token_count, cached_at,
		 is_visible, is_focused, active_level, panel_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		session.WorkspaceID,
		session.ComposerID,
		session.Name,
		session.Type,
		session.CreatedAt,
		session.LastUpdatedAt,
		session.UnifiedMode,
		session.Subtitle,
		session.TotalLinesAdded,
		session.TotalLinesRemoved,
		session.FilesChangedCount,
		session.ContextUsagePercent,
		session.IsArchived,
		session.CreatedOnBranch,
		session.TokenCount,
		session.CachedAt,
		session.IsVisible,
		session.IsFocused,
		session.ActiveLevel,
		session.PanelID,
	)

	if err != nil {
		return fmt.Errorf("failed to save workspace session: %w", err)
	}

	return nil
}

// FindByWorkspaceID 查询指定工作区的所有会话
func (r *workspaceSessionRepository) FindByWorkspaceID(workspaceID string) ([]*WorkspaceSession, error) {
	query := `
		SELECT id, workspace_id, composer_id, name, type, created_at, last_updated_at, unified_mode, subtitle,
		       total_lines_added, total_lines_removed, files_changed_count, context_usage_percent,
		       is_archived, created_on_branch, COALESCE(token_count, 0), cached_at,
		       COALESCE(is_visible, 0), COALESCE(is_focused, 0), COALESCE(active_level, 2), COALESCE(panel_id, '')
		FROM workspace_sessions
		WHERE workspace_id = ?
		ORDER BY last_updated_at DESC`

	rows, err := r.db.Query(query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspace sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*WorkspaceSession
	for rows.Next() {
		session := &WorkspaceSession{}
		var isArchivedInt, isVisibleInt, isFocusedInt int

		if err := rows.Scan(
			&session.ID,
			&session.WorkspaceID,
			&session.ComposerID,
			&session.Name,
			&session.Type,
			&session.CreatedAt,
			&session.LastUpdatedAt,
			&session.UnifiedMode,
			&session.Subtitle,
			&session.TotalLinesAdded,
			&session.TotalLinesRemoved,
			&session.FilesChangedCount,
			&session.ContextUsagePercent,
			&isArchivedInt,
			&session.CreatedOnBranch,
			&session.TokenCount,
			&session.CachedAt,
			&isVisibleInt,
			&isFocusedInt,
			&session.ActiveLevel,
			&session.PanelID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan workspace session: %w", err)
		}

		session.IsArchived = isArchivedInt == 1
		session.IsVisible = isVisibleInt == 1
		session.IsFocused = isFocusedInt == 1
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// FindByWorkspaceIDAndComposerID 查询指定工作区和会话 ID 的会话
func (r *workspaceSessionRepository) FindByWorkspaceIDAndComposerID(workspaceID, composerID string) (*WorkspaceSession, error) {
	query := `
		SELECT id, workspace_id, composer_id, name, type, created_at, last_updated_at, unified_mode, subtitle,
		       total_lines_added, total_lines_removed, files_changed_count, context_usage_percent,
		       is_archived, created_on_branch, COALESCE(token_count, 0), cached_at,
		       COALESCE(is_visible, 0), COALESCE(is_focused, 0), COALESCE(active_level, 2), COALESCE(panel_id, '')
		FROM workspace_sessions
		WHERE workspace_id = ? AND composer_id = ?`

	session := &WorkspaceSession{}
	var isArchivedInt, isVisibleInt, isFocusedInt int

	err := r.db.QueryRow(query, workspaceID, composerID).Scan(
		&session.ID,
		&session.WorkspaceID,
		&session.ComposerID,
		&session.Name,
		&session.Type,
		&session.CreatedAt,
		&session.LastUpdatedAt,
		&session.UnifiedMode,
		&session.Subtitle,
		&session.TotalLinesAdded,
		&session.TotalLinesRemoved,
		&session.FilesChangedCount,
		&session.ContextUsagePercent,
		&isArchivedInt,
		&session.CreatedOnBranch,
		&session.TokenCount,
		&session.CachedAt,
		&isVisibleInt,
		&isFocusedInt,
		&session.ActiveLevel,
		&session.PanelID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query workspace session: %w", err)
	}

	session.IsArchived = isArchivedInt == 1
	session.IsVisible = isVisibleInt == 1
	session.IsFocused = isFocusedInt == 1
	return session, nil
}

// FindByWorkspacesAndDateRange 查询多个工作区在日期范围内的会话
func (r *workspaceSessionRepository) FindByWorkspacesAndDateRange(workspaceIDs []string, startDate, endDate string) ([]*WorkspaceSession, error) {
	if len(workspaceIDs) == 0 {
		return []*WorkspaceSession{}, nil
	}

	// 解析日期范围
	startTime, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date: %w", err)
	}
	endTime, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date: %w", err)
	}

	// 转换为毫秒时间戳范围
	startMs := startTime.UnixMilli()
	endMs := endTime.AddDate(0, 0, 1).UnixMilli() - 1 // 包含结束日期当天

	// 构建查询（使用 IN 子句）
	query := `
		SELECT id, workspace_id, composer_id, name, type, created_at, last_updated_at, unified_mode, subtitle,
		       total_lines_added, total_lines_removed, files_changed_count, context_usage_percent,
		       is_archived, created_on_branch, COALESCE(token_count, 0), cached_at,
		       COALESCE(is_visible, 0), COALESCE(is_focused, 0), COALESCE(active_level, 2), COALESCE(panel_id, '')
		FROM workspace_sessions
		WHERE workspace_id IN (`
	
	args := make([]interface{}, 0, len(workspaceIDs)+2)
	for i, wsID := range workspaceIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, wsID)
	}
	
	query += `) AND (
			(created_at >= ? AND created_at <= ?) OR
			(last_updated_at >= ? AND last_updated_at <= ?)
		)
		ORDER BY last_updated_at DESC`

	args = append(args, startMs, endMs, startMs, endMs)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspace sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*WorkspaceSession
	for rows.Next() {
		session := &WorkspaceSession{}
		var isArchivedInt, isVisibleInt, isFocusedInt int

		if err := rows.Scan(
			&session.ID,
			&session.WorkspaceID,
			&session.ComposerID,
			&session.Name,
			&session.Type,
			&session.CreatedAt,
			&session.LastUpdatedAt,
			&session.UnifiedMode,
			&session.Subtitle,
			&session.TotalLinesAdded,
			&session.TotalLinesRemoved,
			&session.FilesChangedCount,
			&session.ContextUsagePercent,
			&isArchivedInt,
			&session.CreatedOnBranch,
			&session.TokenCount,
			&session.CachedAt,
			&isVisibleInt,
			&isFocusedInt,
			&session.ActiveLevel,
			&session.PanelID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan workspace session: %w", err)
		}

		session.IsArchived = isArchivedInt == 1
		session.IsVisible = isVisibleInt == 1
		session.IsFocused = isFocusedInt == 1
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// FindByWorkspaces 查询多个工作区的所有会话（用于会话列表）
func (r *workspaceSessionRepository) FindByWorkspaces(workspaceIDs []string, search string, limit, offset int) ([]*WorkspaceSession, int, error) {
	if len(workspaceIDs) == 0 {
		return []*WorkspaceSession{}, 0, nil
	}

	// 构建基础查询
	baseQuery := `
		SELECT id, workspace_id, composer_id, name, type, created_at, last_updated_at, unified_mode, subtitle,
		       total_lines_added, total_lines_removed, files_changed_count, context_usage_percent,
		       is_archived, created_on_branch, COALESCE(token_count, 0), cached_at,
		       COALESCE(is_visible, 0), COALESCE(is_focused, 0), COALESCE(active_level, 2), COALESCE(panel_id, '')
		FROM workspace_sessions
		WHERE workspace_id IN (`
	
	args := make([]interface{}, 0)
	for i, wsID := range workspaceIDs {
		if i > 0 {
			baseQuery += ","
		}
		baseQuery += "?"
		args = append(args, wsID)
	}
	baseQuery += ")"

	// 添加搜索条件
	if search != "" {
		baseQuery += " AND LOWER(name) LIKE LOWER(?)"
		args = append(args, "%"+search+"%")
	}

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM workspace_sessions WHERE workspace_id IN ("
	for i := range workspaceIDs {
		if i > 0 {
			countQuery += ","
		}
		countQuery += "?"
	}
	countQuery += ")"
	if search != "" {
		countQuery += " AND LOWER(name) LIKE LOWER(?)"
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count workspace sessions: %w", err)
	}

	// 添加排序和分页
	baseQuery += " ORDER BY last_updated_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(baseQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query workspace sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*WorkspaceSession
	for rows.Next() {
		session := &WorkspaceSession{}
		var isArchivedInt, isVisibleInt, isFocusedInt int

		if err := rows.Scan(
			&session.ID,
			&session.WorkspaceID,
			&session.ComposerID,
			&session.Name,
			&session.Type,
			&session.CreatedAt,
			&session.LastUpdatedAt,
			&session.UnifiedMode,
			&session.Subtitle,
			&session.TotalLinesAdded,
			&session.TotalLinesRemoved,
			&session.FilesChangedCount,
			&session.ContextUsagePercent,
			&isArchivedInt,
			&session.CreatedOnBranch,
			&session.TokenCount,
			&session.CachedAt,
			&isVisibleInt,
			&isFocusedInt,
			&session.ActiveLevel,
			&session.PanelID,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan workspace session: %w", err)
		}

		session.IsArchived = isArchivedInt == 1
		session.IsVisible = isVisibleInt == 1
		session.IsFocused = isFocusedInt == 1
		sessions = append(sessions, session)
	}

	return sessions, total, nil
}

// GetCachedComposerIDs 获取已缓存的 composer_id 列表
func (r *workspaceSessionRepository) GetCachedComposerIDs(workspaceID string) ([]string, error) {
	query := `
		SELECT composer_id
		FROM workspace_sessions
		WHERE workspace_id = ?`

	rows, err := r.db.Query(query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cached composer IDs: %w", err)
	}
	defer rows.Close()

	var composerIDs []string
	for rows.Next() {
		var composerID string
		if err := rows.Scan(&composerID); err != nil {
			return nil, fmt.Errorf("failed to scan composer ID: %w", err)
		}
		composerIDs = append(composerIDs, composerID)
	}

	return composerIDs, nil
}

// GetDailyTokenUsage 按日期聚合 Token 使用量
func (r *workspaceSessionRepository) GetDailyTokenUsage(workspaceIDs []string, startDate, endDate string) ([]*DailyTokenUsage, error) {
	if len(workspaceIDs) == 0 {
		return []*DailyTokenUsage{}, nil
	}

	// 解析日期范围
	startTime, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date: %w", err)
	}
	endTime, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date: %w", err)
	}

	// 转换为毫秒时间戳范围
	startMs := startTime.UnixMilli()
	endMs := endTime.AddDate(0, 0, 1).UnixMilli() - 1 // 包含结束日期当天

	// 构建查询：按日期聚合 Token
	// 使用 last_updated_at 作为统计日期（会话活跃时间）
	query := `
		SELECT 
			DATE(last_updated_at/1000, 'unixepoch', 'localtime') as date,
			SUM(COALESCE(token_count, 0)) as total_tokens
		FROM workspace_sessions
		WHERE workspace_id IN (`

	args := make([]interface{}, 0, len(workspaceIDs)+2)
	for i, wsID := range workspaceIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, wsID)
	}

	query += `) AND last_updated_at >= ? AND last_updated_at <= ?
		GROUP BY DATE(last_updated_at/1000, 'unixepoch', 'localtime')
		ORDER BY date`

	args = append(args, startMs, endMs)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily token usage: %w", err)
	}
	defer rows.Close()

	var result []*DailyTokenUsage
	for rows.Next() {
		usage := &DailyTokenUsage{}
		if err := rows.Scan(&usage.Date, &usage.TokenCount); err != nil {
			return nil, fmt.Errorf("failed to scan daily token usage: %w", err)
		}
		result = append(result, usage)
	}

	return result, nil
}

// UpdateRuntimeState 批量更新运行时状态
func (r *workspaceSessionRepository) UpdateRuntimeState(workspaceID string, updates []*RuntimeStateUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	// 使用事务批量更新
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		UPDATE workspace_sessions 
		SET is_visible = ?, is_focused = ?, active_level = ?, panel_id = ?
		WHERE workspace_id = ? AND composer_id = ?`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, update := range updates {
		_, err := stmt.Exec(
			update.IsVisible,
			update.IsFocused,
			update.ActiveLevel,
			update.PanelID,
			workspaceID,
			update.ComposerID,
		)
		if err != nil {
			return fmt.Errorf("failed to update runtime state for %s: %w", update.ComposerID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ResetRuntimeState 重置工作区所有会话的运行时状态（用于扫描前清理）
func (r *workspaceSessionRepository) ResetRuntimeState(workspaceID string) error {
	query := `
		UPDATE workspace_sessions 
		SET is_visible = 0, is_focused = 0, active_level = CASE WHEN is_archived = 1 THEN 3 ELSE 2 END
		WHERE workspace_id = ?`

	_, err := r.db.Exec(query, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to reset runtime state: %w", err)
	}

	return nil
}

// FindActiveByWorkspaceID 查询指定工作区的活跃会话（按 active_level 排序）
func (r *workspaceSessionRepository) FindActiveByWorkspaceID(workspaceID string) ([]*WorkspaceSession, error) {
	query := `
		SELECT id, workspace_id, composer_id, name, type, created_at, last_updated_at, unified_mode, subtitle,
		       total_lines_added, total_lines_removed, files_changed_count, context_usage_percent,
		       is_archived, created_on_branch, COALESCE(token_count, 0), cached_at,
		       COALESCE(is_visible, 0), COALESCE(is_focused, 0), COALESCE(active_level, 2), COALESCE(panel_id, '')
		FROM workspace_sessions
		WHERE workspace_id = ? AND active_level <= 1
		ORDER BY active_level ASC, context_usage_percent DESC`

	rows, err := r.db.Query(query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*WorkspaceSession
	for rows.Next() {
		session := &WorkspaceSession{}
		var isArchivedInt, isVisibleInt, isFocusedInt int

		if err := rows.Scan(
			&session.ID,
			&session.WorkspaceID,
			&session.ComposerID,
			&session.Name,
			&session.Type,
			&session.CreatedAt,
			&session.LastUpdatedAt,
			&session.UnifiedMode,
			&session.Subtitle,
			&session.TotalLinesAdded,
			&session.TotalLinesRemoved,
			&session.FilesChangedCount,
			&session.ContextUsagePercent,
			&isArchivedInt,
			&session.CreatedOnBranch,
			&session.TokenCount,
			&session.CachedAt,
			&isVisibleInt,
			&isFocusedInt,
			&session.ActiveLevel,
			&session.PanelID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan active session: %w", err)
		}

		session.IsArchived = isArchivedInt == 1
		session.IsVisible = isVisibleInt == 1
		session.IsFocused = isFocusedInt == 1
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Save 保存或更新元数据（upsert）
func (r *workspaceFileMetadataRepository) Save(metadata *WorkspaceFileMetadata) error {
	query := `
		INSERT OR REPLACE INTO workspace_file_metadata 
		(workspace_id, db_path, file_mtime, file_size, last_scan_time, last_sync_time, sessions_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var lastSyncTime interface{}
	if metadata.LastSyncTime != nil {
		lastSyncTime = *metadata.LastSyncTime
	}

	_, err := r.db.Exec(query,
		metadata.WorkspaceID,
		metadata.DBPath,
		metadata.FileMtime,
		metadata.FileSize,
		metadata.LastScanTime,
		lastSyncTime,
		metadata.SessionsCount,
		metadata.CreatedAt,
		metadata.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save workspace file metadata: %w", err)
	}

	return nil
}

// FindByWorkspaceID 查询指定工作区的元数据
func (r *workspaceFileMetadataRepository) FindByWorkspaceID(workspaceID string) (*WorkspaceFileMetadata, error) {
	query := `
		SELECT workspace_id, db_path, file_mtime, file_size, last_scan_time, last_sync_time, sessions_count, created_at, updated_at
		FROM workspace_file_metadata
		WHERE workspace_id = ?`

	metadata := &WorkspaceFileMetadata{}
	var lastSyncTime sql.NullInt64

	err := r.db.QueryRow(query, workspaceID).Scan(
		&metadata.WorkspaceID,
		&metadata.DBPath,
		&metadata.FileMtime,
		&metadata.FileSize,
		&metadata.LastScanTime,
		&lastSyncTime,
		&metadata.SessionsCount,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query workspace file metadata: %w", err)
	}

	if lastSyncTime.Valid {
		metadata.LastSyncTime = &[]int64{lastSyncTime.Int64}[0]
	}

	return metadata, nil
}

// FindAllWorkspaceIDs 查询所有已缓存的工作区 ID
func (r *workspaceFileMetadataRepository) FindAllWorkspaceIDs() ([]string, error) {
	query := `SELECT workspace_id FROM workspace_file_metadata`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspace IDs: %w", err)
	}
	defer rows.Close()

	var workspaceIDs []string
	for rows.Next() {
		var workspaceID string
		if err := rows.Scan(&workspaceID); err != nil {
			return nil, fmt.Errorf("failed to scan workspace ID: %w", err)
		}
		workspaceIDs = append(workspaceIDs, workspaceID)
	}

	return workspaceIDs, nil
}
