package team

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/cocursor/backend/internal/domain/p2p"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/infrastructure/log"
	infraTeam "github.com/cocursor/backend/internal/infrastructure/team"
)

// SyncService 同步服务
// 负责从 Leader 同步技能目录和处理 WebSocket 事件
type SyncService struct {
	mu sync.RWMutex

	// 通过接口依赖服务层
	teamService TeamServiceInterface

	// 技能目录存储（本地缓存）
	skillIndexStores map[string]*infraTeam.SkillIndexStore

	// HTTP 客户端（通过配置共享）
	httpClient *http.Client

	// 事件回调
	onSkillUpdate         func(teamID string, skillEntry *domainTeam.TeamSkillEntry)
	onSkillDelete         func(teamID, pluginID string)
	onMemberChange        func(teamID string, memberID string, online bool)
	onTeamDissolved       func(teamID string)
	onProjectConfigUpdate func(teamID string, config *domainTeam.TeamProjectConfig)

	logger *slog.Logger
}

// NewSyncService 创建同步服务
// 参数:
//   - teamService: 团队服务接口（用于获取团队信息）
//   - config: 共享配置（可选，为 nil 时使用默认配置）
func NewSyncService(
	teamService TeamServiceInterface,
	config *TeamServiceConfig,
) *SyncService {
	if config == nil {
		config = DefaultTeamServiceConfig()
	}

	return &SyncService{
		teamService:      teamService,
		skillIndexStores: make(map[string]*infraTeam.SkillIndexStore),
		httpClient:       config.HTTPClient,
		logger:           log.NewModuleLogger("team", "sync"),
	}
}

// NewSyncServiceLegacy 创建同步服务（兼容旧接口，将在后续版本移除）
// Deprecated: 使用 NewSyncService 替代
func NewSyncServiceLegacy(
	teamService TeamServiceInterface,
	skillIndexStores map[string]*infraTeam.SkillIndexStore,
) *SyncService {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &SyncService{
		teamService:      teamService,
		skillIndexStores: skillIndexStores,
		httpClient:       httpClient,
		logger:           log.NewModuleLogger("team", "sync"),
	}
}

// SetSkillIndexStores 设置技能目录存储（支持动态添加）
func (s *SyncService) SetSkillIndexStores(stores map[string]*infraTeam.SkillIndexStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.skillIndexStores = stores
}

// SetEventCallbacks 设置事件回调
func (s *SyncService) SetEventCallbacks(
	onSkillUpdate func(teamID string, skillEntry *domainTeam.TeamSkillEntry),
	onSkillDelete func(teamID, pluginID string),
	onMemberChange func(teamID string, memberID string, online bool),
	onTeamDissolved func(teamID string),
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onSkillUpdate = onSkillUpdate
	s.onSkillDelete = onSkillDelete
	s.onMemberChange = onMemberChange
	s.onTeamDissolved = onTeamDissolved
}

// SetOnProjectConfigUpdate 设置项目配置更新回调
func (s *SyncService) SetOnProjectConfigUpdate(callback func(teamID string, config *domainTeam.TeamProjectConfig)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onProjectConfigUpdate = callback
}

// SyncSkillIndex 从 Leader 同步技能目录
func (s *SyncService) SyncSkillIndex(ctx context.Context, teamID string) error {
	// 通过接口获取团队信息
	team, err := s.teamService.GetTeam(teamID)
	if err != nil {
		return err
	}

	// Leader 不需要同步
	if team.IsLeader {
		return nil
	}

	// 从 Leader 获取技能目录
	skillsURL := fmt.Sprintf("http://%s/team/%s/skills", team.LeaderEndpoint, teamID)

	req, err := http.NewRequestWithContext(ctx, "GET", skillsURL, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		// 通过接口更新状态
		s.teamService.UpdateLeaderOnline(teamID, false)
		return fmt.Errorf("leader offline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to sync skills: status %d", resp.StatusCode)
	}

	var index domainTeam.TeamSkillIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return fmt.Errorf("failed to decode skills: %w", err)
	}

	// 保存到本地
	s.mu.RLock()
	store := s.skillIndexStores[teamID]
	s.mu.RUnlock()

	if store == nil {
		store, err = infraTeam.NewSkillIndexStore(teamID)
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.skillIndexStores[teamID] = store
		s.mu.Unlock()
	}

	if err := store.Replace(&index); err != nil {
		return err
	}

	// 通过接口更新同步时间和在线状态
	s.teamService.UpdateLastSync(teamID)
	s.teamService.UpdateLeaderOnline(teamID, true)

	s.logger.Info("skill index synced",
		"team_id", teamID,
		"skill_count", len(index.Skills),
	)

	return nil
}

// HandleWebSocketEvent 处理 WebSocket 事件
func (s *SyncService) HandleWebSocketEvent(event *p2p.Event) error {
	switch event.Type {
	case p2p.EventSkillPublished:
		return s.handleSkillPublished(event)
	case p2p.EventSkillUpdated:
		return s.handleSkillUpdated(event)
	case p2p.EventSkillDeleted:
		return s.handleSkillDeleted(event)
	case p2p.EventMemberJoined:
		return s.handleMemberJoined(event)
	case p2p.EventMemberLeft:
		return s.handleMemberLeft(event)
	case p2p.EventMemberOnline:
		return s.handleMemberOnline(event)
	case p2p.EventMemberOffline:
		return s.handleMemberOffline(event)
	case p2p.EventTeamDissolved:
		return s.handleTeamDissolved(event)
	case p2p.EventProjectConfigUpdated:
		return s.handleProjectConfigUpdated(event)
	default:
		s.logger.Debug("ignoring unknown event type",
			"type", event.Type,
		)
		return nil
	}
}

// handleSkillPublished 处理技能发布事件
func (s *SyncService) handleSkillPublished(event *p2p.Event) error {
	var payload p2p.SkillPublishedPayload
	if err := event.ParsePayload(&payload); err != nil {
		return err
	}

	// 更新本地技能目录
	entry := domainTeam.TeamSkillEntry{
		PluginID:       payload.PluginID,
		Name:           payload.Name,
		Description:    payload.Description,
		Version:        payload.Version,
		AuthorID:       payload.AuthorID,
		AuthorName:     payload.AuthorName,
		AuthorEndpoint: payload.AuthorEndpoint,
		FileCount:      payload.FileCount,
		TotalSize:      payload.TotalSize,
		Checksum:       payload.Checksum,
		PublishedAt:    payload.PublishedAt,
	}

	s.mu.RLock()
	store := s.skillIndexStores[event.TeamID]
	s.mu.RUnlock()

	if store != nil {
		if err := store.AddOrUpdate(entry); err != nil {
			return err
		}
	}

	// 触发回调
	s.mu.RLock()
	callback := s.onSkillUpdate
	s.mu.RUnlock()

	if callback != nil {
		callback(event.TeamID, &entry)
	}

	s.logger.Info("skill published event received",
		"team_id", event.TeamID,
		"plugin_id", payload.PluginID,
		"name", payload.Name,
	)

	return nil
}

// handleSkillUpdated 处理技能更新事件
func (s *SyncService) handleSkillUpdated(event *p2p.Event) error {
	var payload p2p.SkillUpdatedPayload
	if err := event.ParsePayload(&payload); err != nil {
		return err
	}

	s.mu.RLock()
	store := s.skillIndexStores[event.TeamID]
	s.mu.RUnlock()

	if store != nil {
		entry := store.GetSkill(payload.PluginID)
		if entry != nil {
			entry.Version = payload.Version
			entry.Checksum = payload.Checksum
			store.AddOrUpdate(*entry)

			// 触发回调
			s.mu.RLock()
			callback := s.onSkillUpdate
			s.mu.RUnlock()

			if callback != nil {
				callback(event.TeamID, entry)
			}
		}
	}

	s.logger.Info("skill updated event received",
		"team_id", event.TeamID,
		"plugin_id", payload.PluginID,
		"version", payload.Version,
	)

	return nil
}

// handleSkillDeleted 处理技能删除事件
func (s *SyncService) handleSkillDeleted(event *p2p.Event) error {
	var payload p2p.SkillDeletedPayload
	if err := event.ParsePayload(&payload); err != nil {
		return err
	}

	s.mu.RLock()
	store := s.skillIndexStores[event.TeamID]
	s.mu.RUnlock()

	if store != nil {
		store.Remove(payload.PluginID)
	}

	// 触发回调
	s.mu.RLock()
	callback := s.onSkillDelete
	s.mu.RUnlock()

	if callback != nil {
		callback(event.TeamID, payload.PluginID)
	}

	s.logger.Info("skill deleted event received",
		"team_id", event.TeamID,
		"plugin_id", payload.PluginID,
	)

	return nil
}

// handleMemberJoined 处理成员加入事件
func (s *SyncService) handleMemberJoined(event *p2p.Event) error {
	var payload p2p.MemberJoinedPayload
	if err := event.ParsePayload(&payload); err != nil {
		return err
	}

	s.logger.Info("member joined event received",
		"team_id", event.TeamID,
		"member_id", payload.MemberID,
		"member_name", payload.MemberName,
	)

	// 触发回调，让上层处理成员数量更新
	// 注意：成员数量更新由回调处理方负责
	s.mu.RLock()
	callback := s.onMemberChange
	s.mu.RUnlock()

	if callback != nil {
		callback(event.TeamID, payload.MemberID, true)
	}

	return nil
}

// handleMemberLeft 处理成员离开事件
func (s *SyncService) handleMemberLeft(event *p2p.Event) error {
	var payload p2p.MemberLeftPayload
	if err := event.ParsePayload(&payload); err != nil {
		return err
	}

	s.logger.Info("member left event received",
		"team_id", event.TeamID,
		"member_id", payload.MemberID,
		"member_name", payload.MemberName,
	)

	// 触发回调，让上层处理成员数量更新
	// 注意：成员数量更新由回调处理方负责
	s.mu.RLock()
	callback := s.onMemberChange
	s.mu.RUnlock()

	if callback != nil {
		callback(event.TeamID, payload.MemberID, false)
	}

	return nil
}

// handleMemberOnline 处理成员上线事件
func (s *SyncService) handleMemberOnline(event *p2p.Event) error {
	var payload p2p.MemberStatusPayload
	if err := event.ParsePayload(&payload); err != nil {
		return err
	}

	s.logger.Debug("member online event received",
		"team_id", event.TeamID,
		"member_id", payload.MemberID,
	)

	return nil
}

// handleMemberOffline 处理成员离线事件
func (s *SyncService) handleMemberOffline(event *p2p.Event) error {
	var payload p2p.MemberStatusPayload
	if err := event.ParsePayload(&payload); err != nil {
		return err
	}

	s.logger.Debug("member offline event received",
		"team_id", event.TeamID,
		"member_id", payload.MemberID,
	)

	return nil
}

// handleTeamDissolved 处理团队解散事件
func (s *SyncService) handleTeamDissolved(event *p2p.Event) error {
	var payload p2p.TeamDissolvedPayload
	if err := event.ParsePayload(&payload); err != nil {
		return err
	}

	s.logger.Info("team dissolved event received",
		"team_id", event.TeamID,
		"team_name", payload.TeamName,
	)

	// 移除本地技能目录缓存
	s.mu.Lock()
	delete(s.skillIndexStores, event.TeamID)
	s.mu.Unlock()

	// 触发回调，让上层处理团队移除
	// 注意：团队从存储中移除由回调处理方负责
	s.mu.RLock()
	callback := s.onTeamDissolved
	s.mu.RUnlock()

	if callback != nil {
		callback(event.TeamID)
	}

	return nil
}

// handleProjectConfigUpdated 处理项目配置更新事件
func (s *SyncService) handleProjectConfigUpdated(event *p2p.Event) error {
	var payload p2p.ProjectConfigPayload
	if err := event.ParsePayload(&payload); err != nil {
		return err
	}

	s.logger.Info("project config updated event received",
		"team_id", event.TeamID,
		"project_count", len(payload.Projects),
	)

	// 转换为领域模型
	config := &domainTeam.TeamProjectConfig{
		TeamID:    event.TeamID,
		Projects:  make([]domainTeam.ProjectMatcher, len(payload.Projects)),
		UpdatedAt: payload.UpdatedAt,
	}
	for i, p := range payload.Projects {
		config.Projects[i] = domainTeam.ProjectMatcher{
			ID:      p.ID,
			Name:    p.Name,
			RepoURL: p.RepoURL,
		}
	}

	// 触发回调
	s.mu.RLock()
	callback := s.onProjectConfigUpdate
	s.mu.RUnlock()

	if callback != nil {
		callback(event.TeamID, config)
	}

	return nil
}

// StartPeriodicSync 启动周期性同步
func (s *SyncService) StartPeriodicSync(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.syncAllTeams(ctx)
			}
		}
	}()
}

// syncAllTeams 同步所有团队
func (s *SyncService) syncAllTeams(ctx context.Context) {
	// 通过接口获取团队列表
	teams := s.teamService.GetTeamList()
	for _, team := range teams {
		// 跳过 Leader 团队
		if team.IsLeader {
			continue
		}

		if err := s.SyncSkillIndex(ctx, team.ID); err != nil {
			s.logger.Warn("failed to sync team",
				"team_id", team.ID,
				"error", err,
			)
		}
	}
}

// EventListener 实现 p2p.EventListener 接口
type EventListener struct {
	syncService *SyncService
	teamService TeamServiceInterface
	logger      *slog.Logger
}

// NewEventListener 创建事件监听器
func NewEventListener(syncService *SyncService, teamService TeamServiceInterface) *EventListener {
	return &EventListener{
		syncService: syncService,
		teamService: teamService,
		logger:      log.NewModuleLogger("team", "event_listener"),
	}
}

// OnEvent 事件回调
func (l *EventListener) OnEvent(event *p2p.Event) {
	if err := l.syncService.HandleWebSocketEvent(event); err != nil {
		l.logger.Warn("failed to handle event",
			"type", event.Type,
			"error", err,
		)
	}
}

// OnConnect 连接成功回调
func (l *EventListener) OnConnect(teamID string) {
	l.logger.Info("connected to team leader",
		"team_id", teamID,
	)
	// 通过接口更新状态
	l.teamService.UpdateLeaderOnline(teamID, true)

	// 连接成功后立即同步
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		l.syncService.SyncSkillIndex(ctx, teamID)
	}()
}

// OnDisconnect 断开连接回调
func (l *EventListener) OnDisconnect(teamID string, err error) {
	l.logger.Info("disconnected from team leader",
		"team_id", teamID,
		"error", err,
	)
	// 通过接口更新状态
	l.teamService.UpdateLeaderOnline(teamID, false)
}
