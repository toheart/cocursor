package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/infrastructure/git"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/cocursor/backend/internal/infrastructure/storage"
)

// WeeklyStatsHandler 周统计 P2P 处理器
// 提供成员端的统计数据接口，供 Leader 拉取
type WeeklyStatsHandler struct {
	gitCollector     *git.StatsCollector
	sessionRepo      storage.WorkspaceSessionRepository
	dailySummaryRepo storage.DailySummaryRepository
	projectManager   ProjectManagerInterface
	logger           *slog.Logger
}

// ProjectManagerInterface 项目管理器接口
type ProjectManagerInterface interface {
	// FindProjectByRemoteURL 根据远程 URL 查找项目
	FindProjectByRemoteURL(remoteURL string) (*domainCursor.ProjectInfo, error)
	// ListAllProjects 列出所有项目
	ListAllProjects() []*domainCursor.ProjectInfo
}

// NewWeeklyStatsHandler 创建周统计处理器
func NewWeeklyStatsHandler(
	gitCollector *git.StatsCollector,
	sessionRepo storage.WorkspaceSessionRepository,
	dailySummaryRepo storage.DailySummaryRepository,
	projectManager ProjectManagerInterface,
) *WeeklyStatsHandler {
	return &WeeklyStatsHandler{
		gitCollector:     gitCollector,
		sessionRepo:      sessionRepo,
		dailySummaryRepo: dailySummaryRepo,
		projectManager:   projectManager,
		logger:           log.NewModuleLogger("p2p", "weekly_stats"),
	}
}

// GetWeeklyStats 获取周统计数据
// 路由: GET /p2p/weekly-stats?week_start=2026-01-20&repo_urls=url1,url2
func (h *WeeklyStatsHandler) GetWeeklyStats(c *gin.Context) {
	weekStart := c.Query("week_start")
	if weekStart == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "week_start is required"})
		return
	}

	// 验证日期格式
	startDate, err := time.Parse("2006-01-02", weekStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid week_start format, expected YYYY-MM-DD"})
		return
	}

	// 解析 repo_urls
	repoURLsStr := c.Query("repo_urls")
	var repoURLs []string
	if repoURLsStr != "" {
		repoURLs = strings.Split(repoURLsStr, ",")
	}

	// 收集统计数据
	stats, err := h.collectWeeklyStats(startDate, repoURLs)
	if err != nil {
		h.logger.Error("failed to collect weekly stats",
			"week_start", weekStart,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to collect stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetDailyDetail 获取日详情数据
// 路由: GET /p2p/daily-detail?date=2026-01-21&repo_urls=url1,url2
func (h *WeeklyStatsHandler) GetDailyDetail(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required"})
		return
	}

	// 验证日期格式
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, expected YYYY-MM-DD"})
		return
	}

	// 解析 repo_urls
	repoURLsStr := c.Query("repo_urls")
	var repoURLs []string
	if repoURLsStr != "" {
		repoURLs = strings.Split(repoURLsStr, ",")
	}

	// 收集日详情
	detail, err := h.collectDailyDetail(date, repoURLs)
	if err != nil {
		h.logger.Error("failed to collect daily detail",
			"date", date,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to collect detail"})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// collectWeeklyStats 收集一周的统计数据
func (h *WeeklyStatsHandler) collectWeeklyStats(startDate time.Time, repoURLs []string) (*domainTeam.MemberWeeklyStats, error) {
	stats := &domainTeam.MemberWeeklyStats{
		WeekStart:  startDate.Format("2006-01-02"),
		DailyStats: make([]domainTeam.MemberDailyStats, 7),
		UpdatedAt:  time.Now(),
	}

	// 获取用户邮箱
	userEmail, err := h.gitCollector.GetUserEmail()
	if err != nil {
		h.logger.Warn("failed to get user email", "error", err)
		userEmail = ""
	}

	// 收集 7 天的数据
	for i := 0; i < 7; i++ {
		currentDate := startDate.AddDate(0, 0, i)
		dateStr := currentDate.Format("2006-01-02")

		dailyStats := domainTeam.MemberDailyStats{
			Date:      dateStr,
			WorkItems: []domainTeam.WorkItemSummary{},
		}

		// 收集 Git 统计
		if userEmail != "" && len(repoURLs) > 0 {
			gitStats := h.collectGitStats(dateStr, repoURLs, userEmail)
			if gitStats != nil && gitStats.TotalCommits > 0 {
				dailyStats.GitStats = gitStats
			}
		}

		// 收集 Cursor 统计
		cursorStats := h.collectCursorStats(dateStr, repoURLs)
		if cursorStats != nil && cursorStats.SessionCount > 0 {
			dailyStats.CursorStats = cursorStats
		}

		// 收集工作条目
		workItems, hasReport := h.collectWorkItems(dateStr, repoURLs)
		dailyStats.WorkItems = workItems
		dailyStats.HasReport = hasReport

		stats.DailyStats[i] = dailyStats
	}

	return stats, nil
}

// collectGitStats 收集 Git 统计
func (h *WeeklyStatsHandler) collectGitStats(date string, repoURLs []string, userEmail string) *domainTeam.GitDailyStats {
	stats := &domainTeam.GitDailyStats{
		TotalCommits: 0,
		TotalAdded:   0,
		TotalRemoved: 0,
		Projects:     []domainTeam.ProjectGitStats{},
	}

	for _, repoURL := range repoURLs {
		// 在本地查找仓库
		repoPath, err := h.gitCollector.FindRepoByRemoteURL(repoURL)
		if err != nil {
			h.logger.Debug("repo not found locally",
				"repo_url", repoURL,
				"error", err,
			)
			continue
		}

		// 尝试获取仓库级别的邮箱
		repoEmail := userEmail
		if email, err := h.gitCollector.GetRepoUserEmail(repoPath); err == nil && email != "" {
			repoEmail = email
		}

		// 收集统计
		projectStats, err := h.gitCollector.CollectDailyStats(repoPath, date, repoEmail)
		if err != nil {
			h.logger.Warn("failed to collect git stats",
				"repo_path", repoPath,
				"date", date,
				"error", err,
			)
			continue
		}

		if projectStats.Commits > 0 {
			stats.Projects = append(stats.Projects, *projectStats)
			stats.TotalCommits += projectStats.Commits
			stats.TotalAdded += projectStats.LinesAdded
			stats.TotalRemoved += projectStats.LinesRemoved
		}
	}

	return stats
}

// collectCursorStats 收集 Cursor 统计
func (h *WeeklyStatsHandler) collectCursorStats(date string, repoURLs []string) *domainTeam.CursorDailyStats {
	if h.sessionRepo == nil || h.projectManager == nil {
		return nil
	}

	stats := &domainTeam.CursorDailyStats{}

	// 查找匹配的工作区
	var workspaceIDs []string
	for _, repoURL := range repoURLs {
		project, err := h.projectManager.FindProjectByRemoteURL(repoURL)
		if err != nil || project == nil {
			continue
		}
		for _, ws := range project.Workspaces {
			workspaceIDs = append(workspaceIDs, ws.WorkspaceID)
		}
	}

	if len(workspaceIDs) == 0 {
		return nil
	}

	// 查询会话数据
	sessions, err := h.sessionRepo.FindByWorkspacesAndDateRange(workspaceIDs, date, date)
	if err != nil {
		h.logger.Warn("failed to query sessions", "error", err)
		return nil
	}

	for _, session := range sessions {
		stats.SessionCount++
		stats.LinesAdded += session.TotalLinesAdded
		stats.LinesRemoved += session.TotalLinesRemoved
	}

	// 获取 Token 使用量
	tokenUsage, err := h.sessionRepo.GetDailyTokenUsage(workspaceIDs, date, date)
	if err == nil && len(tokenUsage) > 0 {
		for _, usage := range tokenUsage {
			stats.TokensUsed += usage.TokenCount
		}
	}

	return stats
}

// collectWorkItems 收集工作条目
func (h *WeeklyStatsHandler) collectWorkItems(date string, repoURLs []string) ([]domainTeam.WorkItemSummary, bool) {
	if h.dailySummaryRepo == nil {
		return nil, false
	}

	// 获取日报
	summary, err := h.dailySummaryRepo.FindByDate(date)
	if err != nil || summary == nil {
		return nil, false
	}

	// 创建 repoURL 集合用于匹配
	repoURLSet := make(map[string]bool)
	for _, url := range repoURLs {
		repoURLSet[normalizeURL(url)] = true
	}

	var workItems []domainTeam.WorkItemSummary

	// 遍历项目提取工作条目
	for _, project := range summary.Projects {
		// 检查是否匹配请求的项目
		// 这里简化处理，如果没有指定 repoURLs 则返回所有
		if len(repoURLs) > 0 {
			// 尝试匹配项目（通过项目名或路径）
			matched := false
			if h.projectManager != nil {
				projects := h.projectManager.ListAllProjects()
				for _, p := range projects {
					if p.ProjectName == project.ProjectName {
						if repoURLSet[normalizeURL(p.GitRemoteURL)] {
							matched = true
							break
						}
					}
				}
			}
			if !matched {
				continue
			}
		}

		for _, item := range project.WorkItems {
			workItems = append(workItems, domainTeam.WorkItemSummary{
				Project:     project.ProjectName,
				Category:    item.Category,
				Description: item.Description,
			})
		}
	}

	return workItems, true
}

// collectDailyDetail 收集日详情
func (h *WeeklyStatsHandler) collectDailyDetail(date string, repoURLs []string) (*domainTeam.MemberDailyDetail, error) {
	// 获取用户邮箱
	userEmail, err := h.gitCollector.GetUserEmail()
	if err != nil {
		userEmail = ""
	}

	detail := &domainTeam.MemberDailyDetail{
		Date:      date,
		WorkItems: []domainTeam.WorkItemSummary{},
	}

	// 收集 Git 统计
	if userEmail != "" && len(repoURLs) > 0 {
		detail.GitStats = h.collectGitStats(date, repoURLs, userEmail)
	}

	// 收集 Cursor 统计
	detail.CursorStats = h.collectCursorStats(date, repoURLs)

	// 收集工作条目
	workItems, hasReport := h.collectWorkItems(date, repoURLs)
	detail.WorkItems = workItems
	detail.HasReport = hasReport

	return detail, nil
}

// RegisterRoutes 注册路由
func (h *WeeklyStatsHandler) RegisterRoutes(router *gin.Engine) {
	p2p := router.Group("/p2p")
	{
		p2p.GET("/weekly-stats", h.GetWeeklyStats)
		p2p.GET("/daily-detail", h.GetDailyDetail)
	}
}

// normalizeURL 规范化 URL
func normalizeURL(url string) string {
	normalized := strings.ToLower(url)
	normalized = strings.TrimSuffix(normalized, ".git")
	normalized = strings.TrimPrefix(normalized, "https://")
	normalized = strings.TrimPrefix(normalized, "http://")
	normalized = strings.TrimPrefix(normalized, "ssh://")
	if strings.HasPrefix(normalized, "git@") {
		normalized = strings.TrimPrefix(normalized, "git@")
		normalized = strings.Replace(normalized, ":", "/", 1)
	}
	return normalized
}
