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

	"github.com/google/uuid"

	"github.com/cocursor/backend/internal/domain/p2p"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/infrastructure/log"
	infraP2P "github.com/cocursor/backend/internal/infrastructure/p2p"
	infraTeam "github.com/cocursor/backend/internal/infrastructure/team"
)

// TeamService 团队服务
type TeamService struct {
	mu sync.RWMutex

	// 存储
	identityStore      *infraTeam.IdentityStore
	teamStore          *infraTeam.TeamStore
	networkConfigStore *infraTeam.NetworkConfigStore

	// Leader 相关（仅当本机是 Leader 时使用）
	memberStores     map[string]*infraTeam.MemberStore     // teamID -> MemberStore
	skillIndexStores map[string]*infraTeam.SkillIndexStore // teamID -> SkillIndexStore

	// P2P 组件
	networkManager *infraP2P.NetworkManager
	discovery      *infraP2P.MDNSDiscovery
	advertiser     *infraP2P.MDNSAdvertiser
	wsServer       *infraP2P.WebSocketServer
	connManager    *infraP2P.ConnectionManager

	// HTTP 客户端
	httpClient *http.Client

	// 配置
	port    int
	version string

	logger *slog.Logger
}

// NewTeamService 创建团队服务
func NewTeamService(port int, version string) (*TeamService, error) {
	identityStore, err := infraTeam.NewIdentityStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create identity store: %w", err)
	}

	teamStore, err := infraTeam.NewTeamStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create team store: %w", err)
	}

	networkConfigStore, err := infraTeam.NewNetworkConfigStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create network config store: %w", err)
	}

	networkManager := infraP2P.NewNetworkManager()

	service := &TeamService{
		identityStore:      identityStore,
		teamStore:          teamStore,
		networkConfigStore: networkConfigStore,
		memberStores:       make(map[string]*infraTeam.MemberStore),
		skillIndexStores:   make(map[string]*infraTeam.SkillIndexStore),
		networkManager:     networkManager,
		discovery:          infraP2P.NewMDNSDiscovery(),
		advertiser:         infraP2P.NewMDNSAdvertiser(networkManager),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		port:    port,
		version: version,
		logger:  log.NewModuleLogger("team", "service"),
	}

	// 初始化 Leader 相关存储（如果是 Leader）
	if err := service.initLeaderStores(); err != nil {
		return nil, err
	}

	return service, nil
}

// initLeaderStores 初始化 Leader 存储
func (s *TeamService) initLeaderStores() error {
	leaderTeam := s.teamStore.GetLeaderTeam()
	if leaderTeam == nil {
		return nil
	}

	memberStore, err := infraTeam.NewMemberStore(leaderTeam.ID)
	if err != nil {
		return err
	}
	s.memberStores[leaderTeam.ID] = memberStore

	skillIndexStore, err := infraTeam.NewSkillIndexStore(leaderTeam.ID)
	if err != nil {
		return err
	}
	s.skillIndexStores[leaderTeam.ID] = skillIndexStore

	return nil
}

// SetWebSocketServer 设置 WebSocket 服务端
func (s *TeamService) SetWebSocketServer(server *infraP2P.WebSocketServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wsServer = server
}

// SetConnectionManager 设置连接管理器
func (s *TeamService) SetConnectionManager(manager *infraP2P.ConnectionManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connManager = manager
}

// GetIdentity 获取本机身份
func (s *TeamService) GetIdentity() (*domainTeam.Identity, error) {
	return s.identityStore.Get()
}

// CreateIdentity 创建本机身份
func (s *TeamService) CreateIdentity(name string) (*domainTeam.Identity, error) {
	return s.identityStore.Create(name)
}

// UpdateIdentity 更新本机身份
func (s *TeamService) UpdateIdentity(name string) (*domainTeam.Identity, error) {
	return s.identityStore.UpdateName(name)
}

// EnsureIdentity 确保身份存在，不存在则创建，存在则更新名称
func (s *TeamService) EnsureIdentity(name string) (*domainTeam.Identity, error) {
	identity, err := s.identityStore.Get()
	if err == nil {
		// 身份已存在，检查名称是否需要更新
		if identity.Name != name {
			return s.identityStore.UpdateName(name)
		}
		return identity, nil
	}
	if err == domainTeam.ErrIdentityNotFound {
		return s.identityStore.Create(name)
	}
	return nil, err
}

// GetNetworkInterfaces 获取可用网卡列表
func (s *TeamService) GetNetworkInterfaces() ([]domainTeam.NetworkInterface, error) {
	return s.networkManager.GetAvailableInterfaces()
}

// GetNetworkConfig 获取网卡配置
func (s *TeamService) GetNetworkConfig() *domainTeam.NetworkConfig {
	return s.networkConfigStore.Get()
}

// SetNetworkConfig 设置网卡配置
func (s *TeamService) SetNetworkConfig(preferredInterface, preferredIP string) error {
	return s.networkConfigStore.Set(preferredInterface, preferredIP)
}

// CreateTeam 创建团队（成为 Leader）
func (s *TeamService) CreateTeam(name, preferredInterface, preferredIP string) (*domainTeam.Team, error) {
	// 检查是否已有 Leader 团队
	if s.teamStore.HasLeaderTeam() {
		return nil, domainTeam.ErrTeamAlreadyExists
	}

	// 获取身份
	identity, err := s.identityStore.Get()
	if err != nil {
		return nil, err
	}

	// 设置网卡配置
	if preferredIP != "" {
		if err := s.networkConfigStore.Set(preferredInterface, preferredIP); err != nil {
			return nil, err
		}
	}

	// 获取端点
	endpoint, err := s.networkManager.BuildMemberEndpoint(s.networkConfigStore.Get(), s.port)
	if err != nil {
		return nil, err
	}

	// 创建团队
	now := time.Now()
	team := &domainTeam.Team{
		ID:             uuid.New().String(),
		Name:           name,
		LeaderID:       identity.ID,
		LeaderName:     identity.Name,
		LeaderEndpoint: endpoint.GetAddress(),
		MemberCount:    1,
		SkillCount:     0,
		CreatedAt:      now,
		JoinedAt:       now,
		IsLeader:       true,
		LeaderOnline:   true,
	}

	// 保存团队
	if err := s.teamStore.Add(team); err != nil {
		return nil, err
	}

	// 创建成员存储并添加自己
	memberStore, err := infraTeam.NewMemberStore(team.ID)
	if err != nil {
		s.teamStore.Remove(team.ID)
		return nil, err
	}
	if err := memberStore.AddLeader(identity.ID, identity.Name, endpoint.GetAddress()); err != nil {
		s.teamStore.Remove(team.ID)
		return nil, err
	}
	s.mu.Lock()
	s.memberStores[team.ID] = memberStore
	s.mu.Unlock()

	// 创建技能目录存储
	skillIndexStore, err := infraTeam.NewSkillIndexStore(team.ID)
	if err != nil {
		s.teamStore.Remove(team.ID)
		return nil, err
	}
	s.mu.Lock()
	s.skillIndexStores[team.ID] = skillIndexStore
	s.mu.Unlock()

	// 启动 mDNS 广播
	serviceInfo := infraP2P.BuildServiceInfo(team.ID, team.Name, identity.Name, s.port, 1, s.version)
	if err := s.advertiser.Start(serviceInfo); err != nil {
		s.logger.Warn("failed to start mDNS advertiser",
			"error", err,
		)
		// 不中断创建流程
	}

	s.logger.Info("team created",
		"team_id", team.ID,
		"name", team.Name,
		"endpoint", endpoint.GetAddress(),
	)

	return team, nil
}

// DiscoverTeams 发现局域网团队
func (s *TeamService) DiscoverTeams(ctx context.Context, timeout time.Duration) ([]infraP2P.DiscoveredTeamInfo, error) {
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	return s.discovery.DiscoverTeams(ctx, timeout)
}

// JoinTeam 加入团队
func (s *TeamService) JoinTeam(ctx context.Context, endpoint string) (*domainTeam.Team, error) {
	// 获取身份
	identity, err := s.identityStore.Get()
	if err != nil {
		return nil, err
	}

	// 获取本地端点
	localEndpoint, err := s.networkManager.BuildMemberEndpoint(s.networkConfigStore.Get(), s.port)
	if err != nil {
		return nil, err
	}

	// 解析目标端点
	host, port, err := infraP2P.ParseEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	// 选择匹配的本地 IP
	matchingIP, err := s.networkManager.SelectMatchingLocalIP(host)
	if err != nil {
		matchingIP = localEndpoint.PrimaryIP
	}

	// 构建加入请求
	joinReq := domainTeam.JoinRequest{
		MemberID:   identity.ID,
		MemberName: identity.Name,
		Endpoint:   fmt.Sprintf("%s:%d", matchingIP, s.port),
	}

	// 发送加入请求
	joinResp, err := s.sendJoinRequest(ctx, host, port, &joinReq)
	if err != nil {
		return nil, err
	}

	if !joinResp.Success {
		return nil, fmt.Errorf("join failed: %s", joinResp.Error)
	}

	// 保存团队信息
	team := joinResp.Team
	team.JoinedAt = time.Now()
	team.IsLeader = false
	team.LeaderOnline = true

	if err := s.teamStore.Add(team); err != nil {
		// 如果已经是成员，更新信息
		if err == domainTeam.ErrAlreadyTeamMember {
			s.teamStore.Update(team)
		} else {
			return nil, err
		}
	}

	// 保存技能目录
	if joinResp.SkillIndex != nil {
		skillIndexStore, err := infraTeam.NewSkillIndexStore(team.ID)
		if err == nil {
			skillIndexStore.Replace(joinResp.SkillIndex)
			s.mu.Lock()
			s.skillIndexStores[team.ID] = skillIndexStore
			s.mu.Unlock()
		}
	}

	// 建立 WebSocket 连接
	if s.connManager != nil {
		go func() {
			if err := s.connManager.Connect(team.ID, team.LeaderEndpoint); err != nil {
				s.logger.Warn("failed to establish websocket connection",
					"team_id", team.ID,
					"error", err,
				)
			}
		}()
	}

	s.logger.Info("joined team",
		"team_id", team.ID,
		"name", team.Name,
		"leader", team.LeaderName,
	)

	return team, nil
}

// sendJoinRequest 发送加入请求
func (s *TeamService) sendJoinRequest(ctx context.Context, host string, port int, req *domainTeam.JoinRequest) (*domainTeam.JoinResponse, error) {
	// 首先获取团队信息
	infoURL := fmt.Sprintf("http://%s:%d/team/info", host, port)
	infoResp, err := s.httpClient.Get(infoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get team info: %w", err)
	}
	defer infoResp.Body.Close()

	if infoResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get team info: status %d", infoResp.StatusCode)
	}

	var teamInfo struct {
		Team    domainTeam.Team         `json:"team"`
		Members []domainTeam.TeamMember `json:"members"`
	}
	if err := json.NewDecoder(infoResp.Body).Decode(&teamInfo); err != nil {
		return nil, fmt.Errorf("failed to decode team info: %w", err)
	}

	// 发送加入请求
	joinURL := fmt.Sprintf("http://%s:%d/team/%s/join", host, port, teamInfo.Team.ID)
	reqBody, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", joinURL,
		io.NopCloser(bytesReader(reqBody)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	joinHTTPResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send join request: %w", err)
	}
	defer joinHTTPResp.Body.Close()

	var joinResp domainTeam.JoinResponse
	if err := json.NewDecoder(joinHTTPResp.Body).Decode(&joinResp); err != nil {
		return nil, fmt.Errorf("failed to decode join response: %w", err)
	}

	// 补充团队信息
	if joinResp.Success && joinResp.Team == nil {
		joinResp.Team = &teamInfo.Team
	}

	return &joinResp, nil
}

// LeaveTeam 离开团队
func (s *TeamService) LeaveTeam(ctx context.Context, teamID string) error {
	team, err := s.teamStore.Get(teamID)
	if err != nil {
		return err
	}

	// Leader 不能离开，只能解散
	if team.IsLeader {
		return domainTeam.ErrIsTeamLeader
	}

	// 获取身份
	identity, err := s.identityStore.Get()
	if err != nil {
		return err
	}

	// 通知 Leader
	leaveURL := fmt.Sprintf("http://%s/team/%s/leave", team.LeaderEndpoint, teamID)
	reqBody, _ := json.Marshal(map[string]string{
		"member_id": identity.ID,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", leaveURL,
		io.NopCloser(bytesReader(reqBody)))
	req.Header.Set("Content-Type", "application/json")

	// 忽略响应错误（可能 Leader 已离线）
	resp, err := s.httpClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}

	// 断开 WebSocket 连接
	if s.connManager != nil {
		s.connManager.Disconnect(teamID)
	}

	// 删除本地数据
	s.mu.Lock()
	delete(s.skillIndexStores, teamID)
	s.mu.Unlock()

	// 从团队列表移除
	if err := s.teamStore.Remove(teamID); err != nil {
		return err
	}

	s.logger.Info("left team",
		"team_id", teamID,
		"name", team.Name,
	)

	return nil
}

// DissolveTeam 解散团队（仅 Leader）
func (s *TeamService) DissolveTeam(ctx context.Context, teamID string) error {
	team, err := s.teamStore.Get(teamID)
	if err != nil {
		return err
	}

	if !team.IsLeader {
		return domainTeam.ErrNotTeamLeader
	}

	// 广播解散事件
	if s.wsServer != nil {
		event, _ := p2p.NewEvent(p2p.EventTeamDissolved, teamID, p2p.TeamDissolvedPayload{
			TeamID:      teamID,
			TeamName:    team.Name,
			DissolvedBy: team.LeaderID,
			DissolvedAt: time.Now(),
		})
		s.wsServer.Broadcast(event)
		s.wsServer.Close()
	}

	// 停止 mDNS 广播
	s.advertiser.Stop()

	// 清理存储
	s.mu.Lock()
	delete(s.memberStores, teamID)
	delete(s.skillIndexStores, teamID)
	s.mu.Unlock()

	// 移除团队
	if err := s.teamStore.Remove(teamID); err != nil {
		return err
	}

	s.logger.Info("team dissolved",
		"team_id", teamID,
		"name", team.Name,
	)

	return nil
}

// GetTeamList 获取已加入团队列表
func (s *TeamService) GetTeamList() []*domainTeam.Team {
	return s.teamStore.List()
}

// GetTeam 获取指定团队
func (s *TeamService) GetTeam(teamID string) (*domainTeam.Team, error) {
	return s.teamStore.Get(teamID)
}

// GetTeamMembers 获取团队成员列表
func (s *TeamService) GetTeamMembers(teamID string) ([]*domainTeam.TeamMember, error) {
	// 如果是 Leader，从本地存储获取
	s.mu.RLock()
	memberStore, isLeader := s.memberStores[teamID]
	s.mu.RUnlock()

	if isLeader {
		return memberStore.List(), nil
	}

	// 否则从 Leader 获取
	team, err := s.teamStore.Get(teamID)
	if err != nil {
		return nil, err
	}

	membersURL := fmt.Sprintf("http://%s/team/%s/members", team.LeaderEndpoint, teamID)
	resp, err := s.httpClient.Get(membersURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	defer resp.Body.Close()

	var members []*domainTeam.TeamMember
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, err
	}

	return members, nil
}

// HandleJoinRequest 处理加入请求（Leader 调用）
func (s *TeamService) HandleJoinRequest(teamID string, req *domainTeam.JoinRequest) (*domainTeam.JoinResponse, error) {
	s.mu.RLock()
	memberStore, isLeader := s.memberStores[teamID]
	skillIndexStore := s.skillIndexStores[teamID]
	s.mu.RUnlock()

	if !isLeader {
		return &domainTeam.JoinResponse{
			Success: false,
			Error:   "not the leader of this team",
		}, nil
	}

	// 检查是否已是成员
	if memberStore.Exists(req.MemberID) {
		// 更新端点
		memberStore.UpdateEndpoint(req.MemberID, req.Endpoint)
		memberStore.SetOnline(req.MemberID, true)
	} else {
		// 添加新成员
		member := &domainTeam.TeamMember{
			ID:       req.MemberID,
			Name:     req.MemberName,
			Endpoint: req.Endpoint,
			IsLeader: false,
			IsOnline: true,
			JoinedAt: time.Now(),
		}
		if err := memberStore.Add(member); err != nil {
			return &domainTeam.JoinResponse{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		// 广播成员加入事件
		if s.wsServer != nil {
			event, _ := p2p.NewEvent(p2p.EventMemberJoined, teamID, p2p.MemberJoinedPayload{
				MemberID:   req.MemberID,
				MemberName: req.MemberName,
				Endpoint:   req.Endpoint,
				JoinedAt:   time.Now(),
			})
			s.wsServer.Broadcast(event)
		}

		// 更新 mDNS 广播的成员数量
		s.updateMDNSMemberCount(teamID)
	}

	// 获取团队信息
	team, _ := s.teamStore.Get(teamID)

	// 获取技能目录
	var skillIndex *domainTeam.TeamSkillIndex
	if skillIndexStore != nil {
		skillIndex = skillIndexStore.Get()
	}

	return &domainTeam.JoinResponse{
		Success:    true,
		Team:       team,
		Members:    s.convertMembersToSlice(memberStore.List()),
		SkillIndex: skillIndex,
	}, nil
}

// HandleLeaveRequest 处理离开请求（Leader 调用）
func (s *TeamService) HandleLeaveRequest(teamID, memberID string) error {
	s.mu.RLock()
	memberStore, isLeader := s.memberStores[teamID]
	s.mu.RUnlock()

	if !isLeader {
		return domainTeam.ErrNotTeamLeader
	}

	member, err := memberStore.Get(memberID)
	if err != nil {
		return nil // 不存在也不报错
	}

	if err := memberStore.Remove(memberID); err != nil {
		return err
	}

	// 广播成员离开事件
	if s.wsServer != nil {
		event, _ := p2p.NewEvent(p2p.EventMemberLeft, teamID, p2p.MemberLeftPayload{
			MemberID:   memberID,
			MemberName: member.Name,
			LeftAt:     time.Now(),
		})
		s.wsServer.Broadcast(event)
	}

	// 更新 mDNS 广播的成员数量
	s.updateMDNSMemberCount(teamID)

	return nil
}

// updateMDNSMemberCount 更新 mDNS 成员数量
func (s *TeamService) updateMDNSMemberCount(teamID string) {
	s.mu.RLock()
	memberStore := s.memberStores[teamID]
	s.mu.RUnlock()

	if memberStore == nil || !s.advertiser.IsRunning() {
		return
	}

	count := memberStore.Count()
	s.advertiser.UpdateTxtRecords(map[string]string{
		"member_count": fmt.Sprintf("%d", count),
	})

	// 同时更新本地团队信息
	if team, err := s.teamStore.Get(teamID); err == nil {
		team.MemberCount = count
		s.teamStore.Update(team)
	}
}

// convertMembersToSlice 转换成员列表
func (s *TeamService) convertMembersToSlice(members []*domainTeam.TeamMember) []domainTeam.TeamMember {
	result := make([]domainTeam.TeamMember, len(members))
	for i, m := range members {
		result[i] = *m
	}
	return result
}

// GetSkillIndex 获取团队技能目录
func (s *TeamService) GetSkillIndex(teamID string) (*domainTeam.TeamSkillIndex, error) {
	s.mu.RLock()
	store, exists := s.skillIndexStores[teamID]
	s.mu.RUnlock()

	if exists {
		return store.Get(), nil
	}

	// 从 Leader 获取
	team, err := s.teamStore.Get(teamID)
	if err != nil {
		return nil, err
	}

	skillsURL := fmt.Sprintf("http://%s/team/%s/skills", team.LeaderEndpoint, teamID)
	resp, err := s.httpClient.Get(skillsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get skills: %w", err)
	}
	defer resp.Body.Close()

	var index domainTeam.TeamSkillIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, err
	}

	return &index, nil
}

// Close 关闭服务
func (s *TeamService) Close() error {
	s.advertiser.Stop()
	if s.wsServer != nil {
		s.wsServer.Close()
	}
	if s.connManager != nil {
		s.connManager.Close()
	}
	s.discovery.Close()
	return nil
}

// bytesReader 辅助函数
func bytesReader(b []byte) *bytesReaderImpl {
	return &bytesReaderImpl{data: b}
}

type bytesReaderImpl struct {
	data []byte
	pos  int
}

func (r *bytesReaderImpl) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
