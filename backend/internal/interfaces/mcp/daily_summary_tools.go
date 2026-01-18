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
	Date string `json:"date,omitempty" jsonschema:"日期，格式 YYYY-MM-DD，默认今天"`
}

// GetDailySessionsOutput 每日会话查询工具输出
type GetDailySessionsOutput struct {
	Date     string             `json:"date" jsonschema:"日期"`
	Projects []*ProjectSessions `json:"projects" jsonschema:"按项目分组的会话"`
	Total    int                `json:"total" jsonschema:"总会话数"`
}

// ProjectSessions 项目会话组
type ProjectSessions struct {
	ProjectName string         `json:"project_name" jsonschema:"项目名称"`
	ProjectPath string         `json:"project_path" jsonschema:"项目路径"`
	WorkspaceID string         `json:"workspace_id" jsonschema:"工作区ID"`
	Sessions    []*SessionInfo `json:"sessions" jsonschema:"会话列表"`
}

// SessionInfo 会话信息
type SessionInfo struct {
	SessionID      string `json:"session_id" jsonschema:"会话ID"`
	Name           string `json:"name" jsonschema:"会话名称"`
	CreatedAt      int64  `json:"created_at" jsonschema:"创建时间戳（毫秒）"`
	UpdatedAt      int64  `json:"updated_at" jsonschema:"更新时间戳（毫秒）"`
	TranscriptPath string `json:"transcript_path,omitempty" jsonschema:"transcript 文件路径（可选）"`
}

// GetSessionContentInput 会话内容读取工具输入
type GetSessionContentInput struct {
	SessionID string `json:"session_id" jsonschema:"会话ID（必填）"`
}

// GetSessionContentOutput 会话内容读取工具输出
type GetSessionContentOutput struct {
	SessionID     string         `json:"session_id" jsonschema:"会话ID"`
	Name          string         `json:"name" jsonschema:"会话名称"`
	ProjectName   string         `json:"project_name,omitempty" jsonschema:"所属项目名称"`
	Messages      []*TextMessage `json:"messages" jsonschema:"纯文本消息列表"`
	TotalMessages int            `json:"total_messages" jsonschema:"总消息数"`
}

// TextMessage 纯文本消息（已过滤 tool 和代码）
type TextMessage struct {
	Type      string `json:"type" jsonschema:"消息类型：user 或 ai"`
	Text      string `json:"text" jsonschema:"消息文本（去除代码块）"`
	Timestamp int64  `json:"timestamp" jsonschema:"时间戳（毫秒）"`
}

// SaveDailySummaryInput 保存每日总结工具输入
type SaveDailySummaryInput struct {
	Date          string                         `json:"date" jsonschema:"日期 YYYY-MM-DD"`
	Summary       string                         `json:"summary" jsonschema:"总结内容（Markdown）"`
	Language      string                         `json:"language" jsonschema:"语言：zh/en"`
	Projects      []*domainCursor.ProjectSummary `json:"projects" jsonschema:"项目列表"`
	Categories    *domainCursor.WorkCategories   `json:"categories,omitempty" jsonschema:"工作分类统计对象，包含 requirements_discussion, coding, problem_solving, refactoring, code_review, documentation, testing, other 字段（可选）"`
	TotalSessions int                            `json:"total_sessions" jsonschema:"总会话数"`
}

// SaveDailySummaryOutput 保存每日总结工具输出
type SaveDailySummaryOutput struct {
	Success   bool   `json:"success" jsonschema:"是否成功"`
	SummaryID string `json:"summary_id,omitempty" jsonschema:"总结ID"`
	Message   string `json:"message" jsonschema:"消息"`
}

// GetDailySummaryInput 查询每日总结工具输入
type GetDailySummaryInput struct {
	Date string `json:"date" jsonschema:"日期 YYYY-MM-DD"`
}

// GetDailySummaryOutput 查询每日总结工具输出
type GetDailySummaryOutput struct {
	Summary *domainCursor.DailySummary `json:"summary,omitempty" jsonschema:"总结对象"`
	Found   bool                       `json:"found" jsonschema:"是否找到"`
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
	sessionService := appCursor.NewSessionService(s.projectManager)
	textMessages, err := sessionService.GetSessionTextContent(input.SessionID)
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
		Date:           input.Date,
		Summary:        input.Summary,
		Language:       input.Language,
		Projects:       input.Projects,
		WorkCategories: categories,
		TotalSessions:  input.TotalSessions,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
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
