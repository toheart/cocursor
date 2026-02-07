package team

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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

	// HTTP 客户端（用于非 Leader 转发请求到 Leader）
	httpClient *http.Client

	logger *slog.Logger
}

// NewSessionSharingService 创建会话分享服务
func NewSessionSharingService(teamService *TeamService, db *sql.DB) (*SessionSharingService, error) {
	return NewSessionSharingServiceWithConfig(teamService, db, nil)
}

// NewSessionSharingServiceWithConfig 使用指定配置创建会话分享服务
func NewSessionSharingServiceWithConfig(teamService *TeamService, db *sql.DB, config *TeamServiceConfig) (*SessionSharingService, error) {
	if config == nil {
		config = DefaultTeamServiceConfig()
	}

	repo := storage.NewSharedSessionRepository(db)

	// 初始化表结构
	if err := repo.InitTables(); err != nil {
		return nil, fmt.Errorf("failed to init shared session tables: %w", err)
	}

	return &SessionSharingService{
		teamService:       teamService,
		sharedSessionRepo: repo,
		db:                db,
		httpClient:        config.HTTPClient,
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

	// 非 Leader，转发请求到 Leader
	if !team.LeaderOnline {
		return "", fmt.Errorf("leader is offline, cannot share session")
	}

	return s.forwardShareToLeader(ctx, team, req, sharerID, sharerName)
}

// forwardShareToLeader 转发分享请求到 Leader
func (s *SessionSharingService) forwardShareToLeader(ctx context.Context, team *domainTeam.Team, req *domainTeam.ShareSessionRequest, sharerID, sharerName string) (string, error) {
	url := fmt.Sprintf("http://%s/api/v1/team/%s/sessions/share", team.LeaderEndpoint, team.ID)

	// 构造转发请求体（包含身份信息）
	forwardReq := struct {
		SessionID   string          `json:"session_id"`
		Title       string          `json:"title"`
		Messages    json.RawMessage `json:"messages"`
		Description string          `json:"description,omitempty"`
	}{
		SessionID:   req.SessionID,
		Title:       req.Title,
		Messages:    req.Messages,
		Description: req.Description,
	}

	body, err := json.Marshal(forwardReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal forward request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create forward request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to forward share request to leader: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read leader response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("leader rejected share request: %s", string(respBody))
	}

	// 解析 Leader 返回的 share_id
	var result struct {
		Data struct {
			ShareID string `json:"share_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse leader response: %w", err)
	}

	s.logger.Info("session shared via leader forwarding",
		"share_id", result.Data.ShareID,
		"team_id", team.ID,
		"sharer_id", sharerID,
		"title", req.Title,
	)

	return result.Data.ShareID, nil
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

	// 非 Leader，转发请求到 Leader
	if !team.LeaderOnline {
		return "", fmt.Errorf("leader is offline, cannot add comment")
	}

	return s.forwardCommentToLeader(ctx, team, shareID, req, authorID, authorName)
}

// forwardCommentToLeader 转发评论请求到 Leader
func (s *SessionSharingService) forwardCommentToLeader(ctx context.Context, team *domainTeam.Team, shareID string, req *domainTeam.AddCommentRequest, authorID, authorName string) (string, error) {
	url := fmt.Sprintf("http://%s/api/v1/team/%s/sessions/%s/comments", team.LeaderEndpoint, team.ID, shareID)

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal forward request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create forward request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to forward comment request to leader: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read leader response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("leader rejected comment request: %s", string(respBody))
	}

	// 解析 Leader 返回的 comment_id
	var result struct {
		Data struct {
			CommentID string `json:"comment_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse leader response: %w", err)
	}

	s.logger.Info("comment added via leader forwarding",
		"comment_id", result.Data.CommentID,
		"share_id", shareID,
		"team_id", team.ID,
		"author_id", authorID,
	)

	return result.Data.CommentID, nil
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
