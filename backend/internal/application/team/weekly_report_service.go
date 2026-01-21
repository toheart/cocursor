package team

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cocursor/backend/internal/domain/p2p"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/infrastructure/log"
	infraTeam "github.com/cocursor/backend/internal/infrastructure/team"
)

// WeeklyReportService 周报服务
type WeeklyReportService struct {
	mu sync.RWMutex

	teamService *TeamService

	// 项目配置存储（按团队）
	projectConfigStores map[string]*infraTeam.ProjectConfigStore

	// 周统计缓存存储（按团队）
	weeklyStatsStores map[string]*infraTeam.WeeklyStatsStore

	httpClient *http.Client
	logger     *slog.Logger
}

// NewWeeklyReportService 创建周报服务
func NewWeeklyReportService(teamService *TeamService) *WeeklyReportService {
	return &WeeklyReportService{
		teamService:         teamService,
		projectConfigStores: make(map[string]*infraTeam.ProjectConfigStore),
		weeklyStatsStores:   make(map[string]*infraTeam.WeeklyStatsStore),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: log.NewModuleLogger("team", "weekly_report"),
	}
}

// GetProjectConfig 获取项目配置
// 如果本地没有配置且不是 Leader，则从 Leader 拉取
func (s *WeeklyReportService) GetProjectConfig(teamID string) (*domainTeam.TeamProjectConfig, error) {
	store, err := s.getProjectConfigStore(teamID)
	if err != nil {
		return nil, err
	}
	
	config, err := store.Load()
	if err != nil {
		return nil, err
	}
	
	// 如果本地没有配置且不是 Leader，尝试从 Leader 拉取
	if len(config.Projects) == 0 {
		team, err := s.teamService.GetTeam(teamID)
		if err == nil && !team.IsLeader && team.LeaderOnline {
			// 从 Leader 拉取配置
			fetchedConfig, fetchErr := s.fetchProjectConfigFromLeader(teamID, team.LeaderEndpoint)
			if fetchErr == nil && len(fetchedConfig.Projects) > 0 {
				// 保存到本地
				_ = store.Save(fetchedConfig)
				return fetchedConfig, nil
			}
		}
	}
	
	return config, nil
}

// fetchProjectConfigFromLeader 从 Leader 拉取项目配置
func (s *WeeklyReportService) fetchProjectConfigFromLeader(teamID, leaderEndpoint string) (*domainTeam.TeamProjectConfig, error) {
	url := fmt.Sprintf("http://%s/api/v1/team/%s/project-config", leaderEndpoint, teamID)
	
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("leader returned status %d", resp.StatusCode)
	}
	
	var result struct {
		Code    int                           `json:"code"`
		Data    domainTeam.TeamProjectConfig  `json:"data"`
		Message string                        `json:"message"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	if result.Code != 0 {
		return nil, fmt.Errorf("leader returned error: %s", result.Message)
	}
	
	return &result.Data, nil
}

// UpdateProjectConfig 更新项目配置
func (s *WeeklyReportService) UpdateProjectConfig(ctx context.Context, teamID string, projects []domainTeam.ProjectMatcher) error {
	// 验证是否是 Leader
	team, err := s.teamService.GetTeam(teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}
	if !team.IsLeader {
		return fmt.Errorf("only leader can update project config")
	}

	store, err := s.getProjectConfigStore(teamID)
	if err != nil {
		return err
	}

	// 为没有 ID 的项目生成 ID
	for i := range projects {
		if projects[i].ID == "" {
			projects[i].ID = uuid.New().String()
		}
	}

	config := &domainTeam.TeamProjectConfig{
		TeamID:    teamID,
		Projects:  projects,
		UpdatedAt: time.Now(),
	}

	if err := store.Save(config); err != nil {
		return err
	}

	// 广播配置更新事件
	payload := &p2p.ProjectConfigPayload{
		Projects:  make([]p2p.ProjectMatcherPayload, len(projects)),
		UpdatedAt: config.UpdatedAt,
	}
	for i, p := range projects {
		payload.Projects[i] = p2p.ProjectMatcherPayload{
			ID:      p.ID,
			Name:    p.Name,
			RepoURL: p.RepoURL,
		}
	}

	if err := s.teamService.BroadcastEvent(teamID, p2p.EventProjectConfigUpdated, payload); err != nil {
		s.logger.Warn("failed to broadcast project config update",
			"teamID", teamID,
			"error", err,
		)
	}

	return nil
}

// AddProject 添加项目
func (s *WeeklyReportService) AddProject(ctx context.Context, teamID, name, repoURL string) (*domainTeam.ProjectMatcher, error) {
	// 验证是否是 Leader
	team, err := s.teamService.GetTeam(teamID)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}
	if !team.IsLeader {
		return nil, fmt.Errorf("only leader can add project")
	}

	store, err := s.getProjectConfigStore(teamID)
	if err != nil {
		return nil, err
	}

	project, err := store.AddProject(name, repoURL)
	if err != nil {
		return nil, err
	}

	// 广播配置更新
	config, _ := store.Load()
	if config != nil {
		s.broadcastProjectConfig(teamID, config)
	}

	return project, nil
}

// RemoveProject 移除项目
func (s *WeeklyReportService) RemoveProject(ctx context.Context, teamID, projectID string) error {
	// 验证是否是 Leader
	team, err := s.teamService.GetTeam(teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}
	if !team.IsLeader {
		return fmt.Errorf("only leader can remove project")
	}

	store, err := s.getProjectConfigStore(teamID)
	if err != nil {
		return err
	}

	if err := store.RemoveProject(projectID); err != nil {
		return err
	}

	// 广播配置更新
	config, _ := store.Load()
	if config != nil {
		s.broadcastProjectConfig(teamID, config)
	}

	return nil
}

// GetWeeklyReport 获取周报
func (s *WeeklyReportService) GetWeeklyReport(ctx context.Context, teamID, weekStart string) (*domainTeam.TeamWeeklyView, error) {
	// 验证日期格式并规范化到周一
	startDate, err := s.normalizeToMonday(weekStart)
	if err != nil {
		return nil, err
	}
	weekStart = startDate.Format("2006-01-02")
	weekEnd := startDate.AddDate(0, 0, 6).Format("2006-01-02")

	// 获取团队成员
	members, err := s.teamService.GetTeamMembers(teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	// 获取项目配置
	config, err := s.GetProjectConfig(teamID)
	if err != nil {
		s.logger.Warn("failed to get project config", "teamID", teamID, "error", err)
		config = &domainTeam.TeamProjectConfig{Projects: []domainTeam.ProjectMatcher{}}
	}

	// 获取缓存存储
	statsStore, err := s.getWeeklyStatsStore(teamID)
	if err != nil {
		return nil, err
	}

	// 收集所有成员的统计数据
	memberStats := make(map[string]*domainTeam.MemberWeeklyStats)

	// 获取本地身份，用于判断是否需要跳过自己
	localIdentity, _ := s.teamService.GetIdentity()
	var localMemberID string
	if localIdentity != nil {
		localMemberID = localIdentity.ID
	}

	for _, member := range members {
		// 先检查缓存
		cached, expired, _ := statsStore.Get(member.ID, weekStart)
		if cached != nil && !expired {
			memberStats[member.ID] = cached
			continue
		}

		// 如果是本地成员，使用 localhost 而不是远程 endpoint
		endpoint := member.Endpoint
		if member.ID == localMemberID {
			endpoint = "127.0.0.1:19960"
		}

		// 缓存过期或不存在，尝试从成员拉取
		if member.IsOnline {
			stats, err := s.fetchMemberStats(ctx, endpoint, weekStart, config.GetRepoURLs())
			if err != nil {
				s.logger.Warn("failed to fetch member stats",
					"memberID", member.ID,
					"endpoint", member.Endpoint,
					"error", err,
				)
				// 使用过期缓存
				if cached != nil {
					memberStats[member.ID] = cached
				}
				continue
			}
			stats.MemberID = member.ID
			stats.MemberName = member.Name

			// 更新缓存
			if err := statsStore.Set(stats); err != nil {
				s.logger.Warn("failed to cache member stats", "error", err)
			}
			memberStats[member.ID] = stats
		} else if cached != nil {
			// 成员离线，使用缓存
			memberStats[member.ID] = cached
		}
	}

	// 构建周视图
	view := s.buildWeeklyView(teamID, weekStart, weekEnd, members, memberStats, config)

	return view, nil
}

// GetMemberDailyDetail 获取成员日详情
func (s *WeeklyReportService) GetMemberDailyDetail(ctx context.Context, teamID, memberID, date string) (*domainTeam.MemberDailyDetail, error) {
	// 获取成员信息
	members, err := s.teamService.GetTeamMembers(teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	var member *domainTeam.TeamMember
	for _, m := range members {
		if m.ID == memberID {
			member = m
			break
		}
	}
	if member == nil {
		return nil, fmt.Errorf("member not found: %s", memberID)
	}

	// 获取项目配置
	config, err := s.GetProjectConfig(teamID)
	if err != nil {
		config = &domainTeam.TeamProjectConfig{Projects: []domainTeam.ProjectMatcher{}}
	}

	// 获取本地身份，用于判断是否是自己
	localIdentity, _ := s.teamService.GetIdentity()
	var localMemberID string
	if localIdentity != nil {
		localMemberID = localIdentity.ID
	}

	// 尝试从成员端获取详情
	if member.IsOnline {
		// 如果是本地成员，使用 127.0.0.1 避免 EOF 错误
		endpoint := member.Endpoint
		if member.ID == localMemberID {
			endpoint = "127.0.0.1:19960"
		}
		detail, err := s.fetchMemberDailyDetail(ctx, endpoint, date, config.GetRepoURLs())
		if err != nil {
			s.logger.Warn("failed to fetch member daily detail",
				"memberID", memberID,
				"error", err,
			)
		} else {
			detail.MemberID = member.ID
			detail.MemberName = member.Name
			detail.IsOnline = true
			return detail, nil
		}
	}

	// 成员离线，尝试从缓存构建
	weekStart := s.getWeekStart(date)
	statsStore, err := s.getWeeklyStatsStore(teamID)
	if err != nil {
		return nil, err
	}

	cached, _, _ := statsStore.Get(memberID, weekStart)
	if cached == nil {
		return nil, fmt.Errorf("no cached data for member %s", memberID)
	}

	// 从缓存中找到对应日期的数据
	for _, daily := range cached.DailyStats {
		if daily.Date == date {
			return &domainTeam.MemberDailyDetail{
				MemberID:    member.ID,
				MemberName:  member.Name,
				Date:        date,
				GitStats:    daily.GitStats,
				CursorStats: daily.CursorStats,
				WorkItems:   daily.WorkItems,
				HasReport:   daily.HasReport,
				IsOnline:    false,
				IsCached:    true,
			}, nil
		}
	}

	return nil, fmt.Errorf("no data for date %s", date)
}

// RefreshWeeklyStats 刷新周统计数据
func (s *WeeklyReportService) RefreshWeeklyStats(ctx context.Context, teamID, weekStart string) error {
	// 规范化到周一
	startDate, err := s.normalizeToMonday(weekStart)
	if err != nil {
		return err
	}
	weekStart = startDate.Format("2006-01-02")

	// 获取团队成员
	members, err := s.teamService.GetTeamMembers(teamID)
	if err != nil {
		return fmt.Errorf("failed to get team members: %w", err)
	}

	// 获取项目配置
	config, err := s.GetProjectConfig(teamID)
	if err != nil {
		config = &domainTeam.TeamProjectConfig{Projects: []domainTeam.ProjectMatcher{}}
	}

	// 获取缓存存储
	statsStore, err := s.getWeeklyStatsStore(teamID)
	if err != nil {
		return err
	}

	// 获取本地身份
	localIdentity, _ := s.teamService.GetIdentity()
	var localMemberID string
	if localIdentity != nil {
		localMemberID = localIdentity.ID
	}

	// 并发刷新所有在线成员
	var wg sync.WaitGroup
	for _, member := range members {
		if !member.IsOnline {
			continue
		}

		wg.Add(1)
		go func(m *domainTeam.TeamMember) {
			defer wg.Done()

			// 如果是本地成员，使用 localhost
			endpoint := m.Endpoint
			if m.ID == localMemberID {
				endpoint = "127.0.0.1:19960"
			}

			stats, err := s.fetchMemberStats(ctx, endpoint, weekStart, config.GetRepoURLs())
			if err != nil {
				s.logger.Warn("failed to refresh member stats",
					"memberID", m.ID,
					"error", err,
				)
				return
			}
			stats.MemberID = m.ID
			stats.MemberName = m.Name

			if err := statsStore.Set(stats); err != nil {
				s.logger.Warn("failed to cache member stats", "error", err)
			}
		}(member)
	}

	wg.Wait()
	return nil
}

// 私有方法

func (s *WeeklyReportService) getProjectConfigStore(teamID string) (*infraTeam.ProjectConfigStore, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if store, ok := s.projectConfigStores[teamID]; ok {
		return store, nil
	}

	store, err := infraTeam.NewProjectConfigStore(teamID)
	if err != nil {
		return nil, err
	}

	s.projectConfigStores[teamID] = store
	return store, nil
}

func (s *WeeklyReportService) getWeeklyStatsStore(teamID string) (*infraTeam.WeeklyStatsStore, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if store, ok := s.weeklyStatsStores[teamID]; ok {
		return store, nil
	}

	store, err := infraTeam.NewWeeklyStatsStore(teamID)
	if err != nil {
		return nil, err
	}

	s.weeklyStatsStores[teamID] = store
	return store, nil
}

func (s *WeeklyReportService) broadcastProjectConfig(teamID string, config *domainTeam.TeamProjectConfig) {
	payload := &p2p.ProjectConfigPayload{
		Projects:  make([]p2p.ProjectMatcherPayload, len(config.Projects)),
		UpdatedAt: config.UpdatedAt,
	}
	for i, p := range config.Projects {
		payload.Projects[i] = p2p.ProjectMatcherPayload{
			ID:      p.ID,
			Name:    p.Name,
			RepoURL: p.RepoURL,
		}
	}

	if err := s.teamService.BroadcastEvent(teamID, p2p.EventProjectConfigUpdated, payload); err != nil {
		s.logger.Warn("failed to broadcast project config",
			"teamID", teamID,
			"error", err,
		)
	}
}

// HandleProjectConfigUpdate 处理从 Leader 接收到的项目配置更新
// 此方法供成员端使用，保存 Leader 推送的配置到本地
func (s *WeeklyReportService) HandleProjectConfigUpdate(teamID string, config *domainTeam.TeamProjectConfig) error {
	store, err := s.getProjectConfigStore(teamID)
	if err != nil {
		return err
	}

	if err := store.Save(config); err != nil {
		s.logger.Error("failed to save project config from leader",
			"teamID", teamID,
			"error", err,
		)
		return err
	}

	s.logger.Info("project config updated from leader",
		"teamID", teamID,
		"projectCount", len(config.Projects),
	)

	return nil
}

func (s *WeeklyReportService) fetchMemberStats(ctx context.Context, endpoint, weekStart string, repoURLs []string) (*domainTeam.MemberWeeklyStats, error) {
	url := fmt.Sprintf("http://%s/p2p/weekly-stats?week_start=%s", endpoint, weekStart)
	if len(repoURLs) > 0 {
		url += "&repo_urls=" + strings.Join(repoURLs, ",")
	}

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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("member returned status %d: %s", resp.StatusCode, string(body))
	}

	var stats domainTeam.MemberWeeklyStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &stats, nil
}

func (s *WeeklyReportService) fetchMemberDailyDetail(ctx context.Context, endpoint, date string, repoURLs []string) (*domainTeam.MemberDailyDetail, error) {
	url := fmt.Sprintf("http://%s/p2p/daily-detail?date=%s", endpoint, date)
	if len(repoURLs) > 0 {
		url += "&repo_urls=" + strings.Join(repoURLs, ",")
	}

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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("member returned status %d: %s", resp.StatusCode, string(body))
	}

	var detail domainTeam.MemberDailyDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &detail, nil
}

func (s *WeeklyReportService) normalizeToMonday(dateStr string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: %w", err)
	}

	// 调整到周一
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7 // 周日
	}
	monday := date.AddDate(0, 0, -(weekday - 1))
	return monday, nil
}

func (s *WeeklyReportService) getWeekStart(dateStr string) string {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	monday, _ := s.normalizeToMonday(date.Format("2006-01-02"))
	return monday.Format("2006-01-02")
}

func (s *WeeklyReportService) buildWeeklyView(
	teamID, weekStart, weekEnd string,
	members []*domainTeam.TeamMember,
	memberStats map[string]*domainTeam.MemberWeeklyStats,
	config *domainTeam.TeamProjectConfig,
) *domainTeam.TeamWeeklyView {
	view := &domainTeam.TeamWeeklyView{
		TeamID:         teamID,
		WeekStart:      weekStart,
		WeekEnd:        weekEnd,
		Calendar:       make([]domainTeam.TeamDayColumn, 7),
		ProjectSummary: []domainTeam.ProjectWeekStats{},
		UpdatedAt:      time.Now(),
	}

	startDate, _ := time.Parse("2006-01-02", weekStart)

	// 构建日历
	for i := 0; i < 7; i++ {
		currentDate := startDate.AddDate(0, 0, i)
		dateStr := currentDate.Format("2006-01-02")
		dayOfWeek := i + 1 // 1=周一, 7=周日

		column := domainTeam.TeamDayColumn{
			Date:      dateStr,
			DayOfWeek: dayOfWeek,
			Members:   make([]domainTeam.MemberDayCell, 0, len(members)),
		}

		for _, member := range members {
			cell := domainTeam.MemberDayCell{
				MemberID:   member.ID,
				MemberName: member.Name,
				IsOnline:   member.IsOnline,
			}

			// 填充统计数据
			if stats, ok := memberStats[member.ID]; ok {
				for _, daily := range stats.DailyStats {
					if daily.Date == dateStr {
						if daily.GitStats != nil {
							cell.Commits = daily.GitStats.TotalCommits
							cell.LinesChanged = daily.GitStats.TotalAdded + daily.GitStats.TotalRemoved
						}
						cell.HasReport = daily.HasReport
						break
					}
				}
			}

			cell.CalculateActivityLevel()
			column.Members = append(column.Members, cell)
		}

		view.Calendar[i] = column
	}

	// 构建项目汇总
	projectStats := make(map[string]*domainTeam.ProjectWeekStats)
	contributorStats := make(map[string]map[string]*domainTeam.ContributorStats) // projectURL -> memberID -> stats

	for memberID, stats := range memberStats {
		memberName := ""
		for _, m := range members {
			if m.ID == memberID {
				memberName = m.Name
				break
			}
		}

		for _, daily := range stats.DailyStats {
			if daily.GitStats == nil {
				continue
			}
			for _, projStats := range daily.GitStats.Projects {
				repoURL := projStats.RepoURL
				if repoURL == "" {
					continue
				}

				if _, ok := projectStats[repoURL]; !ok {
					projectStats[repoURL] = &domainTeam.ProjectWeekStats{
						ProjectName:  projStats.ProjectName,
						RepoURL:      repoURL,
						Contributors: []domainTeam.ContributorStats{},
					}
					contributorStats[repoURL] = make(map[string]*domainTeam.ContributorStats)
				}

				ps := projectStats[repoURL]
				ps.TotalCommits += projStats.Commits
				ps.TotalAdded += projStats.LinesAdded
				ps.TotalRemoved += projStats.LinesRemoved

				if _, ok := contributorStats[repoURL][memberID]; !ok {
					contributorStats[repoURL][memberID] = &domainTeam.ContributorStats{
						MemberID:   memberID,
						MemberName: memberName,
					}
				}
				cs := contributorStats[repoURL][memberID]
				cs.Commits += projStats.Commits
				cs.LinesAdded += projStats.LinesAdded
				cs.LinesRemoved += projStats.LinesRemoved
			}
		}
	}

	// 转换贡献者统计
	for repoURL, ps := range projectStats {
		for _, cs := range contributorStats[repoURL] {
			ps.Contributors = append(ps.Contributors, *cs)
		}
		// 按 commits 排序
		sort.Slice(ps.Contributors, func(i, j int) bool {
			return ps.Contributors[i].Commits > ps.Contributors[j].Commits
		})
		view.ProjectSummary = append(view.ProjectSummary, *ps)
	}

	// 按总 commits 排序项目
	sort.Slice(view.ProjectSummary, func(i, j int) bool {
		return view.ProjectSummary[i].TotalCommits > view.ProjectSummary[j].TotalCommits
	})

	return view
}
