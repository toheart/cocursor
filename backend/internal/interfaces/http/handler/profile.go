package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/cocursor/backend/internal/infrastructure/config"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// 用户消息内容最大长度（超过此长度将被截断）
const maxUserMessageContentLength = 500

// ProfileHandler 用户画像处理器
type ProfileHandler struct {
	projectManager *appCursor.ProjectManager
	sessionService *appCursor.SessionService
}

// NewProfileHandler 创建用户画像处理器
func NewProfileHandler(
	projectManager *appCursor.ProjectManager,
	sessionService *appCursor.SessionService,
) *ProfileHandler {
	return &ProfileHandler{
		projectManager: projectManager,
		sessionService: sessionService,
	}
}

// GetMessagesRequest 获取用户消息请求
type GetMessagesRequest struct {
	Scope             string  `json:"scope"`                         // "global" 或 "project"
	ProjectPath       string  `json:"project_path,omitempty"`        // 项目路径（scope 为 project 时必填）
	DaysBack          int     `json:"days_back,omitempty"`           // 分析天数，默认 30
	RecentSessions    int     `json:"recent_sessions,omitempty"`     // 完整提取的最近会话数，默认 5
	SamplingRate      float64 `json:"sampling_rate,omitempty"`       // 历史会话采样率，默认 0.2
	MaxHistoricalMsgs int     `json:"max_historical_msgs,omitempty"` // 最大历史消息数，默认 100
}

// UserMessage 用户消息
type UserMessage struct {
	Content     string `json:"content"`      // 消息内容
	Timestamp   int64  `json:"timestamp"`    // 时间戳（毫秒）
	Project     string `json:"project"`      // 项目名称
	SessionName string `json:"session_name"` // 会话名称
}

// UserMessagesGroup 用户消息分组
type UserMessagesGroup struct {
	Recent     []*UserMessage `json:"recent"`     // 最近会话的消息
	Historical []*UserMessage `json:"historical"` // 历史会话的消息（采样）
}

// ProfileStats 画像统计特征
type ProfileStats struct {
	TotalSessions       int            `json:"total_sessions"`       // 总会话数
	TotalUserMessages   int            `json:"total_user_messages"`  // 总用户消息数
	TimeDistribution    map[string]int `json:"time_distribution"`    // 按小时分布
	ProjectDistribution map[string]int `json:"project_distribution"` // 按项目分布
	DateRange           *DateRange     `json:"date_range"`           // 日期范围
	PrimaryLanguage     string         `json:"primary_language"`     // 主要语言
}

// DateRange 日期范围
type DateRange struct {
	Start string `json:"start"` // 开始日期
	End   string `json:"end"`   // 结束日期
}

// ProfileMeta 画像元数据
type ProfileMeta struct {
	LastAnalyzed string `json:"last_analyzed,omitempty"` // 上次分析时间
	DataHash     string `json:"data_hash"`               // 数据哈希
	NeedsUpdate  bool   `json:"needs_update"`            // 是否需要更新
}

// GetMessagesResponse 获取用户消息响应
type GetMessagesResponse struct {
	Messages        *UserMessagesGroup `json:"messages"`                   // 用户消息
	Stats           *ProfileStats      `json:"stats"`                      // 统计信息
	ExistingProfile *string            `json:"existing_profile,omitempty"` // 现有画像
	Meta            *ProfileMeta       `json:"meta"`                       // 元数据
}

// SaveProfileRequest 保存画像请求
type SaveProfileRequest struct {
	Scope       string `json:"scope"`                  // "global" 或 "project"
	ProjectPath string `json:"project_path,omitempty"` // 项目路径
	Content     string `json:"content"`                // 画像内容（Markdown）
	Language    string `json:"language,omitempty"`     // 语言（zh/en）
}

// SaveProfileResponse 保存画像响应
type SaveProfileResponse struct {
	Success    bool   `json:"success"`     // 是否成功
	FilePath   string `json:"file_path"`   // 保存路径
	GitIgnored bool   `json:"git_ignored"` // 是否已添加到 .gitignore
	Message    string `json:"message"`     // 消息
}

// GetMessages 获取用户消息用于画像分析
// @Summary 获取用户消息用于画像分析
// @Tags Profile
// @Accept json
// @Produce json
// @Param body body GetMessagesRequest true "请求参数"
// @Success 200 {object} response.Response{data=GetMessagesResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /profile/messages [post]
func (h *ProfileHandler) GetMessages(c *gin.Context) {
	var req GetMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 800001, "request parameter error: "+err.Error())
		return
	}

	// 设置默认值
	if req.Scope == "" {
		req.Scope = "global"
	}
	if req.DaysBack <= 0 {
		req.DaysBack = 30
	}
	if req.RecentSessions <= 0 {
		req.RecentSessions = 5
	}
	if req.SamplingRate <= 0 {
		req.SamplingRate = 0.2
	}
	if req.MaxHistoricalMsgs <= 0 {
		req.MaxHistoricalMsgs = 100
	}

	// 验证参数
	if req.Scope == "project" && req.ProjectPath == "" {
		response.Error(c, http.StatusBadRequest, 800002, "project_path is required when scope is 'project'")
		return
	}

	// 计算时间范围
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -req.DaysBack)
	startTimestamp := startDate.UnixMilli()
	endTimestamp := endDate.UnixMilli()

	// 获取所有项目
	projects := h.projectManager.ListAllProjects()

	// 如果是项目级别，过滤到指定项目
	if req.Scope == "project" {
		var filteredProjects []*domainCursor.ProjectInfo
		for _, p := range projects {
			for _, ws := range p.Workspaces {
				if ws.Path == req.ProjectPath {
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

	// 提取用户消息
	var recentMessages []*UserMessage
	var historicalMessages []*UserMessage
	timeDistribution := make(map[string]int)
	projectDistribution := make(map[string]int)
	var hashData []string
	var allTextForLangDetect strings.Builder

	for i, swp := range allSessions {
		// 获取会话内容
		options := &appCursor.TextContentOptions{
			FilterLogsAndCode: true,
			MaxMessageLength:  5000,
		}
		messages, err := h.sessionService.GetSessionTextContentWithOptions(swp.Session.ComposerID, options)
		if err != nil {
			continue
		}

		// 只提取用户消息
		for _, msg := range messages {
			if msg.Type != domainCursor.MessageTypeUser {
				continue
			}

			text := strings.TrimSpace(msg.Text)
			if text == "" {
				continue
			}

			// 截断超长消息
			if len(text) > maxUserMessageContentLength {
				text = text[:maxUserMessageContentLength] + "..."
			}

			userMsg := &UserMessage{
				Content:     text,
				Timestamp:   msg.Timestamp,
				Project:     swp.ProjectName,
				SessionName: swp.Session.Name,
			}

			// 累积文本用于语言检测
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

			if i < req.RecentSessions {
				recentMessages = append(recentMessages, userMsg)
			} else {
				if rand.Float64() < req.SamplingRate && len(historicalMessages) < req.MaxHistoricalMsgs {
					historicalMessages = append(historicalMessages, userMsg)
				}
			}
		}
	}

	// 计算数据指纹
	sort.Strings(hashData)
	hashInput := strings.Join(hashData, "|")
	hash := sha256.Sum256([]byte(hashInput))
	dataHash := hex.EncodeToString(hash[:16])

	// 检测主要语言
	chineseRatio := detectChineseRatio(allTextForLangDetect.String())
	primaryLanguage := "en"
	if chineseRatio > 0.3 {
		primaryLanguage = "zh"
	}

	// 读取现有 Profile
	var existingProfile *string
	var lastAnalyzed string
	needsUpdate := true

	profilePath, metaPath := getProfilePaths(req.Scope, req.ProjectPath)
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

	resp := GetMessagesResponse{
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

	response.Success(c, resp)
}

// Save 保存用户画像
// @Summary 保存用户画像
// @Tags Profile
// @Accept json
// @Produce json
// @Param body body SaveProfileRequest true "请求参数"
// @Success 200 {object} response.Response{data=SaveProfileResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /profile [post]
func (h *ProfileHandler) Save(c *gin.Context) {
	var req SaveProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 800001, "request parameter error: "+err.Error())
		return
	}

	// 验证参数
	if req.Scope == "" {
		req.Scope = "global"
	}
	if req.Scope == "project" && req.ProjectPath == "" {
		response.Error(c, http.StatusBadRequest, 800002, "project_path is required when scope is 'project'")
		return
	}
	if req.Content == "" {
		response.Error(c, http.StatusBadRequest, 800003, "content is required")
		return
	}

	profilePath, metaPath := getProfilePaths(req.Scope, req.ProjectPath)

	// 确保目录存在
	profileDir := filepath.Dir(profilePath)
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		response.Error(c, http.StatusInternalServerError, 800004, "failed to create directory: "+err.Error())
		return
	}

	// 准备内容
	var finalContent string
	if req.Scope == "project" {
		// 项目级：添加 YAML frontmatter
		description := "User profile containing coding style, technical preferences, communication habits, and work patterns. This rule helps AI understand the user better and provide more personalized responses."
		if req.Language == "zh" {
			description = "用户画像规则，包含编码风格、技术偏好、沟通习惯和工作模式，帮助 AI 更好地理解用户并提供个性化回复。"
		}
		finalContent = fmt.Sprintf(`---
description: %s
alwaysApply: true
---

%s`, description, req.Content)
	} else {
		finalContent = req.Content
	}

	// 写入 Profile 文件
	if err := os.WriteFile(profilePath, []byte(finalContent), 0644); err != nil {
		response.Error(c, http.StatusInternalServerError, 800005, "failed to write profile: "+err.Error())
		return
	}

	// 更新元数据
	meta := ProfileMeta{
		LastAnalyzed: time.Now().Format(time.RFC3339),
		DataHash:     "",
		NeedsUpdate:  false,
	}
	metaContent, _ := json.MarshalIndent(meta, "", "  ")
	os.WriteFile(metaPath, metaContent, 0644)

	// 项目级：更新 .gitignore
	gitIgnored := false
	if req.Scope == "project" {
		gitIgnored = updateGitIgnore(req.ProjectPath, ".cursor/rules/user-profile.mdc")
	}

	response.Success(c, SaveProfileResponse{
		Success:    true,
		FilePath:   profilePath,
		GitIgnored: gitIgnored,
		Message:    "Profile saved successfully",
	})
}

// detectChineseRatio 检测文本中中文字符的比例
func detectChineseRatio(text string) float64 {
	if len(text) == 0 {
		return 0
	}

	chineseCount := 0
	totalCount := 0

	for _, r := range text {
		if r <= 127 {
			continue
		}
		totalCount++
		if r >= 0x4E00 && r <= 0x9FFF {
			chineseCount++
		}
	}

	if totalCount == 0 {
		return 0
	}
	return float64(chineseCount) / float64(totalCount)
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

	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return false
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return true
		}
	}

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
