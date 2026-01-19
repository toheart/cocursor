package mcp

import (
	"context"
	"fmt"
	"time"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetDailySessionsInput 每日会话查询工具输入
type GetDailySessionsInput struct {
	Date string `json:"date,omitempty" jsonschema:"Date in YYYY-MM-DD format, defaults to today"`
}

// GetDailySessionsOutput 每日会话查询工具输出
type GetDailySessionsOutput struct {
	Date     string             `json:"date" jsonschema:"Date"`
	Projects []*ProjectSessions `json:"projects" jsonschema:"Sessions grouped by project"`
	Total    int                `json:"total" jsonschema:"Total number of sessions"`
}

// ProjectSessions 项目会话组
type ProjectSessions struct {
	ProjectName string         `json:"project_name" jsonschema:"Project name"`
	ProjectPath string         `json:"project_path" jsonschema:"Project path"`
	WorkspaceID string         `json:"workspace_id" jsonschema:"Workspace ID"`
	Sessions    []*SessionInfo `json:"sessions" jsonschema:"Session list"`
}

// SessionInfo 会话信息
type SessionInfo struct {
	SessionID      string `json:"session_id" jsonschema:"Session ID"`
	Name           string `json:"name" jsonschema:"Session name"`
	CreatedAt      int64  `json:"created_at" jsonschema:"Creation timestamp in milliseconds"`
	UpdatedAt      int64  `json:"updated_at" jsonschema:"Update timestamp in milliseconds"`
	TranscriptPath string `json:"transcript_path,omitempty" jsonschema:"Transcript file path (optional)"`
}

// GetSessionContentInput 会话内容读取工具输入
type GetSessionContentInput struct {
	SessionID string `json:"session_id" jsonschema:"Session ID (required)"`
}

// GetSessionContentOutput 会话内容读取工具输出
type GetSessionContentOutput struct {
	SessionID     string         `json:"session_id" jsonschema:"Session ID"`
	Name          string         `json:"name" jsonschema:"Session name"`
	ProjectName   string         `json:"project_name,omitempty" jsonschema:"Project name"`
	Messages      []*TextMessage `json:"messages" jsonschema:"Plain text messages list"`
	TotalMessages int            `json:"total_messages" jsonschema:"Total number of messages"`
}

// TextMessage 纯文本消息（已过滤 tool 和代码）
type TextMessage struct {
	Type      string `json:"type" jsonschema:"Message type: user or ai"`
	Text      string `json:"text" jsonschema:"Message text (code blocks removed)"`
	Timestamp int64  `json:"timestamp" jsonschema:"Timestamp in milliseconds"`
}

// SaveDailySummaryInput 保存每日总结工具输入
type SaveDailySummaryInput struct {
	Date              string                                 `json:"date" jsonschema:"Date in YYYY-MM-DD format"`
	Summary           string                                 `json:"summary" jsonschema:"Summary content (Markdown)"`
	Language          string                                 `json:"language" jsonschema:"Language: zh/en"`
	Projects          []*domainCursor.ProjectSummary         `json:"projects" jsonschema:"Project list"`
	Categories        *domainCursor.WorkCategories           `json:"categories,omitempty" jsonschema:"Work category statistics object, containing requirements_discussion, coding, problem_solving, refactoring, code_review, documentation, testing, other fields (optional)"`
	TotalSessions     int                                    `json:"total_sessions" jsonschema:"Total number of sessions"`
	CodeChanges       *domainCursor.CodeChangeSummary        `json:"code_changes,omitempty" jsonschema:"Code change statistics (optional)"`
	TimeDistribution  *domainCursor.TimeDistributionSummary  `json:"time_distribution,omitempty" jsonschema:"Time distribution statistics (optional)"`
	EfficiencyMetrics *domainCursor.EfficiencyMetricsSummary `json:"efficiency_metrics,omitempty" jsonschema:"Efficiency metrics (optional)"`
}

// SaveDailySummaryOutput 保存每日总结工具输出
type SaveDailySummaryOutput struct {
	Success   bool   `json:"success" jsonschema:"Whether the operation succeeded"`
	SummaryID string `json:"summary_id,omitempty" jsonschema:"Summary ID"`
	Message   string `json:"message" jsonschema:"Message"`
}

// GetDailySummaryInput 查询每日总结工具输入
type GetDailySummaryInput struct {
	Date string `json:"date" jsonschema:"Date in YYYY-MM-DD format"`
}

// GetDailySummaryOutput 查询每日总结工具输出
type GetDailySummaryOutput struct {
	Summary *domainCursor.DailySummary `json:"summary,omitempty" jsonschema:"Summary object"`
	Found   bool                       `json:"found" jsonschema:"Whether the summary was found"`
}

// GetDailyConversationsInput 获取每日对话内容工具输入
type GetDailyConversationsInput struct {
	Date string `json:"date,omitempty" jsonschema:"Date in YYYY-MM-DD format, defaults to today"`
}

// GetDailyConversationsOutput 获取每日对话内容工具输出
type GetDailyConversationsOutput struct {
	Date     string                  `json:"date" jsonschema:"Date"`
	Projects []*ProjectConversations `json:"projects" jsonschema:"Conversations grouped by project"`
	Total    int                     `json:"total" jsonschema:"Total number of sessions"`
}

// ProjectConversations 项目对话组（包含完整的会话信息和消息内容）
type ProjectConversations struct {
	ProjectName string                 `json:"project_name" jsonschema:"Project name"`
	ProjectPath string                 `json:"project_path" jsonschema:"Project path"`
	WorkspaceID string                 `json:"workspace_id" jsonschema:"Workspace ID"`
	Sessions    []*SessionConversation `json:"sessions" jsonschema:"Session conversations list"`
}

// SessionConversation 会话对话（包含会话信息和消息内容）
type SessionConversation struct {
	SessionID     string         `json:"session_id" jsonschema:"Session ID"`
	Name          string         `json:"name" jsonschema:"Session name"`
	ProjectName   string         `json:"project_name" jsonschema:"Project name"`
	CreatedAt     int64          `json:"created_at" jsonschema:"Creation timestamp in milliseconds"`
	UpdatedAt     int64          `json:"updated_at" jsonschema:"Update timestamp in milliseconds"`
	Messages      []*TextMessage `json:"messages" jsonschema:"Plain text messages list"`
	TotalMessages int            `json:"total_messages" jsonschema:"Total number of messages"`
}

// getDailySessionsTool 获取每日会话列表工具
func (s *MCPServer) getDailySessionsTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetDailySessionsInput,
) (*mcp.CallToolResult, GetDailySessionsOutput, error) {
	// 确定日期
	date := input.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// 解析日期
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, GetDailySessionsOutput{}, fmt.Errorf("invalid date format: %w", err)
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
	pathResolver := infraCursor.NewPathResolver()
	dbReader := infraCursor.NewDBReader()

	for _, project := range projects {
		for _, ws := range project.Workspaces {
			// 获取工作区数据库路径
			workspaceDBPath, err := pathResolver.GetWorkspaceDBPath(ws.WorkspaceID)
			if err != nil {
				continue
			}

			// 读取 composer.composerData
			composerDataValue, err := dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
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
				// 检查是否在目标日期创建或更新
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

	output := GetDailySessionsOutput{
		Date:     date,
		Projects: projectsList,
		Total:    totalSessions,
	}

	return nil, output, nil
}

// getSessionContentTool 获取会话内容工具
func (s *MCPServer) getSessionContentTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetSessionContentInput,
) (*mcp.CallToolResult, GetSessionContentOutput, error) {
	if input.SessionID == "" {
		return nil, GetSessionContentOutput{}, fmt.Errorf("session_id 参数是必需的")
	}

	// 创建 SessionService 来获取会话文本内容
	// 使用深度过滤选项，过滤日志和代码相关内容，只保留自然语言文本
	sessionService := appCursor.NewSessionService(s.projectManager, s.sessionRepo)
	options := &appCursor.TextContentOptions{
		FilterLogsAndCode: true, // 启用深度过滤，用于日志分析
		MaxMessageLength:  5000,
	}
	textMessages, err := sessionService.GetSessionTextContentWithOptions(input.SessionID, options)
	if err != nil {
		return nil, GetSessionContentOutput{}, fmt.Errorf("无法获取会话内容: %w", err)
	}

	// 获取会话详情以获取名称和项目信息
	sessionDetail, err := sessionService.GetSessionDetail(input.SessionID, 1)
	if err != nil {
		return nil, GetSessionContentOutput{}, fmt.Errorf("无法获取会话详情: %w", err)
	}

	// 获取项目名称：通过查找所有项目，找到包含此会话的项目
	projectName := ""
	if sessionDetail.Session != nil {
		projects := s.projectManager.ListAllProjects()
		pathResolver := infraCursor.NewPathResolver()
		dbReader := infraCursor.NewDBReader()

		for _, project := range projects {
			for _, ws := range project.Workspaces {
				// 读取此工作区的 composer 数据
				workspaceDBPath, err := pathResolver.GetWorkspaceDBPath(ws.WorkspaceID)
				if err != nil {
					continue
				}

				composerDataValue, err := dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
				if err != nil {
					continue
				}

				composers, err := domainCursor.ParseComposerData(string(composerDataValue))
				if err != nil {
					continue
				}

				// 检查是否包含此会话
				for _, composer := range composers {
					if composer.ComposerID == input.SessionID {
						projectName = project.ProjectName
						break
					}
				}

				if projectName != "" {
					break
				}
			}
			if projectName != "" {
				break
			}
		}
	}

	// 转换为 TextMessage 格式
	var textMsgList []*TextMessage
	for _, msg := range textMessages {
		textMsgList = append(textMsgList, &TextMessage{
			Type:      string(msg.Type),
			Text:      msg.Text,
			Timestamp: msg.Timestamp,
		})
	}

	output := GetSessionContentOutput{
		SessionID:     input.SessionID,
		Name:          sessionDetail.Session.Name,
		ProjectName:   projectName,
		Messages:      textMsgList,
		TotalMessages: len(textMsgList),
	}

	return nil, output, nil
}

// saveDailySummaryTool 保存每日总结工具
func (s *MCPServer) saveDailySummaryTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SaveDailySummaryInput,
) (*mcp.CallToolResult, SaveDailySummaryOutput, error) {
	// 验证必需字段
	if input.Date == "" {
		return nil, SaveDailySummaryOutput{}, fmt.Errorf("date 参数是必需的")
	}
	if input.Summary == "" {
		return nil, SaveDailySummaryOutput{}, fmt.Errorf("summary 参数是必需的")
	}
	if input.Language == "" {
		input.Language = "zh" // 默认中文
	}

	// 处理 categories 参数：如果为 nil，创建空对象
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
		return nil, SaveDailySummaryOutput{
			Success: false,
			Message: fmt.Sprintf("保存失败: %v", err),
		}, nil
	}

	output := SaveDailySummaryOutput{
		Success:   true,
		SummaryID: summary.ID,
		Message:   "保存成功",
	}

	return nil, output, nil
}

// getDailySummaryTool 查询每日总结工具
func (s *MCPServer) getDailySummaryTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetDailySummaryInput,
) (*mcp.CallToolResult, GetDailySummaryOutput, error) {
	if input.Date == "" {
		return nil, GetDailySummaryOutput{}, fmt.Errorf("date 参数是必需的")
	}

	// 从数据库查询
	summary, err := s.summaryRepo.FindByDate(input.Date)
	if err != nil {
		return nil, GetDailySummaryOutput{
			Found: false,
		}, nil // 查询错误不返回错误，只返回 found=false
	}

	if summary == nil {
		return nil, GetDailySummaryOutput{
			Found: false,
		}, nil
	}

	output := GetDailySummaryOutput{
		Summary: summary,
		Found:   true,
	}

	return nil, output, nil
}

// getDailyConversationsTool 获取每日对话内容工具（一次性返回所有项目的所有对话）
func (s *MCPServer) getDailyConversationsTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetDailyConversationsInput,
) (*mcp.CallToolResult, GetDailyConversationsOutput, error) {
	// 确定日期
	date := input.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// 解析日期
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, GetDailyConversationsOutput{}, fmt.Errorf("invalid date format: %w", err)
	}

	// 计算日期的开始和结束时间（本地时区）
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.Local)
	endOfDay := startOfDay.Add(24 * time.Hour)
	startTimestamp := startOfDay.UnixMilli()
	endTimestamp := endOfDay.UnixMilli()

	// 获取所有项目
	projects := s.projectManager.ListAllProjects()

	// 创建 SessionService
	sessionService := appCursor.NewSessionService(s.projectManager, s.sessionRepo)

	// 按项目分组收集会话和对话内容
	projectConversationsMap := make(map[string]*ProjectConversations)
	pathResolver := infraCursor.NewPathResolver()
	dbReader := infraCursor.NewDBReader()
	totalSessions := 0

	for _, project := range projects {
		for _, ws := range project.Workspaces {
			// 获取工作区数据库路径
			workspaceDBPath, err := pathResolver.GetWorkspaceDBPath(ws.WorkspaceID)
			if err != nil {
				continue
			}

			// 读取 composer.composerData
			composerDataValue, err := dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
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
				// 检查是否在目标日期创建或更新
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
				// 使用深度过滤选项，过滤日志和代码相关内容，只保留自然语言文本
				options := &appCursor.TextContentOptions{
					FilterLogsAndCode: true, // 启用深度过滤，用于日志分析
					MaxMessageLength:  5000,
				}
				textMessages, err := sessionService.GetSessionTextContentWithOptions(session.ComposerID, options)
				if err != nil {
					// 如果获取内容失败，仍然添加会话信息，但消息列表为空
					projectConversationsMap[projectKey].Sessions = append(
						projectConversationsMap[projectKey].Sessions,
						&SessionConversation{
							SessionID:     session.ComposerID,
							Name:          session.Name,
							ProjectName:   ws.ProjectName,
							CreatedAt:     createdAt,
							UpdatedAt:     updatedAt,
							Messages:      []*TextMessage{},
							TotalMessages: 0,
						},
					)
					totalSessions++
					continue
				}

				// 转换为 TextMessage 格式
				var textMsgList []*TextMessage
				for _, msg := range textMessages {
					textMsgList = append(textMsgList, &TextMessage{
						Type:      string(msg.Type),
						Text:      msg.Text,
						Timestamp: msg.Timestamp,
					})
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
						Messages:      textMsgList,
						TotalMessages: len(textMsgList),
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

	output := GetDailyConversationsOutput{
		Date:     date,
		Projects: projectsList,
		Total:    totalSessions,
	}

	return nil, output, nil
}
