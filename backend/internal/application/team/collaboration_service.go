package team

import (
	"context"
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
	infraTeam "github.com/cocursor/backend/internal/infrastructure/team"
)

// CollaborationService 团队协作服务
type CollaborationService struct {
	mu sync.RWMutex

	teamService *TeamService

	// 本地日报仓储（用于分享自己的日报）
	dailySummaryRepo storage.DailySummaryRepository

	// 日报索引存储（Leader 维护）
	dailySummaryIndexStores map[string]*infraTeam.DailySummaryIndexStore

	// 日报缓存（成员使用）
	dailySummaryCache map[string]*domainTeam.TeamDailySummary // key: teamID:date:memberID

	// 成员工作状态缓存
	workStatusCache map[string]*domainTeam.MemberWorkStatus // key: teamID:memberID

	httpClient *http.Client
	logger     *slog.Logger
}

// NewCollaborationService 创建协作服务
func NewCollaborationService(teamService *TeamService, dailySummaryRepo storage.DailySummaryRepository) *CollaborationService {
	return &CollaborationService{
		teamService:             teamService,
		dailySummaryRepo:        dailySummaryRepo,
		dailySummaryIndexStores: make(map[string]*infraTeam.DailySummaryIndexStore),
		dailySummaryCache:       make(map[string]*domainTeam.TeamDailySummary),
		workStatusCache:         make(map[string]*domainTeam.MemberWorkStatus),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: log.NewModuleLogger("team", "collaboration"),
	}
}

// ShareCode 分享代码片段
func (s *CollaborationService) ShareCode(ctx context.Context, snippet *domainTeam.CodeSnippet) error {
	// 验证代码片段
	if err := snippet.Validate(); err != nil {
		return fmt.Errorf("invalid code snippet: %w", err)
	}

	// 获取团队信息
	team, err := s.teamService.GetTeam(snippet.TeamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}

	// 创建事件 payload
	payload := &p2p.CodeSharedPayload{
		ID:         snippet.ID,
		SenderID:   snippet.SenderID,
		SenderName: snippet.SenderName,
		FileName:   snippet.FileName,
		FilePath:   snippet.FilePath,
		Language:   snippet.Language,
		StartLine:  snippet.StartLine,
		EndLine:    snippet.EndLine,
		Code:       snippet.Code,
		Message:    snippet.Message,
		CreatedAt:  snippet.CreatedAt,
	}

	// 如果是 Leader，直接广播
	if team.IsLeader {
		return s.teamService.BroadcastEvent(snippet.TeamID, p2p.EventCodeShared, payload)
	}

	// 否则发送到 Leader
	return s.sendToLeader(ctx, team.LeaderEndpoint, snippet.TeamID, "share-code", payload)
}

// UpdateWorkStatus 更新工作状态
func (s *CollaborationService) UpdateWorkStatus(ctx context.Context, teamID string, status *p2p.MemberWorkStatusPayload) error {
	// 获取团队信息
	team, err := s.teamService.GetTeam(teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}

	// 更新本地缓存
	cacheKey := fmt.Sprintf("%s:%s", teamID, status.MemberID)
	s.mu.Lock()
	s.workStatusCache[cacheKey] = &domainTeam.MemberWorkStatus{
		ProjectName:   status.ProjectName,
		CurrentFile:   status.CurrentFile,
		LastActiveAt:  status.LastActiveAt,
		StatusVisible: status.StatusVisible,
	}
	s.mu.Unlock()

	// 如果是 Leader，直接广播
	if team.IsLeader {
		return s.teamService.BroadcastEvent(teamID, p2p.EventMemberStatusChanged, status)
	}

	// 否则发送到 Leader
	return s.sendToLeader(ctx, team.LeaderEndpoint, teamID, "status", status)
}

// GetMemberWorkStatus 获取成员工作状态
func (s *CollaborationService) GetMemberWorkStatus(teamID, memberID string) *domainTeam.MemberWorkStatus {
	cacheKey := fmt.Sprintf("%s:%s", teamID, memberID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workStatusCache[cacheKey]
}

// ShareDailySummary 分享日报
func (s *CollaborationService) ShareDailySummary(ctx context.Context, teamID, memberID, memberName, date string) error {
	// 获取团队信息
	team, err := s.teamService.GetTeam(teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}

	// 获取本地日报
	summary, err := s.getLocalDailySummary(date)
	if err != nil {
		return fmt.Errorf("failed to get local daily summary: %w", err)
	}

	// 创建团队日报条目
	entry := &domainTeam.TeamDailySummary{
		MemberID:      memberID,
		MemberName:    memberName,
		Date:          date,
		Summary:       summary.Summary,
		Language:      summary.Language,
		SharedAt:      time.Now(),
		TotalSessions: summary.TotalSessions,
		ProjectCount:  summary.ProjectCount,
	}

	// 创建事件 payload
	payload := &p2p.DailySummarySharedPayload{
		MemberID:      memberID,
		MemberName:    memberName,
		Date:          date,
		TotalSessions: summary.TotalSessions,
		ProjectCount:  summary.ProjectCount,
		SharedAt:      entry.SharedAt,
	}

	// 如果是 Leader，直接更新索引并广播
	if team.IsLeader {
		// 更新日报索引
		if err := s.updateDailySummaryIndex(teamID, entry); err != nil {
			return fmt.Errorf("failed to update daily summary index: %w", err)
		}
		// 尝试广播，但失败不影响保存结果
		if err := s.teamService.BroadcastEvent(teamID, p2p.EventDailySummaryShared, payload); err != nil {
			s.logger.Warn("failed to broadcast daily summary shared event",
				"teamID", teamID,
				"memberID", memberID,
				"error", err,
			)
		}
		return nil
	}

	// 否则发送到 Leader
	return s.sendToLeader(ctx, team.LeaderEndpoint, teamID, "daily-summaries/share", map[string]interface{}{
		"date":           date,
		"total_sessions": summary.TotalSessions,
		"project_count":  summary.ProjectCount,
	})
}

// GetDailySummaries 获取团队日报列表
func (s *CollaborationService) GetDailySummaries(teamID, date string) ([]domainTeam.TeamDailySummary, error) {
	// 获取团队信息
	team, err := s.teamService.GetTeam(teamID)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}

	// 如果是 Leader，从本地存储获取
	if team.IsLeader {
		store, err := s.getDailySummaryIndexStore(teamID)
		if err != nil {
			return nil, err
		}
		index, err := store.Load()
		if err != nil {
			return nil, err
		}
		return index.GetSummariesByDate(date), nil
	}

	// 否则从缓存获取（可能不是最新的）
	// TODO: 可以考虑从 Leader 同步
	var result []domainTeam.TeamDailySummary
	s.mu.RLock()
	for key, summary := range s.dailySummaryCache {
		if summary.Date == date {
			// 检查是否属于这个团队
			if len(key) > len(teamID) && key[:len(teamID)] == teamID {
				result = append(result, *summary)
			}
		}
	}
	s.mu.RUnlock()

	return result, nil
}

// GetDailySummaryDetail 获取日报详情
func (s *CollaborationService) GetDailySummaryDetail(ctx context.Context, teamID, memberID, date string) (*domainTeam.TeamDailySummary, error) {
	// 先检查缓存
	cacheKey := fmt.Sprintf("%s:%s:%s", teamID, date, memberID)
	s.mu.RLock()
	if cached, ok := s.dailySummaryCache[cacheKey]; ok && cached.Summary != "" {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	// 获取团队信息
	team, err := s.teamService.GetTeam(teamID)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}

	// 获取成员信息
	members, err := s.teamService.GetTeamMembers(teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	var memberEndpoint string
	var memberName string
	for _, m := range members {
		if m.ID == memberID {
			memberEndpoint = m.Endpoint
			memberName = m.Name
			break
		}
	}

	if memberEndpoint == "" {
		return nil, fmt.Errorf("member not found or offline")
	}

	// 如果是自己的日报
	identity, _ := s.teamService.GetIdentity()
	if identity != nil && identity.ID == memberID {
		localSummary, err := s.getLocalDailySummary(date)
		if err != nil {
			return nil, err
		}
		return &domainTeam.TeamDailySummary{
			MemberID:      memberID,
			MemberName:    identity.Name,
			Date:          date,
			Summary:       localSummary.Summary,
			Language:      localSummary.Language,
			TotalSessions: localSummary.TotalSessions,
			ProjectCount:  localSummary.ProjectCount,
		}, nil
	}

	// 从成员处 P2P 获取日报
	summary, err := s.fetchDailySummaryFromMember(ctx, memberEndpoint, date)
	if err != nil {
		// 尝试返回缓存（不包含完整内容）
		s.mu.RLock()
		if cached, ok := s.dailySummaryCache[cacheKey]; ok {
			s.mu.RUnlock()
			return cached, nil
		}
		s.mu.RUnlock()
		return nil, fmt.Errorf("failed to fetch daily summary: %w (member may be offline)", err)
	}

	summary.MemberID = memberID
	summary.MemberName = memberName

	// 缓存结果
	s.mu.Lock()
	s.dailySummaryCache[cacheKey] = summary
	s.mu.Unlock()

	// 如果是 Leader，也更新索引
	if team.IsLeader {
		_ = s.updateDailySummaryIndex(teamID, summary)
	}

	return summary, nil
}

// HandleCodeSharedEvent 处理代码分享事件（Leader 调用）
func (s *CollaborationService) HandleCodeSharedEvent(teamID string, payload *p2p.CodeSharedPayload) error {
	// Leader 收到成员的代码分享请求后，广播给所有成员
	return s.teamService.BroadcastEvent(teamID, p2p.EventCodeShared, payload)
}

// HandleWorkStatusEvent 处理工作状态事件（Leader 调用）
func (s *CollaborationService) HandleWorkStatusEvent(teamID string, payload *p2p.MemberWorkStatusPayload) error {
	// 更新本地缓存
	cacheKey := fmt.Sprintf("%s:%s", teamID, payload.MemberID)
	s.mu.Lock()
	s.workStatusCache[cacheKey] = &domainTeam.MemberWorkStatus{
		ProjectName:   payload.ProjectName,
		CurrentFile:   payload.CurrentFile,
		LastActiveAt:  payload.LastActiveAt,
		StatusVisible: payload.StatusVisible,
	}
	s.mu.Unlock()

	// 广播给所有成员
	return s.teamService.BroadcastEvent(teamID, p2p.EventMemberStatusChanged, payload)
}

// HandleDailySummarySharedEvent 处理日报分享事件（Leader 调用）
func (s *CollaborationService) HandleDailySummarySharedEvent(ctx context.Context, teamID string, payload *p2p.DailySummarySharedPayload, memberEndpoint string) error {
	// 更新日报索引
	entry := &domainTeam.TeamDailySummary{
		MemberID:      payload.MemberID,
		MemberName:    payload.MemberName,
		Date:          payload.Date,
		TotalSessions: payload.TotalSessions,
		ProjectCount:  payload.ProjectCount,
		SharedAt:      payload.SharedAt,
	}

	if err := s.updateDailySummaryIndex(teamID, entry); err != nil {
		s.logger.Error("failed to update daily summary index",
			"teamID", teamID,
			"memberID", payload.MemberID,
			"error", err,
		)
	}

	// 广播给所有成员
	return s.teamService.BroadcastEvent(teamID, p2p.EventDailySummaryShared, payload)
}

// 私有方法

// sendToLeader 发送请求到 Leader
func (s *CollaborationService) sendToLeader(ctx context.Context, leaderEndpoint, teamID, action string, payload interface{}) error {
	url := fmt.Sprintf("http://%s/api/v1/team/%s/%s", leaderEndpoint, teamID, action)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("leader returned error: %s - %s", resp.Status, string(body))
	}

	s.logger.Debug("sent request to leader",
		"action", action,
		"payloadSize", len(data),
	)

	return nil
}

// getLocalDailySummary 获取本地日报
func (s *CollaborationService) getLocalDailySummary(date string) (*localDailySummary, error) {
	if s.dailySummaryRepo == nil {
		return nil, fmt.Errorf("daily summary repository not configured")
	}

	// 从本地仓储获取日报
	summary, err := s.dailySummaryRepo.FindByDate(date)
	if err != nil {
		return nil, fmt.Errorf("failed to find daily summary for date %s: %w", date, err)
	}

	if summary == nil {
		return nil, fmt.Errorf("no daily summary found for date %s, please generate it first", date)
	}

	// 计算项目数量
	projectCount := len(summary.Projects)

	return &localDailySummary{
		Summary:       summary.Summary,
		Language:      summary.Language,
		TotalSessions: summary.TotalSessions,
		ProjectCount:  projectCount,
	}, nil
}

type localDailySummary struct {
	Summary       string
	Language      string
	TotalSessions int
	ProjectCount  int
}

// fetchDailySummaryFromMember 从成员处获取日报
func (s *CollaborationService) fetchDailySummaryFromMember(ctx context.Context, memberEndpoint, date string) (*domainTeam.TeamDailySummary, error) {
	url := fmt.Sprintf("http://%s/p2p/daily-summary?date=%s", memberEndpoint, date)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("member returned status %d", resp.StatusCode)
	}

	var summary domainTeam.TeamDailySummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &summary, nil
}

// getDailySummaryIndexStore 获取日报索引存储
func (s *CollaborationService) getDailySummaryIndexStore(teamID string) (*infraTeam.DailySummaryIndexStore, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if store, ok := s.dailySummaryIndexStores[teamID]; ok {
		return store, nil
	}

	store, err := infraTeam.NewDailySummaryIndexStore(teamID)
	if err != nil {
		return nil, err
	}

	s.dailySummaryIndexStores[teamID] = store
	return store, nil
}

// updateDailySummaryIndex 更新日报索引
func (s *CollaborationService) updateDailySummaryIndex(teamID string, entry *domainTeam.TeamDailySummary) error {
	store, err := s.getDailySummaryIndexStore(teamID)
	if err != nil {
		return err
	}

	index, err := store.Load()
	if err != nil {
		// 如果索引不存在，创建新的
		index = &domainTeam.TeamDailySummaryIndex{
			TeamID:    teamID,
			UpdatedAt: time.Now(),
			Summaries: []domainTeam.TeamDailySummary{},
		}
	}

	index.AddOrUpdateSummary(*entry)
	return store.Save(index)
}
