package team

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cocursor/backend/internal/domain/p2p"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/google/uuid"
)

// SessionSharingService 会话分享服务
type SessionSharingService struct {
	mu sync.RWMutex

	teamService *TeamService

	// 共享会话仓储（Leader 端存储）
	sharedSessionRepo storage.SharedSessionRepository

	// 数据库连接（用于初始化仓储）
	db *sql.DB

	logger *slog.Logger
}

// NewSessionSharingService 创建会话分享服务
func NewSessionSharingService(teamService *TeamService, db *sql.DB) (*SessionSharingService, error) {
	repo := storage.NewSharedSessionRepository(db)

	// 初始化表结构
	if err := repo.InitTables(); err != nil {
		return nil, fmt.Errorf("failed to init shared session tables: %w", err)
	}

	return &SessionSharingService{
		teamService:       teamService,
		sharedSessionRepo: repo,
		db:                db,
		logger:            log.NewModuleLogger("team", "session_sharing"),
	}, nil
}

// ShareSession 分享会话到团队
func (s *SessionSharingService) ShareSession(ctx context.Context, teamID string, req *domainTeam.ShareSessionRequest, sharerID, sharerName string) (string, error) {
	// 验证请求
	if err := req.Validate(); err != nil {
		return "", fmt.Errorf("invalid request: %w", err)
	}

	// 获取团队信息
	team, err := s.teamService.GetTeam(teamID)
	if err != nil {
		return "", fmt.Errorf("team not found: %w", err)
	}

	// 计算消息数量
	var messages []interface{}
	if err := json.Unmarshal(req.Messages, &messages); err != nil {
		return "", fmt.Errorf("invalid messages format: %w", err)
	}
	messageCount := len(messages)

	// 创建分享记录
	sharedSession := &domainTeam.SharedSession{
		ID:           uuid.New().String(),
		TeamID:       teamID,
		SharerID:     sharerID,
		SharerName:   sharerName,
		SessionID:    req.SessionID,
		Title:        req.Title,
		Messages:     req.Messages,
		MessageCount: messageCount,
		Description:  req.Description,
		SharedAt:     time.Now(),
		CommentCount: 0,
	}

	// 如果是 Leader，直接存储并广播
	if team.IsLeader {
		if err := s.sharedSessionRepo.CreateSharedSession(sharedSession); err != nil {
			return "", fmt.Errorf("failed to save shared session: %w", err)
		}

		// 广播分享事件
		payload := &p2p.SessionSharedPayload{
			ShareID:      sharedSession.ID,
			SharerID:     sharerID,
			SharerName:   sharerName,
			Title:        req.Title,
			MessageCount: messageCount,
			Description:  req.Description,
		}
		if err := s.teamService.BroadcastEvent(teamID, p2p.EventSessionShared, payload); err != nil {
			s.logger.Warn("failed to broadcast session shared event",
				"error", err,
				"team_id", teamID,
				"share_id", sharedSession.ID,
			)
		}

		s.logger.Info("session shared successfully",
			"share_id", sharedSession.ID,
			"team_id", teamID,
			"sharer_id", sharerID,
			"title", req.Title,
		)

		return sharedSession.ID, nil
	}

	// 非 Leader，需要将请求转发到 Leader（由 HTTP handler 处理）
	return "", fmt.Errorf("only leader can store shared sessions, please forward to leader")
}

// GetSharedSessions 获取团队分享的会话列表
func (s *SessionSharingService) GetSharedSessions(ctx context.Context, teamID string, limit, offset int) ([]domainTeam.SharedSessionListItem, int, error) {
	// 验证团队存在
	_, err := s.teamService.GetTeam(teamID)
	if err != nil {
		return nil, 0, fmt.Errorf("team not found: %w", err)
	}

	// 设置默认值
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return s.sharedSessionRepo.FindByTeamID(teamID, limit, offset)
}

// GetSharedSessionDetail 获取分享会话详情
func (s *SessionSharingService) GetSharedSessionDetail(ctx context.Context, teamID, shareID string) (*domainTeam.SharedSession, []domainTeam.SessionComment, error) {
	// 验证团队存在
	_, err := s.teamService.GetTeam(teamID)
	if err != nil {
		return nil, nil, fmt.Errorf("team not found: %w", err)
	}

	// 获取分享详情
	session, err := s.sharedSessionRepo.FindByID(shareID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get shared session: %w", err)
	}
	if session == nil {
		return nil, nil, fmt.Errorf("shared session not found")
	}

	// 验证会话属于该团队
	if session.TeamID != teamID {
		return nil, nil, fmt.Errorf("shared session not found in this team")
	}

	// 获取评论列表
	comments, err := s.sharedSessionRepo.FindCommentsByShareID(shareID)
	if err != nil {
		s.logger.Warn("failed to get comments",
			"error", err,
			"share_id", shareID,
		)
		comments = []domainTeam.SessionComment{}
	}

	return session, comments, nil
}

// AddComment 添加评论
func (s *SessionSharingService) AddComment(ctx context.Context, teamID, shareID string, req *domainTeam.AddCommentRequest, authorID, authorName string) (string, error) {
	// 验证请求
	if err := req.Validate(); err != nil {
		return "", fmt.Errorf("invalid request: %w", err)
	}

	// 获取团队信息
	team, err := s.teamService.GetTeam(teamID)
	if err != nil {
		return "", fmt.Errorf("team not found: %w", err)
	}

	// 验证分享会话存在
	session, err := s.sharedSessionRepo.FindByID(shareID)
	if err != nil {
		return "", fmt.Errorf("failed to get shared session: %w", err)
	}
	if session == nil {
		return "", fmt.Errorf("shared session not found")
	}
	if session.TeamID != teamID {
		return "", fmt.Errorf("shared session not found in this team")
	}

	// 创建评论
	comment := &domainTeam.SessionComment{
		ID:         uuid.New().String(),
		ShareID:    shareID,
		AuthorID:   authorID,
		AuthorName: authorName,
		Content:    req.Content,
		Mentions:   req.Mentions,
		CreatedAt:  time.Now(),
	}

	// 如果是 Leader，直接存储并广播
	if team.IsLeader {
		if err := s.sharedSessionRepo.CreateComment(comment); err != nil {
			return "", fmt.Errorf("failed to save comment: %w", err)
		}

		// 更新评论数
		if err := s.sharedSessionRepo.IncrementCommentCount(shareID); err != nil {
			s.logger.Warn("failed to increment comment count",
				"error", err,
				"share_id", shareID,
			)
		}

		// 广播评论事件
		payload := &p2p.SessionCommentAddedPayload{
			ShareID:    shareID,
			CommentID:  comment.ID,
			AuthorID:   authorID,
			AuthorName: authorName,
			Content:    req.Content,
			Mentions:   req.Mentions,
		}
		if err := s.teamService.BroadcastEvent(teamID, p2p.EventSessionCommentAdded, payload); err != nil {
			s.logger.Warn("failed to broadcast comment added event",
				"error", err,
				"team_id", teamID,
				"comment_id", comment.ID,
			)
		}

		s.logger.Info("comment added successfully",
			"comment_id", comment.ID,
			"share_id", shareID,
			"author_id", authorID,
		)

		return comment.ID, nil
	}

	// 非 Leader，需要将请求转发到 Leader（由 HTTP handler 处理）
	return "", fmt.Errorf("only leader can store comments, please forward to leader")
}

// GetComments 获取评论列表
func (s *SessionSharingService) GetComments(ctx context.Context, shareID string) ([]domainTeam.SessionComment, error) {
	return s.sharedSessionRepo.FindCommentsByShareID(shareID)
}

// HandleSessionSharedEvent 处理会话分享事件（Leader 端）
func (s *SessionSharingService) HandleSessionSharedEvent(teamID string, payload *p2p.SessionSharedPayload) {
	s.logger.Info("received session shared event",
		"team_id", teamID,
		"share_id", payload.ShareID,
		"sharer_name", payload.SharerName,
	)
	// Leader 已在 ShareSession 中处理存储和广播，这里主要用于日志
}

// HandleCommentAddedEvent 处理评论新增事件（Leader 端）
func (s *SessionSharingService) HandleCommentAddedEvent(teamID string, payload *p2p.SessionCommentAddedPayload) {
	s.logger.Info("received comment added event",
		"team_id", teamID,
		"share_id", payload.ShareID,
		"comment_id", payload.CommentID,
		"author_name", payload.AuthorName,
	)
	// Leader 已在 AddComment 中处理存储和广播，这里主要用于日志
}
