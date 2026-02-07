package team

import (
	"time"
)

// =====================
// 团队项目配置
// =====================

// TeamProjectConfig 团队项目配置
type TeamProjectConfig struct {
	TeamID    string           `json:"team_id"`    // 团队 ID
	Projects  []ProjectMatcher `json:"projects"`   // 项目列表
	UpdatedAt time.Time        `json:"updated_at"` // 更新时间
}

// ProjectMatcher 项目匹配规则
type ProjectMatcher struct {
	ID      string `json:"id"`       // 规则 ID (UUID)
	Name    string `json:"name"`     // 显示名称，如 "CoCursor 主项目"
	RepoURL string `json:"repo_url"` // Git Remote URL（不含协议前缀，如 "github.com/org/repo"）
}

// FindProject 查找项目
func (c *TeamProjectConfig) FindProject(repoURL string) *ProjectMatcher {
	for i := range c.Projects {
		if c.Projects[i].RepoURL == repoURL {
			return &c.Projects[i]
		}
	}
	return nil
}

// AddProject 添加项目
func (c *TeamProjectConfig) AddProject(project ProjectMatcher) {
	for i := range c.Projects {
		if c.Projects[i].ID == project.ID || c.Projects[i].RepoURL == project.RepoURL {
			c.Projects[i] = project
			c.UpdatedAt = time.Now()
			return
		}
	}
	c.Projects = append(c.Projects, project)
	c.UpdatedAt = time.Now()
}

// RemoveProject 移除项目
func (c *TeamProjectConfig) RemoveProject(projectID string) bool {
	for i := range c.Projects {
		if c.Projects[i].ID == projectID {
			c.Projects = append(c.Projects[:i], c.Projects[i+1:]...)
			c.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// GetRepoURLs 获取所有项目的 RepoURL 列表
func (c *TeamProjectConfig) GetRepoURLs() []string {
	urls := make([]string, len(c.Projects))
	for i, p := range c.Projects {
		urls[i] = p.RepoURL
	}
	return urls
}

// =====================
// 成员周统计数据
// =====================

// MemberWeeklyStats 成员周统计数据
type MemberWeeklyStats struct {
	MemberID   string             `json:"member_id"`   // 成员 ID
	MemberName string             `json:"member_name"` // 成员名称
	WeekStart  string             `json:"week_start"`  // 周一日期 YYYY-MM-DD
	DailyStats []MemberDailyStats `json:"daily_stats"` // 7 天数据
	UpdatedAt  time.Time          `json:"updated_at"`  // 数据更新时间
}

// MemberDailyStats 成员每日统计
type MemberDailyStats struct {
	Date        string             `json:"date"`                   // 日期 YYYY-MM-DD
	GitStats    *GitDailyStats     `json:"git_stats,omitempty"`    // Git 统计
	CursorStats *CursorDailyStats  `json:"cursor_stats,omitempty"` // Cursor 统计
	WorkItems   []WorkItemSummary  `json:"work_items,omitempty"`   // 工作内容列表
	HasReport   bool               `json:"has_report"`             // 是否有日报
}

// GitDailyStats Git 每日统计
type GitDailyStats struct {
	TotalCommits int               `json:"total_commits"` // 总提交数
	TotalAdded   int               `json:"total_added"`   // 总添加行数
	TotalRemoved int               `json:"total_removed"` // 总删除行数
	Projects     []ProjectGitStats `json:"projects"`      // 按项目细分
}

// ProjectGitStats 项目 Git 统计
type ProjectGitStats struct {
	ProjectName    string          `json:"project_name"`    // 项目名称
	RepoURL        string          `json:"repo_url"`        // Git Remote URL
	Commits        int             `json:"commits"`         // 提交数
	LinesAdded     int             `json:"lines_added"`     // 添加行数
	LinesRemoved   int             `json:"lines_removed"`   // 删除行数
	CommitMessages []CommitSummary `json:"commit_messages"` // 提交摘要列表
}

// CommitSummary 提交摘要
type CommitSummary struct {
	Hash       string `json:"hash"`        // 简短 hash
	Message    string `json:"message"`     // commit message（首行）
	Time       string `json:"time"`        // 提交时间
	FilesCount int    `json:"files_count"` // 变更文件数
}

// CursorDailyStats Cursor 每日统计
type CursorDailyStats struct {
	SessionCount int `json:"session_count"` // 会话数
	TokensUsed   int `json:"tokens_used"`   // Token 使用量
	LinesAdded   int `json:"lines_added"`   // AI 生成添加行数
	LinesRemoved int `json:"lines_removed"` // AI 生成删除行数
}

// WorkItemSummary 工作条目摘要
type WorkItemSummary struct {
	Project     string `json:"project"`     // 项目名
	Category    string `json:"category"`    // 工作类型
	Description string `json:"description"` // 描述
}

// =====================
// 团队周视图（Leader 展示用）
// =====================

// TeamWeeklyView 团队周视图
type TeamWeeklyView struct {
	TeamID         string             `json:"team_id"`         // 团队 ID
	WeekStart      string             `json:"week_start"`      // 周一日期
	WeekEnd        string             `json:"week_end"`        // 周日日期
	Calendar       []TeamDayColumn    `json:"calendar"`        // 7 天列
	ProjectSummary []ProjectWeekStats `json:"project_summary"` // 项目汇总
	UpdatedAt      time.Time          `json:"updated_at"`      // 更新时间
}

// TeamDayColumn 日历中的一天（纵向切割）
type TeamDayColumn struct {
	Date      string          `json:"date"`        // 日期 YYYY-MM-DD
	DayOfWeek int             `json:"day_of_week"` // 1=周一...7=周日
	Members   []MemberDayCell `json:"members"`     // 每个成员当天的数据
}

// MemberDayCell 日历格子（一个成员一天的数据）
type MemberDayCell struct {
	MemberID      string `json:"member_id"`      // 成员 ID
	MemberName    string `json:"member_name"`    // 成员名称
	ActivityLevel int    `json:"activity_level"` // 活跃度级别 0-4
	Commits       int    `json:"commits"`        // 提交数
	LinesChanged  int    `json:"lines_changed"`  // 变更行数（added + removed）
	HasReport     bool   `json:"has_report"`     // 是否有日报
	IsOnline      bool   `json:"is_online"`      // 是否在线
}

// CalculateActivityLevel 计算活跃度级别
// 基于 commits 数量：0=无活动, 1=1-2, 2=3-5, 3=6-10, 4=>10
func (c *MemberDayCell) CalculateActivityLevel() {
	switch {
	case c.Commits == 0:
		c.ActivityLevel = 0
	case c.Commits <= 2:
		c.ActivityLevel = 1
	case c.Commits <= 5:
		c.ActivityLevel = 2
	case c.Commits <= 10:
		c.ActivityLevel = 3
	default:
		c.ActivityLevel = 4
	}
}

// ProjectWeekStats 项目周统计
type ProjectWeekStats struct {
	ProjectName  string             `json:"project_name"`  // 项目名称
	RepoURL      string             `json:"repo_url"`      // Git Remote URL
	TotalCommits int                `json:"total_commits"` // 总提交数
	TotalAdded   int                `json:"total_added"`   // 总添加行数
	TotalRemoved int                `json:"total_removed"` // 总删除行数
	Contributors []ContributorStats `json:"contributors"`  // 贡献者列表
}

// ContributorStats 贡献者统计
type ContributorStats struct {
	MemberID     string `json:"member_id"`     // 成员 ID
	MemberName   string `json:"member_name"`   // 成员名称
	Commits      int    `json:"commits"`       // 提交数
	LinesAdded   int    `json:"lines_added"`   // 添加行数
	LinesRemoved int    `json:"lines_removed"` // 删除行数
}

// =====================
// 成员日详情（点击格子后展示）
// =====================

// MemberDailyDetail 成员日详情
type MemberDailyDetail struct {
	MemberID   string             `json:"member_id"`   // 成员 ID
	MemberName string             `json:"member_name"` // 成员名称
	Date       string             `json:"date"`        // 日期
	GitStats   *GitDailyStats     `json:"git_stats"`   // Git 统计
	CursorStats *CursorDailyStats `json:"cursor_stats"` // Cursor 统计
	WorkItems  []WorkItemSummary  `json:"work_items"`  // 工作条目
	HasReport  bool               `json:"has_report"`  // 是否有日报
	IsOnline   bool               `json:"is_online"`   // 成员是否在线
	IsCached   bool               `json:"is_cached"`   // 是否为缓存数据
}

// =====================
// 周统计缓存条目
// =====================

// WeeklyStatsCacheEntry 周统计缓存条目
type WeeklyStatsCacheEntry struct {
	MemberID   string            `json:"member_id"`   // 成员 ID
	WeekStart  string            `json:"week_start"`  // 周一日期
	Stats      MemberWeeklyStats `json:"stats"`       // 统计数据
	CachedAt   time.Time         `json:"cached_at"`   // 缓存时间
	ExpireAt   time.Time         `json:"expire_at"`   // 过期时间
}

// IsExpired 检查缓存是否过期
func (e *WeeklyStatsCacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpireAt)
}

// WeeklyStatsCache 周统计缓存
type WeeklyStatsCache struct {
	TeamID    string                  `json:"team_id"`    // 团队 ID
	Entries   []WeeklyStatsCacheEntry `json:"entries"`    // 缓存条目
	UpdatedAt time.Time               `json:"updated_at"` // 更新时间
}

// GetEntry 获取缓存条目
func (c *WeeklyStatsCache) GetEntry(memberID, weekStart string) *WeeklyStatsCacheEntry {
	for i := range c.Entries {
		if c.Entries[i].MemberID == memberID && c.Entries[i].WeekStart == weekStart {
			return &c.Entries[i]
		}
	}
	return nil
}

// SetEntry 设置缓存条目
func (c *WeeklyStatsCache) SetEntry(entry WeeklyStatsCacheEntry) {
	for i := range c.Entries {
		if c.Entries[i].MemberID == entry.MemberID && c.Entries[i].WeekStart == entry.WeekStart {
			c.Entries[i] = entry
			c.UpdatedAt = time.Now()
			return
		}
	}
	c.Entries = append(c.Entries, entry)
	c.UpdatedAt = time.Now()
}

// =====================
// 成员周报汇总（团队周报素材）
// =====================

// MemberWeeklySummaryInfo 成员周报信息（从成员端拉取）
type MemberWeeklySummaryInfo struct {
	MemberID   string `json:"member_id"`   // 成员 ID
	MemberName string `json:"member_name"` // 成员名称
	WeekStart  string `json:"week_start"`  // 周起始日期
	HasSummary bool   `json:"has_summary"` // 是否有周报
	Summary    string `json:"summary"`     // 周报 Markdown 内容
	IsOnline   bool   `json:"is_online"`   // 成员是否在线
	Error      string `json:"error,omitempty"` // 拉取失败的错误信息
}

// TeamMemberSummariesView 团队成员周报汇总视图
type TeamMemberSummariesView struct {
	TeamID     string                    `json:"team_id"`     // 团队 ID
	TeamName   string                    `json:"team_name"`   // 团队名称
	WeekStart  string                    `json:"week_start"`  // 周起始日期
	WeekEnd    string                    `json:"week_end"`    // 周结束日期
	Members    []MemberWeeklySummaryInfo `json:"members"`     // 成员周报列表
	AllReady   bool                      `json:"all_ready"`   // 是否所有成员都有周报
	MissingMembers []string              `json:"missing_members"` // 缺少周报的成员名称列表
}

// CleanExpired 清理过期条目
func (c *WeeklyStatsCache) CleanExpired() int {
	now := time.Now()
	var validEntries []WeeklyStatsCacheEntry
	for _, entry := range c.Entries {
		if !entry.IsExpired() || now.Sub(entry.ExpireAt) < 24*time.Hour*7 {
			// 保留未过期的，或过期不超过 7 天的（用于离线成员）
			validEntries = append(validEntries, entry)
		}
	}
	removed := len(c.Entries) - len(validEntries)
	c.Entries = validEntries
	if removed > 0 {
		c.UpdatedAt = time.Now()
	}
	return removed
}
