package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cocursor/backend/internal/domain/team"
	"github.com/google/uuid"
)

// SharedSessionRepository 共享会话仓储接口
type SharedSessionRepository interface {
	// 分享会话
	CreateSharedSession(session *team.SharedSession) error
	// 获取分享列表
	FindByTeamID(teamID string, limit, offset int) ([]team.SharedSessionListItem, int, error)
	// 获取分享详情
	FindByID(shareID string) (*team.SharedSession, error)
	// 添加评论
	CreateComment(comment *team.SessionComment) error
	// 获取评论列表
	FindCommentsByShareID(shareID string) ([]team.SessionComment, error)
	// 更新评论数
	IncrementCommentCount(shareID string) error
	// 初始化表结构
	InitTables() error
}

// sharedSessionRepository 共享会话仓储实现
type sharedSessionRepository struct {
	db *sql.DB
}

// NewSharedSessionRepository 创建共享会话仓储实例
func NewSharedSessionRepository(db *sql.DB) SharedSessionRepository {
	return &sharedSessionRepository{
		db: db,
	}
}

// InitTables 初始化表结构
func (r *sharedSessionRepository) InitTables() error {
	// 创建分享会话表
	createSharedSessionsTable := `
		CREATE TABLE IF NOT EXISTS shared_sessions (
			id TEXT PRIMARY KEY,
			team_id TEXT NOT NULL,
			sharer_id TEXT NOT NULL,
			sharer_name TEXT NOT NULL,
			session_id TEXT NOT NULL,
			title TEXT NOT NULL,
			messages TEXT NOT NULL,
			message_count INTEGER,
			description TEXT,
			shared_at INTEGER NOT NULL,
			comment_count INTEGER DEFAULT 0
		)`

	if _, err := r.db.Exec(createSharedSessionsTable); err != nil {
		return fmt.Errorf("failed to create shared_sessions table: %w", err)
	}

	// 创建评论表
	createCommentsTable := `
		CREATE TABLE IF NOT EXISTS session_comments (
			id TEXT PRIMARY KEY,
			share_id TEXT NOT NULL,
			author_id TEXT NOT NULL,
			author_name TEXT NOT NULL,
			content TEXT NOT NULL,
			mentions TEXT,
			created_at INTEGER NOT NULL,
			FOREIGN KEY (share_id) REFERENCES shared_sessions(id)
		)`

	if _, err := r.db.Exec(createCommentsTable); err != nil {
		return fmt.Errorf("failed to create session_comments table: %w", err)
	}

	// 创建索引
	createIndexTeam := `CREATE INDEX IF NOT EXISTS idx_shared_sessions_team ON shared_sessions(team_id, shared_at DESC)`
	if _, err := r.db.Exec(createIndexTeam); err != nil {
		return fmt.Errorf("failed to create index idx_shared_sessions_team: %w", err)
	}

	createIndexComments := `CREATE INDEX IF NOT EXISTS idx_session_comments_share ON session_comments(share_id, created_at)`
	if _, err := r.db.Exec(createIndexComments); err != nil {
		return fmt.Errorf("failed to create index idx_session_comments_share: %w", err)
	}

	return nil
}

// CreateSharedSession 创建分享会话
func (r *sharedSessionRepository) CreateSharedSession(session *team.SharedSession) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	query := `
		INSERT INTO shared_sessions 
		(id, team_id, sharer_id, sharer_name, session_id, title, messages, message_count, description, shared_at, comment_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		session.ID,
		session.TeamID,
		session.SharerID,
		session.SharerName,
		session.SessionID,
		session.Title,
		string(session.Messages),
		session.MessageCount,
		session.Description,
		session.SharedAt.UnixMilli(),
		session.CommentCount,
	)

	if err != nil {
		return fmt.Errorf("failed to create shared session: %w", err)
	}

	return nil
}

// FindByTeamID 按团队 ID 查询分享列表
func (r *sharedSessionRepository) FindByTeamID(teamID string, limit, offset int) ([]team.SharedSessionListItem, int, error) {
	// 查询总数
	var total int
	countQuery := `SELECT COUNT(*) FROM shared_sessions WHERE team_id = ?`
	if err := r.db.QueryRow(countQuery, teamID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count shared sessions: %w", err)
	}

	// 查询列表（不包含消息内容）
	query := `
		SELECT id, team_id, sharer_id, sharer_name, session_id, title, message_count, description, shared_at, comment_count
		FROM shared_sessions
		WHERE team_id = ?
		ORDER BY shared_at DESC
		LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, teamID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query shared sessions: %w", err)
	}
	defer rows.Close()

	var sessions []team.SharedSessionListItem
	for rows.Next() {
		var item team.SharedSessionListItem
		var description sql.NullString
		var sharedAt int64

		if err := rows.Scan(
			&item.ID,
			&item.TeamID,
			&item.SharerID,
			&item.SharerName,
			&item.SessionID,
			&item.Title,
			&item.MessageCount,
			&description,
			&sharedAt,
			&item.CommentCount,
		); err != nil {
			continue
		}

		if description.Valid {
			item.Description = description.String
		}
		item.SharedAt = time.UnixMilli(sharedAt)

		sessions = append(sessions, item)
	}

	return sessions, total, nil
}

// FindByID 按 ID 查询分享详情
func (r *sharedSessionRepository) FindByID(shareID string) (*team.SharedSession, error) {
	query := `
		SELECT id, team_id, sharer_id, sharer_name, session_id, title, messages, message_count, description, shared_at, comment_count
		FROM shared_sessions
		WHERE id = ?`

	var session team.SharedSession
	var messagesJSON string
	var description sql.NullString
	var sharedAt int64

	err := r.db.QueryRow(query, shareID).Scan(
		&session.ID,
		&session.TeamID,
		&session.SharerID,
		&session.SharerName,
		&session.SessionID,
		&session.Title,
		&messagesJSON,
		&session.MessageCount,
		&description,
		&sharedAt,
		&session.CommentCount,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query shared session: %w", err)
	}

	session.Messages = json.RawMessage(messagesJSON)
	if description.Valid {
		session.Description = description.String
	}
	session.SharedAt = time.UnixMilli(sharedAt)

	return &session, nil
}

// CreateComment 创建评论
func (r *sharedSessionRepository) CreateComment(comment *team.SessionComment) error {
	if comment.ID == "" {
		comment.ID = uuid.New().String()
	}

	// 序列化 mentions
	var mentionsJSON string
	if len(comment.Mentions) > 0 {
		mentionsBytes, err := json.Marshal(comment.Mentions)
		if err == nil {
			mentionsJSON = string(mentionsBytes)
		}
	}

	query := `
		INSERT INTO session_comments 
		(id, share_id, author_id, author_name, content, mentions, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		comment.ID,
		comment.ShareID,
		comment.AuthorID,
		comment.AuthorName,
		comment.Content,
		mentionsJSON,
		comment.CreatedAt.UnixMilli(),
	)

	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

// FindCommentsByShareID 按分享 ID 查询评论列表
func (r *sharedSessionRepository) FindCommentsByShareID(shareID string) ([]team.SessionComment, error) {
	query := `
		SELECT id, share_id, author_id, author_name, content, mentions, created_at
		FROM session_comments
		WHERE share_id = ?
		ORDER BY created_at ASC`

	rows, err := r.db.Query(query, shareID)
	if err != nil {
		return nil, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	var comments []team.SessionComment
	for rows.Next() {
		var comment team.SessionComment
		var mentionsJSON sql.NullString
		var createdAt int64

		if err := rows.Scan(
			&comment.ID,
			&comment.ShareID,
			&comment.AuthorID,
			&comment.AuthorName,
			&comment.Content,
			&mentionsJSON,
			&createdAt,
		); err != nil {
			continue
		}

		if mentionsJSON.Valid && mentionsJSON.String != "" {
			json.Unmarshal([]byte(mentionsJSON.String), &comment.Mentions)
		}
		comment.CreatedAt = time.UnixMilli(createdAt)

		comments = append(comments, comment)
	}

	return comments, nil
}

// IncrementCommentCount 增加评论数
func (r *sharedSessionRepository) IncrementCommentCount(shareID string) error {
	query := `UPDATE shared_sessions SET comment_count = comment_count + 1 WHERE id = ?`
	_, err := r.db.Exec(query, shareID)
	if err != nil {
		return fmt.Errorf("failed to increment comment count: %w", err)
	}
	return nil
}
