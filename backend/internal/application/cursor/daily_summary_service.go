package cursor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/cocursor/backend/internal/infrastructure/storage"
)

// DailySummaryService 日报服务（统一 MCP 和 HTTP 的业务逻辑）
type DailySummaryService struct {
	projectManager *ProjectManager
	summaryRepo    storage.DailySummaryRepository
	sessionRepo    storage.WorkspaceSessionRepository
	pathResolver   *infraCursor.PathResolver
	dbReader       *infraCursor.DBReader
}

// NewDailySummaryService 创建日报服务实例
func NewDailySummaryService(
	projectManager *ProjectManager,
	summaryRepo storage.DailySummaryRepository,
	sessionRepo storage.WorkspaceSessionRepository,
) *DailySummaryService {
	return &DailySummaryService{
		projectManager: projectManager,
		summaryRepo:    summaryRepo,
		sessionRepo:    sessionRepo,
		pathResolver:   infraCursor.NewPathResolver(),
		dbReader:       infraCursor.NewDBReader(),
	}
}

// ProjectSessions 项目会话组
type ProjectSessions struct {
	ProjectName string         `json:"project_name"`
	ProjectPath string         `json:"project_path"`
	WorkspaceID string         `json:"workspace_id"`
	Sessions    []*SessionInfo `json:"sessions"`
}

// SessionInfo 会话信息
type SessionInfo struct {
	SessionID      string `json:"session_id"`
	Name           string `json:"name"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
	TranscriptPath string `json:"transcript_path,omitempty"`
}

// DailySessionsResult 每日会话查询结果
type DailySessionsResult struct {
	Date     string             `json:"date"`
	Projects []*ProjectSessions `json:"projects"`
	Total    int                `json:"total"`
}

// GetDailySessions 获取每日会话列表
func (s *DailySummaryService) GetDailySessions(date string) (*DailySessionsResult, error) {
	// 确定日期
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// 解析日期
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// 计算日期的开始和结束时间（本地时区）
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.Local)
	endOfDay := startOfDay.Add(24 * time.Hour)
	startTimestamp := startOfDay.UnixMilli()
	endTimestamp := endOfDay.UnixMilli()

	// 获取所有项目
	projects := s.projectManager.ListAllProjects()

	// 按项目分组收集会话
	projectSessionsMap := make(map[string]*ProjectSessions)

	for _, project := range projects {
		for _, ws := range project.Workspaces {
			// 获取工作区数据库路径
			workspaceDBPath, err := s.pathResolver.GetWorkspaceDBPath(ws.WorkspaceID)
			if err != nil {
				continue
			}

			// 读取 composer.composerData
			composerDataValue, err := s.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
			if err != nil {
				continue
			}

			// 解析 Composer 数据
			composers, err := domainCursor.ParseComposerData(string(composerDataValue))
			if err != nil {
				continue
			}

			// 筛选当天创建或更新的会话
			for i := range composers {
				session := &composers[i]
				createdAt := session.CreatedAt
				updatedAt := session.LastUpdatedAt

				isCreatedToday := createdAt >= startTimestamp && createdAt < endTimestamp
				isUpdatedToday := updatedAt >= startTimestamp && updatedAt < endTimestamp

				if !isCreatedToday && !isUpdatedToday {
					continue
				}

				// 获取或创建项目会话组
				projectKey := ws.ProjectName
				if _, exists := projectSessionsMap[projectKey]; !exists {
					projectSessionsMap[projectKey] = &ProjectSessions{
						ProjectName: ws.ProjectName,
						ProjectPath: ws.Path,
						WorkspaceID: ws.WorkspaceID,
						Sessions:    []*SessionInfo{},
					}
				}

				// 添加到项目会话组
				projectSessionsMap[projectKey].Sessions = append(projectSessionsMap[projectKey].Sessions, &SessionInfo{
					SessionID: session.ComposerID,
					Name:      session.Name,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				})
			}
		}
	}

	// 转换为切片
	projectsList := make([]*ProjectSessions, 0, len(projectSessionsMap))
	totalSessions := 0
	for _, ps := range projectSessionsMap {
		projectsList = append(projectsList, ps)
		totalSessions += len(ps.Sessions)
	}

	return &DailySessionsResult{
		Date:     date,
		Projects: projectsList,
		Total:    totalSessions,
	}, nil
}

// TextMessage 纯文本消息
type TextMessage struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	Timestamp int64  `json:"timestamp"`
}

// SessionConversation 会话对话
type SessionConversation struct {
	SessionID     string         `json:"session_id"`
	Name          string         `json:"name"`
	ProjectName   string         `json:"project_name"`
	CreatedAt     int64          `json:"created_at"`
	UpdatedAt     int64          `json:"updated_at"`
	Messages      []*TextMessage `json:"messages"`
	TotalMessages int            `json:"total_messages"`
}

// ProjectConversations 项目对话组
type ProjectConversations struct {
	ProjectName string                 `json:"project_name"`
	ProjectPath string                 `json:"project_path"`
	WorkspaceID string                 `json:"workspace_id"`
	Sessions    []*SessionConversation `json:"sessions"`
}

// DailyConversationsResult 每日对话查询结果
type DailyConversationsResult struct {
	Date     string                  `json:"date"`
	Projects []*ProjectConversations `json:"projects"`
	Total    int                     `json:"total"`
}

// GetDailyConversations 获取每日对话内容（一次性返回所有项目的所有对话）
func (s *DailySummaryService) GetDailyConversations(date string) (*DailyConversationsResult, error) {
	// 确定日期
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// 解析日期
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// 计算日期的开始和结束时间（本地时区）
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.Local)
	endOfDay := startOfDay.Add(24 * time.Hour)
	startTimestamp := startOfDay.UnixMilli()
	endTimestamp := endOfDay.UnixMilli()

	// 获取所有项目
	projects := s.projectManager.ListAllProjects()

	// 创建 SessionService
	sessionService := NewSessionService(s.projectManager, s.sessionRepo)

	// 按项目分组收集会话和对话内容
	projectConversationsMap := make(map[string]*ProjectConversations)
	totalSessions := 0

	for _, project := range projects {
		for _, ws := range project.Workspaces {
			// 获取工作区数据库路径
			workspaceDBPath, err := s.pathResolver.GetWorkspaceDBPath(ws.WorkspaceID)
			if err != nil {
				continue
			}

			// 读取 composer.composerData
			composerDataValue, err := s.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
			if err != nil {
				continue
			}

			// 解析 Composer 数据
			composers, err := domainCursor.ParseComposerData(string(composerDataValue))
			if err != nil {
				continue
			}

			// 筛选当天创建或更新的会话
			for i := range composers {
				session := &composers[i]
				createdAt := session.CreatedAt
				updatedAt := session.LastUpdatedAt

				isCreatedToday := createdAt >= startTimestamp && createdAt < endTimestamp
				isUpdatedToday := updatedAt >= startTimestamp && updatedAt < endTimestamp

				if !isCreatedToday && !isUpdatedToday {
					continue
				}

				// 获取或创建项目对话组
				projectKey := ws.ProjectName
				if _, exists := projectConversationsMap[projectKey]; !exists {
					projectConversationsMap[projectKey] = &ProjectConversations{
						ProjectName: ws.ProjectName,
						ProjectPath: ws.Path,
						WorkspaceID: ws.WorkspaceID,
						Sessions:    []*SessionConversation{},
					}
				}

				// 获取会话的文本内容
				options := &TextContentOptions{
					FilterLogsAndCode: true,
					MaxMessageLength:  5000,
				}
				textMessages, err := sessionService.GetSessionTextContentWithOptions(session.ComposerID, options)
				
				var messages []*TextMessage
				if err == nil {
					for _, msg := range textMessages {
						messages = append(messages, &TextMessage{
							Type:      string(msg.Type),
							Text:      msg.Text,
							Timestamp: msg.Timestamp,
						})
					}
				}

				// 添加到项目对话组
				projectConversationsMap[projectKey].Sessions = append(
					projectConversationsMap[projectKey].Sessions,
					&SessionConversation{
						SessionID:     session.ComposerID,
						Name:          session.Name,
						ProjectName:   ws.ProjectName,
						CreatedAt:     createdAt,
						UpdatedAt:     updatedAt,
						Messages:      messages,
						TotalMessages: len(messages),
					},
				)
				totalSessions++
			}
		}
	}

	// 转换为切片
	projectsList := make([]*ProjectConversations, 0, len(projectConversationsMap))
	for _, pc := range projectConversationsMap {
		projectsList = append(projectsList, pc)
	}

	return &DailyConversationsResult{
		Date:     date,
		Projects: projectsList,
		Total:    totalSessions,
	}, nil
}

// SaveDailySummaryInput 保存日报输入
type SaveDailySummaryInput struct {
	Date              string                                 `json:"date"`
	Summary           string                                 `json:"summary"`
	Language          string                                 `json:"language"`
	Projects          []*domainCursor.ProjectSummary         `json:"projects"`
	Categories        *domainCursor.WorkCategories           `json:"categories"`
	TotalSessions     int                                    `json:"total_sessions"`
	CodeChanges       *domainCursor.CodeChangeSummary        `json:"code_changes,omitempty"`
	TimeDistribution  *domainCursor.TimeDistributionSummary  `json:"time_distribution,omitempty"`
	EfficiencyMetrics *domainCursor.EfficiencyMetricsSummary `json:"efficiency_metrics,omitempty"`
}

// SaveDailySummaryResult 保存日报结果
type SaveDailySummaryResult struct {
	Success   bool   `json:"success"`
	SummaryID string `json:"summary_id,omitempty"`
	Message   string `json:"message"`
}

// SaveDailySummary 保存日报
func (s *DailySummaryService) SaveDailySummary(input *SaveDailySummaryInput) (*SaveDailySummaryResult, error) {
	// 验证必需字段
	if input.Date == "" {
		return nil, fmt.Errorf("date is required")
	}
	if input.Summary == "" {
		return nil, fmt.Errorf("summary is required")
	}
	if input.Language == "" {
		input.Language = "zh"
	}

	// 处理 categories 参数
	categories := input.Categories
	if categories == nil {
		categories = &domainCursor.WorkCategories{}
	}

	// 构建 DailySummary 对象
	summary := &domainCursor.DailySummary{
		Date:              input.Date,
		Summary:           input.Summary,
		Language:          input.Language,
		Projects:          input.Projects,
		WorkCategories:    categories,
		TotalSessions:     input.TotalSessions,
		CodeChanges:       input.CodeChanges,
		TimeDistribution:  input.TimeDistribution,
		EfficiencyMetrics: input.EfficiencyMetrics,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// 保存到数据库
	if err := s.summaryRepo.Save(summary); err != nil {
		return &SaveDailySummaryResult{
			Success: false,
			Message: fmt.Sprintf("save failed: %v", err),
		}, nil
	}

	return &SaveDailySummaryResult{
		Success:   true,
		SummaryID: summary.ID,
		Message:   "save successful",
	}, nil
}

// GetDailySummary 查询日报
func (s *DailySummaryService) GetDailySummary(date string) (*domainCursor.DailySummary, bool, error) {
	if date == "" {
		return nil, false, fmt.Errorf("date is required")
	}

	summary, err := s.summaryRepo.FindByDate(date)
	if err != nil {
		return nil, false, err
	}

	if summary == nil {
		return nil, false, nil
	}

	return summary, true, nil
}

// GetDailySummariesRange 批量查询日报
func (s *DailySummaryService) GetDailySummariesRange(startDate, endDate string) ([]*domainCursor.DailySummary, error) {
	if startDate == "" || endDate == "" {
		return nil, fmt.Errorf("start_date and end_date are required")
	}

	return s.summaryRepo.FindByDateRange(startDate, endDate)
}

// ComputeDataHash 计算日报数据的哈希值（用于周报幂等检查）
func ComputeDataHash(summaries []*domainCursor.DailySummary) string {
	if len(summaries) == 0 {
		return ""
	}

	// 收集所有日报的关键信息
	var hashData []string
	for _, s := range summaries {
		// 使用日期、更新时间和会话数作为哈希输入
		hashData = append(hashData, fmt.Sprintf("%s:%d:%d", s.Date, s.UpdatedAt.Unix(), s.TotalSessions))
	}

	// 排序确保一致性
	sort.Strings(hashData)
	hashInput := strings.Join(hashData, "|")

	// 计算 SHA256 哈希
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:16]) // 使用前 16 字节
}
