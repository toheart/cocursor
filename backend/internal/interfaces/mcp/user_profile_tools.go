package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/cocursor/backend/internal/infrastructure/config"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// 用户消息内容最大长度（超过此长度将被截断）
const maxUserMessageContentLength = 500

// detectChineseRatio 检测文本中中文字符的比例
func detectChineseRatio(text string) float64 {
	if len(text) == 0 {
		return 0
	}
	
	chineseCount := 0
	totalCount := 0
	
	for _, r := range text {
		// 跳过空白字符和标点
		if r <= 127 {
			continue
		}
		totalCount++
		// 中文 Unicode 范围：CJK Unified Ideographs
		if r >= 0x4E00 && r <= 0x9FFF {
			chineseCount++
		}
	}
	
	if totalCount == 0 {
		return 0
	}
	return float64(chineseCount) / float64(totalCount)
}

// GetUserMessagesForProfileInput 获取用户消息用于画像分析的工具输入
type GetUserMessagesForProfileInput struct {
	Scope             string  `json:"scope" jsonschema:"Scope: 'global' for all projects or 'project' for specific project"`
	ProjectPath       string  `json:"project_path,omitempty" jsonschema:"Project path (required when scope is 'project')"`
	DaysBack          int     `json:"days_back,omitempty" jsonschema:"Number of days to analyze, defaults to 30"`
	RecentSessions    int     `json:"recent_sessions,omitempty" jsonschema:"Number of recent sessions to fully extract, defaults to 10"`
	SamplingRate      float64 `json:"sampling_rate,omitempty" jsonschema:"Sampling rate for historical sessions, defaults to 0.3"`
	MaxHistoricalMsgs int     `json:"max_historical_msgs,omitempty" jsonschema:"Maximum historical messages, defaults to 200"`
}

// GetUserMessagesForProfileOutput 获取用户消息用于画像分析的工具输出
type GetUserMessagesForProfileOutput struct {
	Messages        *UserMessagesGroup `json:"messages" jsonschema:"User messages grouped by recency"`
	Stats           *ProfileStats      `json:"stats" jsonschema:"Statistics computed from user messages"`
	ExistingProfile *string            `json:"existing_profile,omitempty" jsonschema:"Existing profile content if any"`
	Meta            *ProfileMeta       `json:"meta" jsonschema:"Metadata for idempotency check"`
}

// UserMessagesGroup 用户消息分组
type UserMessagesGroup struct {
	Recent     []*UserMessage `json:"recent" jsonschema:"Messages from recent sessions (fully extracted)"`
	Historical []*UserMessage `json:"historical" jsonschema:"Messages from historical sessions (sampled)"`
}

// UserMessage 用户消息
type UserMessage struct {
	Content     string `json:"content" jsonschema:"User message content (filtered)"`
	Timestamp   int64  `json:"timestamp" jsonschema:"Timestamp in milliseconds"`
	Project     string `json:"project" jsonschema:"Project name"`
	SessionName string `json:"session_name" jsonschema:"Session name"`
	// SessionID 已移除，对画像分析无用
}

// ProfileStats 画像统计特征
type ProfileStats struct {
	TotalSessions       int            `json:"total_sessions" jsonschema:"Total number of sessions analyzed"`
	TotalUserMessages   int            `json:"total_user_messages" jsonschema:"Total number of user messages"`
	TimeDistribution    map[string]int `json:"time_distribution" jsonschema:"Message count by hour (0-23)"`
	ProjectDistribution map[string]int `json:"project_distribution" jsonschema:"Message count by project"`
	DateRange           *DateRange     `json:"date_range" jsonschema:"Date range of analyzed data"`
	PrimaryLanguage     string         `json:"primary_language" jsonschema:"Primary language detected: 'zh' for Chinese, 'en' for English"`
}

// DateRange 日期范围
type DateRange struct {
	Start string `json:"start" jsonschema:"Start date (YYYY-MM-DD)"`
	End   string `json:"end" jsonschema:"End date (YYYY-MM-DD)"`
}

// ProfileMeta 画像元数据
type ProfileMeta struct {
	LastAnalyzed string `json:"last_analyzed,omitempty" jsonschema:"Last analysis timestamp"`
	DataHash     string `json:"data_hash" jsonschema:"Hash of session data for idempotency"`
	NeedsUpdate  bool   `json:"needs_update" jsonschema:"Whether new data is available for analysis"`
}

// SaveUserProfileInput 保存用户画像的工具输入
type SaveUserProfileInput struct {
	Scope       string `json:"scope" jsonschema:"Scope: 'global' or 'project'"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Project path (required when scope is 'project')"`
	Content     string `json:"content" jsonschema:"Profile content in Markdown format (without frontmatter)"`
	Language    string `json:"language,omitempty" jsonschema:"Language for frontmatter description: 'zh' or 'en', defaults to 'en'"`
}

// SaveUserProfileOutput 保存用户画像的工具输出
type SaveUserProfileOutput struct {
	Success    bool   `json:"success" jsonschema:"Whether the operation succeeded"`
	FilePath   string `json:"file_path" jsonschema:"Path where profile was saved"`
	GitIgnored bool   `json:"git_ignored" jsonschema:"Whether .gitignore was updated"`
	Message    string `json:"message" jsonschema:"Operation message"`
}

// getUserMessagesForProfileTool 获取用户消息用于画像分析
func (s *MCPServer) getUserMessagesForProfileTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetUserMessagesForProfileInput,
) (*mcp.CallToolResult, GetUserMessagesForProfileOutput, error) {
	// 设置默认值
	if input.Scope == "" {
		input.Scope = "global"
	}
	if input.DaysBack <= 0 {
		input.DaysBack = 30
	}
	if input.RecentSessions <= 0 {
		input.RecentSessions = 5 // 减少到 5 个最近会话
	}
	if input.SamplingRate <= 0 {
		input.SamplingRate = 0.2 // 降低采样率
	}
	if input.MaxHistoricalMsgs <= 0 {
		input.MaxHistoricalMsgs = 100 // 减少历史消息上限
	}

	// 验证参数
	if input.Scope == "project" && input.ProjectPath == "" {
		return nil, GetUserMessagesForProfileOutput{}, fmt.Errorf("project_path is required when scope is 'project'")
	}

	// 计算时间范围
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -input.DaysBack)
	startTimestamp := startDate.UnixMilli()
	endTimestamp := endDate.UnixMilli()

	// 获取所有项目
	projects := s.projectManager.ListAllProjects()

	// 如果是项目级别，过滤到指定项目
	if input.Scope == "project" {
		var filteredProjects []*domainCursor.ProjectInfo
		for _, p := range projects {
			for _, ws := range p.Workspaces {
				if ws.Path == input.ProjectPath {
					filteredProjects = append(filteredProjects, p)
					break
				}
			}
		}
		projects = filteredProjects
	}

	// 收集所有会话
	type sessionWithProject struct {
		Session     *domainCursor.ComposerData
		ProjectName string
		ProjectPath string
		WorkspaceID string
	}

	var allSessions []*sessionWithProject
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

			// 筛选时间范围内的会话
			for i := range composers {
				session := &composers[i]
				updatedAt := session.LastUpdatedAt

				if updatedAt >= startTimestamp && updatedAt <= endTimestamp {
					allSessions = append(allSessions, &sessionWithProject{
						Session:     session,
						ProjectName: ws.ProjectName,
						ProjectPath: ws.Path,
						WorkspaceID: ws.WorkspaceID,
					})
				}
			}
		}
	}

	// 按更新时间倒序排序（最新的在前）
	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].Session.LastUpdatedAt > allSessions[j].Session.LastUpdatedAt
	})

	// 创建 SessionService
	sessionService := appCursor.NewSessionService(s.projectManager, s.sessionRepo)

	// 提取用户消息
	var recentMessages []*UserMessage
	var historicalMessages []*UserMessage
	timeDistribution := make(map[string]int)
	projectDistribution := make(map[string]int)
	var hashData []string
	var allTextForLangDetect strings.Builder // 用于语言检测的文本累积

	for i, swp := range allSessions {
		// 获取会话内容
		options := &appCursor.TextContentOptions{
			FilterLogsAndCode: true,
			MaxMessageLength:  5000,
		}
		messages, err := sessionService.GetSessionTextContentWithOptions(swp.Session.ComposerID, options)
		if err != nil {
			continue
		}

		// 只提取用户消息
		for _, msg := range messages {
			if msg.Type != domainCursor.MessageTypeUser {
				continue
			}

			// 消息内容已在 GetSessionTextContentWithOptions 中过滤（移除了 attached_files、code_selection 等）
			text := strings.TrimSpace(msg.Text)
			if text == "" {
				continue
			}

			// 截断超长消息（对画像分析来说，关键信息通常在开头）
			if len(text) > maxUserMessageContentLength {
				text = text[:maxUserMessageContentLength] + "..."
			}

			userMsg := &UserMessage{
				Content:     text,
				Timestamp:   msg.Timestamp,
				Project:     swp.ProjectName,
				SessionName: swp.Session.Name,
			}

			// 累积文本用于语言检测（限制总量避免过长）
			if allTextForLangDetect.Len() < 50000 {
				allTextForLangDetect.WriteString(text)
				allTextForLangDetect.WriteString(" ")
			}

			// 更新统计
			hour := time.UnixMilli(msg.Timestamp).Hour()
			hourKey := fmt.Sprintf("%02d", hour)
			timeDistribution[hourKey]++
			projectDistribution[swp.ProjectName]++

			// 添加到 hash 数据
			hashData = append(hashData, fmt.Sprintf("%s:%d", swp.Session.ComposerID, msg.Timestamp))

			if i < input.RecentSessions {
				// 最近会话：完整提取
				recentMessages = append(recentMessages, userMsg)
			} else {
				// 历史会话：采样
				if rand.Float64() < input.SamplingRate && len(historicalMessages) < input.MaxHistoricalMsgs {
					historicalMessages = append(historicalMessages, userMsg)
				}
			}
		}
	}

	// 计算数据指纹
	sort.Strings(hashData)
	hashInput := strings.Join(hashData, "|")
	hash := sha256.Sum256([]byte(hashInput))
	dataHash := hex.EncodeToString(hash[:16]) // 使用前 16 字节

	// 检测主要语言
	chineseRatio := detectChineseRatio(allTextForLangDetect.String())
	primaryLanguage := "en"
	if chineseRatio > 0.3 { // 中文字符超过 30% 认为是中文为主
		primaryLanguage = "zh"
	}

	// 读取现有 Profile
	var existingProfile *string
	var lastAnalyzed string
	needsUpdate := true

	profilePath, metaPath := getProfilePaths(input.Scope, input.ProjectPath)
	if content, err := os.ReadFile(profilePath); err == nil {
		profileStr := string(content)
		existingProfile = &profileStr
	}

	// 读取元数据检查幂等性
	if metaContent, err := os.ReadFile(metaPath); err == nil {
		var meta ProfileMeta
		if json.Unmarshal(metaContent, &meta) == nil {
			lastAnalyzed = meta.LastAnalyzed
			if meta.DataHash == dataHash {
				needsUpdate = false
			}
		}
	}

	output := GetUserMessagesForProfileOutput{
		Messages: &UserMessagesGroup{
			Recent:     recentMessages,
			Historical: historicalMessages,
		},
		Stats: &ProfileStats{
			TotalSessions:       len(allSessions),
			TotalUserMessages:   len(recentMessages) + len(historicalMessages),
			TimeDistribution:    timeDistribution,
			ProjectDistribution: projectDistribution,
			DateRange: &DateRange{
				Start: startDate.Format("2006-01-02"),
				End:   endDate.Format("2006-01-02"),
			},
			PrimaryLanguage: primaryLanguage,
		},
		ExistingProfile: existingProfile,
		Meta: &ProfileMeta{
			LastAnalyzed: lastAnalyzed,
			DataHash:     dataHash,
			NeedsUpdate:  needsUpdate,
		},
	}

	return nil, output, nil
}

// saveUserProfileTool 保存用户画像
func (s *MCPServer) saveUserProfileTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SaveUserProfileInput,
) (*mcp.CallToolResult, SaveUserProfileOutput, error) {
	// 验证参数
	if input.Scope == "" {
		input.Scope = "global"
	}
	if input.Scope == "project" && input.ProjectPath == "" {
		return nil, SaveUserProfileOutput{}, fmt.Errorf("project_path is required when scope is 'project'")
	}
	if input.Content == "" {
		return nil, SaveUserProfileOutput{}, fmt.Errorf("content is required")
	}

	profilePath, metaPath := getProfilePaths(input.Scope, input.ProjectPath)

	// 确保目录存在
	profileDir := filepath.Dir(profilePath)
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return nil, SaveUserProfileOutput{
			Success: false,
			Message: fmt.Sprintf("failed to create directory: %v", err),
		}, nil
	}

	// 准备内容
	var finalContent string
	if input.Scope == "project" {
		// 项目级：添加 YAML frontmatter
		// 根据 Cursor Rules 文档：description 是必需的，alwaysApply: true 确保每次对话都加载
		// 根据用户语言选择 description
		description := "User profile containing coding style, technical preferences, communication habits, and work patterns. This rule helps AI understand the user better and provide more personalized responses."
		if input.Language == "zh" {
			description = "用户画像规则，包含编码风格、技术偏好、沟通习惯和工作模式，帮助 AI 更好地理解用户并提供个性化回复。"
		}
		finalContent = fmt.Sprintf(`---
description: %s
alwaysApply: true
---

%s`, description, input.Content)
	} else {
		// 全局级：纯 Markdown
		finalContent = input.Content
	}

	// 写入 Profile 文件
	if err := os.WriteFile(profilePath, []byte(finalContent), 0644); err != nil {
		return nil, SaveUserProfileOutput{
			Success: false,
			Message: fmt.Sprintf("failed to write profile: %v", err),
		}, nil
	}

	// 更新元数据
	meta := ProfileMeta{
		LastAnalyzed: time.Now().Format(time.RFC3339),
		DataHash:     "", // 由下次获取时计算
		NeedsUpdate:  false,
	}
	metaContent, _ := json.MarshalIndent(meta, "", "  ")
	os.WriteFile(metaPath, metaContent, 0644)

	// 项目级：更新 .gitignore
	gitIgnored := false
	if input.Scope == "project" {
		gitIgnored = updateGitIgnore(input.ProjectPath, ".cursor/rules/user-profile.mdc")
	}

	return nil, SaveUserProfileOutput{
		Success:    true,
		FilePath:   profilePath,
		GitIgnored: gitIgnored,
		Message:    "Profile saved successfully",
	}, nil
}

// getProfilePaths 获取 Profile 和元数据文件路径
func getProfilePaths(scope, projectPath string) (profilePath, metaPath string) {
	if scope == "project" {
		profilePath = filepath.Join(projectPath, ".cursor", "rules", "user-profile.mdc")
		metaPath = filepath.Join(projectPath, ".cursor", "rules", "user-profile.meta.json")
	} else {
		profilePath = filepath.Join(config.GetDataDir(), "profiles", "global.md")
		metaPath = filepath.Join(config.GetDataDir(), "profiles", "global.meta.json")
	}
	return
}

// updateGitIgnore 更新 .gitignore 文件
func updateGitIgnore(projectPath, entry string) bool {
	gitignorePath := filepath.Join(projectPath, ".gitignore")

	// 读取现有内容
	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return false
	}

	// 检查是否已存在
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return true // 已存在
		}
	}

	// 添加新条目
	var newContent string
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		newContent = string(content) + "\n" + entry + "\n"
	} else {
		newContent = string(content) + entry + "\n"
	}

	if err := os.WriteFile(gitignorePath, []byte(newContent), 0644); err != nil {
		return false
	}

	return true
}
