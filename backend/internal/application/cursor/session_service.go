package cursor

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/cocursor/backend/internal/infrastructure/storage"
)

// SessionService 会话查询服务
type SessionService struct {
	projectManager *ProjectManager
	pathResolver   *infraCursor.PathResolver
	dbReader       *infraCursor.DBReader
	sessionRepo    storage.WorkspaceSessionRepository
}

// NewSessionService 创建会话查询服务实例（接受 Repository 作为参数）
func NewSessionService(
	projectManager *ProjectManager,
	sessionRepo storage.WorkspaceSessionRepository,
) *SessionService {
	return &SessionService{
		projectManager: projectManager,
		pathResolver:   infraCursor.NewPathResolver(),
		dbReader:       infraCursor.NewDBReader(),
		sessionRepo:    sessionRepo,
	}
}

// GetSessionList 获取会话列表（分页）
// projectName: 项目名称（可选）
// limit: 每页条数
// offset: 偏移量
// search: 搜索关键词（会话名称）
// 返回: 会话列表、总数、是否有更多
func (s *SessionService) GetSessionList(projectName string, limit, offset int, search string) ([]*domainCursor.ComposerData, int, bool, error) {
	// 验证参数
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// 获取工作区列表
	var workspaceIDs []string
	if projectName != "" {
		project := s.projectManager.GetProject(projectName)
		if project == nil {
			return nil, 0, false, fmt.Errorf("项目不存在: %s", projectName)
		}
		for _, ws := range project.Workspaces {
			workspaceIDs = append(workspaceIDs, ws.WorkspaceID)
		}
	} else {
		// 跨项目：获取所有工作区
		projects := s.projectManager.ListAllProjects()
		for _, project := range projects {
			for _, ws := range project.Workspaces {
				workspaceIDs = append(workspaceIDs, ws.WorkspaceID)
			}
		}
	}

	// 从缓存表查询会话数据
	sessions, total, err := s.sessionRepo.FindByWorkspaces(workspaceIDs, search, limit, offset)
	if err != nil {
		return nil, 0, false, fmt.Errorf("failed to query sessions from cache: %w", err)
	}

	// 转换为 ComposerData
	result := make([]*domainCursor.ComposerData, len(sessions))
	for i, session := range sessions {
		composer := s.sessionToComposerData(session)
		result[i] = &composer
	}

	hasMore := (offset + limit) < total

	return result, total, hasMore, nil
}

// GetSessionDetail 获取会话详情（完整对话）
// sessionID: 会话 ID（ComposerID）
// limit: 限制消息数量（默认 100）
// 返回: SessionDetail 和错误
func (s *SessionService) GetSessionDetail(sessionID string, limit int) (*domainCursor.SessionDetail, error) {
	if limit < 1 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	log.Printf("[GetSessionDetail] 开始查找会话: %s", sessionID)

	// 1. 直接扫描 Cursor projects 目录下的所有项目目录，查找 transcript 文件
	// Windows: %USERPROFILE%\.cursor\projects
	// macOS/Linux: ~/.cursor/projects
	projectsDir := s.pathResolver.GetCursorProjectsDirOrDefault()
	if projectsDir == "" {
		return nil, fmt.Errorf("无法获取 Cursor projects 目录")
	}

	projectEntries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, fmt.Errorf("无法读取项目目录: %w", err)
	}

	transcriptFileName := sessionID + ".txt"
	var transcriptPath string
	var projectKey string

	// 遍历所有项目目录，查找 transcript 文件
	for _, entry := range projectEntries {
		if !entry.IsDir() {
			continue
		}

		projectKey = entry.Name()
		transcriptDir := filepath.Join(projectsDir, projectKey, "agent-transcripts")
		candidatePath := filepath.Join(transcriptDir, transcriptFileName)

		if _, err := os.Stat(candidatePath); err == nil {
			transcriptPath = candidatePath
			log.Printf("[GetSessionDetail] 在项目 %s 找到 transcript 文件: %s", projectKey, transcriptPath)
			break
		}
	}

	if transcriptPath == "" {
		return nil, fmt.Errorf("会话不存在: %s (未找到 transcript 文件)", sessionID)
	}

	// 2. 先获取会话元数据（用于时间戳基准）
	session, err := s.findSessionMetadata(sessionID, projectKey)
	var baseTimestamp int64
	var sessionCreatedAt int64
	var sessionLastUpdatedAt int64

	if err == nil && session != nil {
		baseTimestamp = session.CreatedAt
		sessionCreatedAt = session.CreatedAt
		sessionLastUpdatedAt = session.LastUpdatedAt
	} else {
		log.Printf("[GetSessionDetail] 无法获取会话元数据: %v，使用 transcript 文件时间", err)
		// 获取 transcript 文件的修改时间作为基准
		fileInfo, err := os.Stat(transcriptPath)
		if err == nil {
			baseTimestamp = fileInfo.ModTime().UnixMilli()
			sessionCreatedAt = baseTimestamp
			sessionLastUpdatedAt = baseTimestamp
		} else {
			// 如果连文件信息都获取不到，使用当前时间
			baseTimestamp = time.Now().UnixMilli()
			sessionCreatedAt = baseTimestamp
			sessionLastUpdatedAt = baseTimestamp
		}
		// 如果无法获取元数据，创建一个基本的会话对象
		session = &domainCursor.ComposerData{
			ComposerID:    sessionID,
			Name:          "会话 " + sessionID[:8],
			CreatedAt:     sessionCreatedAt,
			LastUpdatedAt: sessionLastUpdatedAt,
		}
	}

	// 3. 加载 transcript 消息
	transcriptContent, err := os.ReadFile(transcriptPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取 transcript 文件: %w", err)
	}

	transcriptMessages, err := s.parseTranscript(string(transcriptContent), baseTimestamp)
	if err != nil {
		return nil, fmt.Errorf("无法解析 transcript 文件: %w", err)
	}

	log.Printf("[GetSessionDetail] 从 transcript 加载了 %d 条消息", len(transcriptMessages))

	// 如果无法获取元数据，从消息时间戳推断会话时间范围
	if session.CreatedAt == 0 || session.LastUpdatedAt == 0 {
		if len(transcriptMessages) > 0 {
			// 从消息时间戳推断
			firstMsgTime := transcriptMessages[0].Timestamp
			lastMsgTime := transcriptMessages[len(transcriptMessages)-1].Timestamp
			session.CreatedAt = firstMsgTime
			session.LastUpdatedAt = lastMsgTime
		}
	}

	// 应用 limit（取最后 N 条消息）
	totalMessages := len(transcriptMessages)
	if totalMessages > limit {
		transcriptMessages = transcriptMessages[totalMessages-limit:]
	}

	return &domainCursor.SessionDetail{
		Session:       session,
		Messages:      transcriptMessages,
		TotalMessages: totalMessages,
		HasMore:       totalMessages > limit,
	}, nil
}

// findSessionMetadata 通过 sessionID 和 projectKey 查找会话元数据
func (s *SessionService) findSessionMetadata(sessionID, projectKey string) (*domainCursor.ComposerData, error) {
	// 通过 projectKey 找到对应的工作区
	// projectKey 格式: d-code-cocursor，需要转换为项目路径
	// 由于 projectKey 是路径转换来的，我们需要遍历工作区来匹配

	workspaceDir, err := s.pathResolver.GetWorkspaceStorageDir()
	if err != nil {
		return nil, fmt.Errorf("无法获取工作区目录: %w", err)
	}

	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("无法读取工作区目录: %w", err)
	}

	// 遍历所有工作区，查找匹配的项目
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workspaceID := entry.Name()
		workspaceJSONPath := filepath.Join(workspaceDir, workspaceID, "workspace.json")

		// 读取 workspace.json 获取项目路径
		workspaceJSON, err := os.ReadFile(workspaceJSONPath)
		if err != nil {
			continue
		}

		var workspaceConfig struct {
			Folder string `json:"folder"`
		}
		if err := json.Unmarshal(workspaceJSON, &workspaceConfig); err != nil {
			continue
		}

		// 解析项目路径并生成 projectKey
		// 使用 url.Parse 解析 file:// URI
		parsedURL, err := url.Parse(workspaceConfig.Folder)
		if err != nil {
			continue
		}
		if parsedURL.Scheme != "file" {
			continue
		}

		// 获取路径并处理 Windows 路径
		projectPath := parsedURL.Path
		if decoded, err := url.PathUnescape(projectPath); err == nil {
			projectPath = decoded
		}
		// Windows 路径: /d:/code/cocursor -> d:/code/cocursor
		if len(projectPath) > 2 && projectPath[1] == ':' && projectPath[0] == '/' {
			projectPath = projectPath[1:]
		}
		projectPath = filepath.FromSlash(projectPath)

		// 生成 projectKey 进行比较
		computedKey := s.generateProjectKey(projectPath)

		if computedKey != projectKey {
			continue
		}

		// 找到匹配的工作区，读取 composer 数据
		workspaceDBPath := filepath.Join(workspaceDir, workspaceID, "state.vscdb")
		composerDataValue, err := s.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
		if err != nil {
			continue
		}

		composers, err := domainCursor.ParseComposerData(string(composerDataValue))
		if err != nil {
			continue
		}

		// 查找会话
		for i := range composers {
			if composers[i].ComposerID == sessionID {
				return &composers[i], nil
			}
		}
	}

	return nil, fmt.Errorf("未找到会话元数据")
}

// generateProjectKey 从项目路径生成 projectKey
// 格式: d:\code\cocursor -> d-code-cocursor
func (s *SessionService) generateProjectKey(projectPath string) string {
	key := strings.ReplaceAll(projectPath, ":", "")
	key = strings.ReplaceAll(key, string(filepath.Separator), "-")
	return strings.ToLower(key)
}

// combineMessages 组合 prompts 和 generations 为消息列表
// 注意：由于 prompts 和 generations 没有直接的关联字段，我们按时间顺序组合
// 假设 prompts 和 generations 是交替出现的（用户输入 → AI 回复）
func (s *SessionService) combineMessages(
	prompts []map[string]interface{},
	generations []domainCursor.GenerationData,
	session *domainCursor.ComposerData,
	limit int,
) []*domainCursor.Message {
	var messages []*domainCursor.Message

	// 创建时间范围（会话创建时间到更新时间）
	sessionStart := session.CreatedAt
	sessionEnd := session.LastUpdatedAt
	sessionDuration := sessionEnd - sessionStart

	// 用于去重的 map：
	// 1. 基于时间窗口和内容：key = "type:timestamp:text_hash"，value = message
	// 2. 基于完整内容：key = "type:full_text"，用于检测完全相同的消息
	dedupMap := make(map[string]*domainCursor.Message)
	contentDedupMap := make(map[string]bool) // 用于检测完全相同的消息内容
	const timeWindow = int64(5000)           // 5秒时间窗口

	// 辅助函数：生成去重 key（基于时间窗口和内容片段）
	getDedupKey := func(msgType domainCursor.MessageType, timestamp int64, text string) string {
		// 使用时间窗口（5秒）来分组相似时间的消息
		timeBucket := timestamp / timeWindow
		// 使用文本的前100个字符作为内容标识（避免过长）
		textHash := text
		if len(textHash) > 100 {
			textHash = textHash[:100]
		}
		return fmt.Sprintf("%s:%d:%s", msgType, timeBucket, textHash)
	}

	// 辅助函数：生成完整内容去重 key
	getContentKey := func(msgType domainCursor.MessageType, text string) string {
		return fmt.Sprintf("%s:%s", msgType, text)
	}

	// 收集所有 prompts（用户消息）
	// 先按 generations 的时间戳分布来估算 prompts 的时间戳
	// 只使用 Composer 模式的 generations 来估算
	genTimestamps := make([]int64, 0, len(generations))
	for _, gen := range generations {
		// 只使用 Composer 模式的 generations
		if gen.Type == "composer" && gen.UnixMs >= sessionStart && gen.UnixMs <= sessionEnd {
			genTimestamps = append(genTimestamps, gen.UnixMs)
		}
	}
	sort.Slice(genTimestamps, func(i, j int) bool {
		return genTimestamps[i] < genTimestamps[j]
	})

	promptIndex := 0
	for i, prompt := range prompts {
		text, ok := prompt["text"].(string)
		if !ok || text == "" {
			continue
		}

		// 只包含 Composer 模式的 prompts（commandType 4）
		// 因为 prompts 是工作区级别的数据，包含所有类型（tab/chat/composer）
		// 我们需要只显示 Composer 会话的对话
		commandType, ok := prompt["commandType"].(float64)
		if !ok || int(commandType) != 4 {
			continue
		}

		// 改进时间戳估算：
		// 1. 如果有 generations，尝试在它们之间分配 prompts
		// 2. 否则，按会话时长均匀分配
		var estimatedTime int64
		if len(genTimestamps) > 0 && promptIndex < len(genTimestamps) {
			// 在对应的 generation 之前 10 秒
			estimatedTime = genTimestamps[promptIndex] - 10000
			if estimatedTime < sessionStart {
				estimatedTime = sessionStart + int64(promptIndex)*1000*30 // 30秒间隔
			}
		} else {
			// 均匀分布在会话时长内
			if len(prompts) > 1 {
				estimatedTime = sessionStart + int64(i)*sessionDuration/int64(len(prompts))
			} else {
				estimatedTime = sessionStart
			}
		}

		// 只包含会话时间范围内的消息
		if estimatedTime < sessionStart || estimatedTime > sessionEnd {
			continue
		}

		// 去重检查：先检查完整内容是否重复
		contentKey := getContentKey(domainCursor.MessageTypeUser, text)
		if contentDedupMap[contentKey] {
			log.Printf("[combineMessages] 跳过完全重复的用户消息 (时间戳: %d)", estimatedTime)
			continue
		}

		// 再检查时间窗口内的重复
		dedupKey := getDedupKey(domainCursor.MessageTypeUser, estimatedTime, text)
		if _, exists := dedupMap[dedupKey]; exists {
			log.Printf("[combineMessages] 跳过时间窗口内重复的用户消息: %s (时间戳: %d)", text[:min(50, len(text))], estimatedTime)
			continue
		}

		// 过滤内部标签
		text = s.filterMessageText(text)

		msg := &domainCursor.Message{
			Type:      domainCursor.MessageTypeUser,
			Text:      text,
			Timestamp: estimatedTime,
		}

		// 提取代码块
		msg.CodeBlocks = s.extractCodeBlocks(text)
		messages = append(messages, msg)
		dedupMap[dedupKey] = msg
		contentDedupMap[contentKey] = true
		promptIndex++
	}

	// 收集所有 generations（AI 消息）
	for _, gen := range generations {
		// 只包含 Composer 模式的 generations（type == "composer"）
		// 因为 prompts 和 generations 是工作区级别的数据，包含所有类型（tab/chat/composer）
		// 我们需要只显示 Composer 会话的对话
		if gen.Type != "composer" {
			continue
		}

		// 只包含会话时间范围内的消息
		if gen.UnixMs < sessionStart || gen.UnixMs > sessionEnd {
			continue
		}

		text := gen.TextDescription
		if text == "" {
			continue
		}

		// 去重检查：先检查完整内容是否重复
		contentKey := getContentKey(domainCursor.MessageTypeAI, text)
		if contentDedupMap[contentKey] {
			log.Printf("[combineMessages] 跳过完全重复的 AI 消息 (时间戳: %d, UUID: %s)", gen.UnixMs, gen.GenerationUUID)
			continue
		}

		// 再检查时间窗口内的重复
		dedupKey := getDedupKey(domainCursor.MessageTypeAI, gen.UnixMs, text)
		if existing, exists := dedupMap[dedupKey]; exists {
			// 如果已存在且内容相同，跳过
			if existing.Text == text {
				log.Printf("[combineMessages] 跳过时间窗口内重复的 AI 消息 (时间戳: %d, UUID: %s)", gen.UnixMs, gen.GenerationUUID)
				continue
			}
		}

		// 过滤内部标签
		text = s.filterMessageText(text)

		msg := &domainCursor.Message{
			Type:      domainCursor.MessageTypeAI,
			Text:      text,
			Timestamp: gen.UnixMs,
		}

		// 提取代码块
		msg.CodeBlocks = s.extractCodeBlocks(text)
		messages = append(messages, msg)
		dedupMap[dedupKey] = msg
		contentDedupMap[contentKey] = true
	}

	// 按时间排序
	sort.Slice(messages, func(i, j int) bool {
		// 如果时间戳相同，用户消息优先
		if messages[i].Timestamp == messages[j].Timestamp {
			return messages[i].Type == domainCursor.MessageTypeUser
		}
		return messages[i].Timestamp < messages[j].Timestamp
	})

	// 限制数量
	if len(messages) > limit {
		messages = messages[:limit]
	}

	// 添加文件引用（从 session.Subtitle）
	if session.Subtitle != "" {
		files := s.parseFileList(session.Subtitle)
		// 将文件引用添加到最后一条消息（或所有消息？）
		if len(messages) > 0 {
			messages[len(messages)-1].Files = files
		}
	}

	log.Printf("[combineMessages] 组合完成: %d 条消息 (prompts: %d, generations: %d, 去重后: %d)",
		len(messages), len(prompts), len(generations), len(messages))

	return messages
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractCodeBlocks 从文本中提取代码块
// 支持 Markdown 格式：```language\ncode\n```
func (s *SessionService) extractCodeBlocks(text string) []*domainCursor.CodeBlock {
	var blocks []*domainCursor.CodeBlock

	// 查找代码块（Markdown 格式）
	lines := strings.Split(text, "\n")
	inCodeBlock := false
	currentLanguage := ""
	currentCode := []string{}

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				// 结束代码块
				if len(currentCode) > 0 {
					blocks = append(blocks, &domainCursor.CodeBlock{
						Language: currentLanguage,
						Code:     strings.Join(currentCode, "\n"),
					})
				}
				currentCode = []string{}
				currentLanguage = ""
				inCodeBlock = false
			} else {
				// 开始代码块
				language := strings.TrimPrefix(line, "```")
				language = strings.TrimSpace(language)
				currentLanguage = language
				if currentLanguage == "" {
					currentLanguage = "text"
				}
				inCodeBlock = true
			}
		} else if inCodeBlock {
			currentCode = append(currentCode, line)
		}
	}

	// 处理未闭合的代码块
	if inCodeBlock && len(currentCode) > 0 {
		blocks = append(blocks, &domainCursor.CodeBlock{
			Language: currentLanguage,
			Code:     strings.Join(currentCode, "\n"),
		})
	}

	return blocks
}

// parseTranscript 解析 agent-transcripts 文件内容
// baseTimestamp: 会话创建时间戳（毫秒），用于计算消息的相对时间
func (s *SessionService) parseTranscript(content string, baseTimestamp int64) ([]*domainCursor.Message, error) {
	var messages []*domainCursor.Message
	lines := strings.Split(content, "\n")

	var currentRole string
	var currentText strings.Builder
	var currentTools []*domainCursor.ToolCall
	var messageIndex int

	// 保存当前消息到列表
	saveMessage := func() {
		if currentRole == "" || currentText.Len() == 0 {
			return
		}

		text := strings.TrimSpace(currentText.String())
		if text == "" {
			return
		}

		text = s.filterMessageText(text)
		if text == "" {
			return
		}

		timestamp := baseTimestamp + int64(messageIndex*1000) // 每条消息间隔1秒
		msg := &domainCursor.Message{
			Type:      getMessageType(currentRole),
			Text:      text,
			Timestamp: timestamp,
		}

		if msg.Type == domainCursor.MessageTypeAI {
			msg.CodeBlocks = s.extractCodeBlocks(text)
			// 将工具调用附加到AI消息
			if len(currentTools) > 0 {
				msg.Tools = make([]*domainCursor.ToolCall, len(currentTools))
				copy(msg.Tools, currentTools)
			}
		}

		messages = append(messages, msg)
		messageIndex++
		currentTools = nil // 重置工具调用列表
	}

	// 跳过标签内容（如 <think>...</think>）
	skipTagContent := func(lines []string, i int, openTag, closeTag string) int {
		for j := i + 1; j < len(lines); j++ {
			if strings.Contains(lines[j], closeTag) {
				return j
			}
		}
		return len(lines) - 1
	}

	// 提取标签内容（如 <user_query>...</user_query>）
	extractTagContent := func(lines []string, i int, openTag, closeTag string) (string, int) {
		line := lines[i]
		startIdx := strings.Index(line, openTag)
		if startIdx == -1 {
			return "", i
		}

		contentStart := startIdx + len(openTag)
		endIdx := strings.Index(line[contentStart:], closeTag)

		if endIdx != -1 {
			// 在同一行
			return strings.TrimSpace(line[contentStart : contentStart+endIdx]), i
		}

		// 跨行提取
		var content strings.Builder
		content.WriteString(line[contentStart:])
		for j := i + 1; j < len(lines); j++ {
			if strings.Contains(lines[j], closeTag) {
				content.WriteString(strings.TrimSuffix(lines[j], closeTag))
				return strings.TrimSpace(content.String()), j
			}
			if j > i+1 {
				content.WriteString("\n")
			}
			content.WriteString(lines[j])
		}
		return strings.TrimSpace(content.String()), len(lines) - 1
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// 检测角色标记
		if line == "user:" {
			saveMessage()
			currentRole = "user"
			currentText.Reset()
			currentTools = nil
			continue
		}

		if line == "assistant:" {
			saveMessage()
			currentRole = "assistant"
			currentText.Reset()
			currentTools = nil
			continue
		}

		// 提取工具调用信息（只在assistant消息中）
		if strings.HasPrefix(line, "[Tool call]") && currentRole == "assistant" {
			toolName := strings.TrimSpace(strings.TrimPrefix(line, "[Tool call]"))
			if toolName == "" {
				continue
			}

			// 解析工具参数（从下一行开始，直到空行或下一个标记）
			args := make(map[string]string)
			var currentArgKey string
			var currentArgValue strings.Builder

			for j := i + 1; j < len(lines); j++ {
				nextLine := lines[j]
				// 遇到空行、下一个工具调用、工具结果或消息开始，停止
				if nextLine == "" || strings.HasPrefix(nextLine, "[Tool call]") ||
					strings.HasPrefix(nextLine, "[Tool result]") ||
					nextLine == "user:" || nextLine == "assistant:" {
					// 保存最后一个参数
					if currentArgKey != "" {
						args[currentArgKey] = strings.TrimSpace(currentArgValue.String())
					}
					i = j - 1
					break
				}

				// 检查是否是新的参数（以非空格开头，且包含冒号）
				trimmed := strings.TrimSpace(nextLine)
				if strings.Contains(trimmed, ":") && !strings.HasPrefix(nextLine, " ") && !strings.HasPrefix(nextLine, "\t") {
					// 保存上一个参数
					if currentArgKey != "" {
						args[currentArgKey] = strings.TrimSpace(currentArgValue.String())
						currentArgValue.Reset()
					}
					// 解析新参数
					parts := strings.SplitN(trimmed, ":", 2)
					if len(parts) == 2 {
						currentArgKey = strings.TrimSpace(parts[0])
						currentArgValue.WriteString(strings.TrimSpace(parts[1]))
					}
				} else if currentArgKey != "" {
					// 继续累积当前参数的值（可能是多行）
					if currentArgValue.Len() > 0 {
						currentArgValue.WriteString(" ")
					}
					currentArgValue.WriteString(trimmed)
				} else if strings.Contains(trimmed, "=") {
					// 尝试解析 key=value 格式
					parts := strings.SplitN(trimmed, "=", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						value := strings.TrimSpace(parts[1])
						if key != "" {
							args[key] = value
						}
					}
				}

				if j == len(lines)-1 {
					// 保存最后一个参数
					if currentArgKey != "" {
						args[currentArgKey] = strings.TrimSpace(currentArgValue.String())
					}
					i = j
				}
			}

			// 创建工具调用对象
			toolCall := &domainCursor.ToolCall{
				Name:      toolName,
				Arguments: args,
			}
			currentTools = append(currentTools, toolCall)
			continue
		}

		// 跳过工具结果（不解析结果内容，只跳过）
		if strings.HasPrefix(line, "[Tool result]") {
			for j := i + 1; j < len(lines); j++ {
				nextLine := lines[j]
				if nextLine == "" || strings.HasPrefix(nextLine, "[Tool call]") ||
					strings.HasPrefix(nextLine, "[Tool result]") ||
					nextLine == "user:" || nextLine == "assistant:" {
					i = j - 1
					break
				}
				if j == len(lines)-1 {
					i = j
				}
			}
			continue
		}

		// 提取 user_query 内容
		if strings.Contains(line, "<user_query>") {
			content, newI := extractTagContent(lines, i, "<user_query>", "</user_query>")
			if content != "" {
				currentText.WriteString(content)
			}
			i = newI
			continue
		}

		// 跳过 <think> 标签及其内容
		if strings.Contains(line, "<think>") {
			i = skipTagContent(lines, i, "<think>", "</think>")
			continue
		}

		// 累积当前消息的文本
		if currentRole != "" {
			if currentText.Len() > 0 {
				currentText.WriteString("\n")
			}
			currentText.WriteString(line)
		}
	}

	// 保存最后一条消息
	saveMessage()

	log.Printf("[parseTranscript] 解析了 %d 条消息", len(messages))
	return messages, nil
}

// getMessageType 将角色转换为消息类型
func getMessageType(role string) domainCursor.MessageType {
	switch role {
	case "user":
		return domainCursor.MessageTypeUser
	case "assistant":
		return domainCursor.MessageTypeAI
	default:
		return domainCursor.MessageTypeUser
	}
}

// filterMessageText 过滤消息中的内部标签，只保留用户可见的内容
func (s *SessionService) filterMessageText(text string) string {
	if text == "" {
		return text
	}

	// 使用正则表达式移除 <think>...</think> 标签及其内容（AI 的内部思考过程）
	// 匹配 <think> 和 </think> 之间的所有内容，包括换行符
	thinkRegex := regexp.MustCompile(`(?i)<think>[\s\S]*?</think>`)
	text = thinkRegex.ReplaceAllString(text, "")

	// 移除 <user_query>...</user_query> 标签，但保留内容
	userQueryRegex := regexp.MustCompile(`(?i)</?user_query>`)
	text = userQueryRegex.ReplaceAllString(text, "")

	// 移除 [Tool call] 和 [Tool result] 标记及其后续内容（直到下一个空行或消息开始）
	toolCallRegex := regexp.MustCompile(`(?i)\[Tool call\][^\n]*\n[^\n]*\n`)
	text = toolCallRegex.ReplaceAllString(text, "")
	toolResultRegex := regexp.MustCompile(`(?i)\[Tool result\][^\n]*\n`)
	text = toolResultRegex.ReplaceAllString(text, "")

	// 清理多余的空行（连续3个或更多空行替换为2个）
	lines := strings.Split(text, "\n")
	var filteredLines []string
	emptyCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			emptyCount++
			if emptyCount <= 2 {
				filteredLines = append(filteredLines, line)
			}
		} else {
			emptyCount = 0
			filteredLines = append(filteredLines, line)
		}
	}
	text = strings.Join(filteredLines, "\n")

	// 去除首尾空白
	text = strings.TrimSpace(text)

	return text
}

// parseFileList 解析文件列表（逗号分隔）
func (s *SessionService) parseFileList(subtitle string) []string {
	var files []string
	parts := strings.Split(subtitle, ",")
	for _, part := range parts {
		file := strings.TrimSpace(part)
		if file != "" {
			files = append(files, file)
		}
	}
	return files
}

// TextContentOptions 文本内容过滤选项
type TextContentOptions struct {
	// FilterLogsAndCode 是否过滤日志和代码相关内容（只保留自然语言文本）
	// 启用后会将日志行、代码行、文件路径等过滤掉，只保留自然语言对话
	// 默认 false，保持向后兼容
	FilterLogsAndCode bool
	
	// MaxMessageLength 最大消息长度，超过此长度会被截断，0 表示不限制
	// 默认 5000
	MaxMessageLength int
}

// DefaultTextContentOptions 返回默认的文本内容选项
func DefaultTextContentOptions() *TextContentOptions {
	return &TextContentOptions{
		FilterLogsAndCode: false,
		MaxMessageLength:  5000,
	}
}

// GetSessionTextContent 获取会话的纯文本内容（过滤 tool 和代码块）
// 返回用户和 AI 的文本消息，去除所有 tool 调用和代码块
// 使用默认选项（不启用深度过滤）
func (s *SessionService) GetSessionTextContent(sessionID string) ([]*domainCursor.Message, error) {
	return s.GetSessionTextContentWithOptions(sessionID, DefaultTextContentOptions())
}

// GetSessionTextContentWithOptions 获取会话的纯文本内容（带过滤选项）
// options 为 nil 时使用默认选项
func (s *SessionService) GetSessionTextContentWithOptions(sessionID string, options *TextContentOptions) ([]*domainCursor.Message, error) {
	// 使用默认选项
	if options == nil {
		options = DefaultTextContentOptions()
	}
	
	// 获取会话详情
	sessionDetail, err := s.GetSessionDetail(sessionID, 1000)
	if err != nil {
		return nil, err
	}

	// 过滤消息：只保留文本，去除 tool 和代码块
	var textMessages []*domainCursor.Message
	for _, msg := range sessionDetail.Messages {
		// 跳过有 tool 调用的消息
		if len(msg.Tools) > 0 {
			continue
		}

		// 去除代码块，只保留文本
		text := msg.Text
		// 移除代码块（Markdown 格式：```language\ncode\n```）
		text = s.removeCodeBlocksFromText(text)
		
		// 过滤系统信息（如 git_status）
		text = s.filterSystemInfo(text)
		
		// 根据选项决定是否过滤日志和代码相关内容
		if options.FilterLogsAndCode {
			text = s.filterLogsAndCode(text)
		}

		// 如果文本为空，跳过
		if strings.TrimSpace(text) == "" {
			continue
		}

		// 限制消息长度
		if options.MaxMessageLength > 0 && len(text) > options.MaxMessageLength {
			text = text[:options.MaxMessageLength] + "\n... (消息已截断)"
		}

		// 创建新的消息对象（不包含代码块和工具）
		textMsg := &domainCursor.Message{
			Type:      msg.Type,
			Text:      text,
			Timestamp: msg.Timestamp,
			// 不包含 CodeBlocks, Tools, Files
		}

		textMessages = append(textMessages, textMsg)
	}

	return textMessages, nil
}

// removeCodeBlocksFromText 从文本中移除代码块
// 使用正则表达式更可靠地移除所有代码块（包括行内和跨行的）
func (s *SessionService) removeCodeBlocksFromText(text string) string {
	if text == "" {
		return text
	}
	
	// 使用正则表达式移除所有代码块（包括语言标识）
	// 匹配 ```language 或 ``` 开头，到下一个 ``` 结束的所有内容
	codeBlockRegex := regexp.MustCompile("(?s)```[\\w]*\\n.*?```")
	text = codeBlockRegex.ReplaceAllString(text, "")
	
	// 清理可能残留的代码块标记
	text = strings.ReplaceAll(text, "```", "")
	
	return text
}

// filterSystemInfo 过滤系统信息（如 git_status、文件列表等）
func (s *SessionService) filterSystemInfo(text string) string {
	if text == "" {
		return text
	}
	
	// 移除 <git_status>...</git_status> 标签及其内容
	gitStatusRegex := regexp.MustCompile("(?i)(?s)<git_status>.*?</git_status>")
	text = gitStatusRegex.ReplaceAllString(text, "")
	
	// 移除其他常见的系统信息标签
	systemTags := []string{
		"<file_list>", "</file_list>",
		"<file_contents>", "</file_contents>",
		"<system_info>", "</system_info>",
		"<context>", "</context>",
	}
	
	for i := 0; i < len(systemTags); i += 2 {
		openTag := systemTags[i]
		closeTag := systemTags[i+1]
		pattern := fmt.Sprintf("(?i)(?s)%s.*?%s", regexp.QuoteMeta(openTag), regexp.QuoteMeta(closeTag))
		tagRegex := regexp.MustCompile(pattern)
		text = tagRegex.ReplaceAllString(text, "")
	}
	
	// 清理多余的空行
	lines := strings.Split(text, "\n")
	var filteredLines []string
	emptyCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			emptyCount++
			if emptyCount <= 2 {
				filteredLines = append(filteredLines, line)
			}
		} else {
			emptyCount = 0
			filteredLines = append(filteredLines, line)
		}
	}
	text = strings.Join(filteredLines, "\n")
	
	return strings.TrimSpace(text)
}

// filterLogsAndCode 过滤日志和代码相关内容，只保留自然语言文本
func (s *SessionService) filterLogsAndCode(text string) string {
	if text == "" {
		return text
	}
	
	lines := strings.Split(text, "\n")
	var filteredLines []string
	
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			// 保留空行（用于段落分隔）
			filteredLines = append(filteredLines, "")
			continue
		}
		
		// 跳过日志行（包含时间戳、日志级别等）
		if s.isLogLine(trimmedLine) {
			continue
		}
		
		// 跳过代码行（包含大量代码特征）
		if s.isCodeLine(trimmedLine) {
			continue
		}
		
		// 跳过文件路径行（通常是代码相关）
		if s.isFilePathLine(trimmedLine) {
			continue
		}
		
		// 移除行内代码标记，但保留文本内容
		filteredLine := s.removeInlineCode(trimmedLine)
		
		// 如果过滤后还有内容，保留这一行
		if strings.TrimSpace(filteredLine) != "" {
			filteredLines = append(filteredLines, filteredLine)
		}
	}
	
	// 重新组合文本
	result := strings.Join(filteredLines, "\n")
	
	// 清理多余的空行（连续3个或更多空行替换为2个）
	result = s.cleanupEmptyLines(result)
	
	return strings.TrimSpace(result)
}

// isLogLine 判断是否是日志行
// 使用多种模式识别各种日志格式，包括但不限于：
// - 标准时间戳格式（ISO 8601, RFC 3339等）
// - 各种日志框架格式（logrus, zap, log4j等）
// - 自定义日志格式
func (s *SessionService) isLogLine(line string) bool {
	// 跳过太短的行
	if len(line) < 5 {
		return false
	}
	
	// 匹配常见的日志格式模式
	logPatterns := []*regexp.Regexp{
		// ISO 8601 时间戳 + 日志级别: 2024-01-19T10:30:45.123Z [INFO]
		regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?.*?(INFO|ERROR|WARN|DEBUG|FATAL|TRACE|PANIC)`),
		
		// RFC 3339 时间戳: [2024-01-19 10:30:45] INFO
		regexp.MustCompile(`^\[?\d{4}-\d{2}-\d{2}[\sT]\d{2}:\d{2}:\d{2}(\.\d+)?\]?.*?(INFO|ERROR|WARN|DEBUG|FATAL|TRACE|PANIC)`),
		
		// Unix 时间戳格式: [1234567890] INFO 或 1234567890 INFO
		regexp.MustCompile(`^\[?\d{10,13}\]?.*?(INFO|ERROR|WARN|DEBUG|FATAL|TRACE|PANIC)`),
		
		// 时间格式 HH:MM:SS + 日志级别: 10:30:45 INFO
		regexp.MustCompile(`^[\[\s]*\d{2}:\d{2}:\d{2}(\.\d+)?[\]\s]+.*?(INFO|ERROR|WARN|DEBUG|FATAL|TRACE|PANIC)`),
		
		// 日志级别标记: [LOG], [ERROR], [WARN] 等
		regexp.MustCompile(`^\[(LOG|ERROR|WARN|INFO|DEBUG|FATAL|TRACE|PANIC|FATAL|CRITICAL)\]`),
		
		// 日志级别（小写）: [log], [error], [warn]
		regexp.MustCompile(`^\[(log|error|warn|info|debug|fatal|trace|panic|critical)\]`),
		
		// 包含日志级别且格式像日志（行首有特殊字符）
		regexp.MustCompile(`^[\[\s]*\d{1,2}[:/\-]\d{1,2}[:/\-]\d{2,4}.*?(INFO|ERROR|WARN|DEBUG|FATAL|TRACE|PANIC)`),
		
		// 包含 "log" 关键字且格式像日志（行首有括号或时间）
		regexp.MustCompile(`^[\[\(]?.*?(log|LOG|Log).*?[\]\)]?:`),
		
		// 包含日志级别和文件路径模式: INFO /path/to/file.go:123
		regexp.MustCompile(`^(INFO|ERROR|WARN|DEBUG|FATAL|TRACE|PANIC).*?[/\\].*?\.\w+:\d+`),
		
		// 包含日志级别和包名: INFO github.com/user/repo
		regexp.MustCompile(`^(INFO|ERROR|WARN|DEBUG|FATAL|TRACE|PANIC).*?[\w\-_\.]+/[\w\-_\./]+`),
		
		// 包含日志级别和函数名: INFO funcName()
		regexp.MustCompile(`^(INFO|ERROR|WARN|DEBUG|FATAL|TRACE|PANIC).*?\w+\(\)`),
		
		// 常见的日志框架格式
		// logrus: time="2024-01-19T10:30:45Z" level=info
		regexp.MustCompile(`^\w+=".*?"\s+level=(info|error|warn|debug|fatal|trace|panic)`),
		
		// zap: {"level":"info","ts":1234567890}
		regexp.MustCompile(`^\{.*?"level"\s*:\s*"(info|error|warn|debug|fatal|trace|panic)".*?\}`),
		
		// 包含多个日志特征的行（时间戳 + 级别 + 消息）
		regexp.MustCompile(`^\d{4}.*?\d{2}:\d{2}:\d{2}.*?(INFO|ERROR|WARN|DEBUG|FATAL|TRACE|PANIC).*?[:\-].*?`),
	}
	
	// 检查是否匹配任何日志模式
	for _, pattern := range logPatterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	
	// 额外检查：如果行包含时间戳模式且包含日志相关关键词
	hasTimestamp := regexp.MustCompile(`\d{4}[-/]\d{2}[-/]\d{2}[\sT]\d{2}:\d{2}:\d{2}`).MatchString(line) ||
		regexp.MustCompile(`\d{2}:\d{2}:\d{2}`).MatchString(line)
	hasLogKeywords := regexp.MustCompile(`(?i)(log|error|warn|info|debug|fatal|trace|panic)`).MatchString(line)
	
	if hasTimestamp && hasLogKeywords {
		// 但如果包含大量中文，可能是正常文本
		chineseCount := 0
		for _, char := range line {
			if char >= 0x4e00 && char <= 0x9fff {
				chineseCount++
			}
		}
		// 如果中文字符占比超过30%，可能是正常文本，不是日志
		if float64(chineseCount)/float64(len(line)) < 0.3 {
			return true
		}
	}
	
	return false
}

// isCodeLine 判断是否是代码行
func (s *SessionService) isCodeLine(line string) bool {
	// 跳过太短的行（可能是正常文本）
	if len(line) < 10 {
		return false
	}
	
	// 代码特征：
	// 1. 包含大量特殊符号（{}, (), [], ->, =>, ::, //, /*, */）
	// 2. 包含函数调用 pattern (func(), method(), etc.)
	// 3. 包含变量赋值 pattern (var =, const =, let =, etc.)
	// 4. 包含导入语句 (import, require, include)
	// 5. 包含类型定义 (type, interface, class, struct)
	// 6. 行首有大量空格或制表符（缩进代码）
	
	// 检查特殊符号密度
	specialCharCount := 0
	specialChars := "{}()[];:=-><>"
	for _, char := range line {
		if strings.ContainsRune(specialChars, char) {
			specialCharCount++
		}
	}
	// 如果特殊字符占比超过30%，可能是代码
	if float64(specialCharCount)/float64(len(line)) > 0.3 {
		return true
	}
	
	// 检查代码关键字模式
	codePatterns := []*regexp.Regexp{
		// 函数定义或调用
		regexp.MustCompile(`\b(func|function|def|class|interface|struct|type)\s+\w+`),
		// 变量赋值
		regexp.MustCompile(`^\s*(var|let|const|final)\s+\w+\s*[=:]`),
		// 导入语句
		regexp.MustCompile(`^\s*(import|require|include|from|using)\s+`),
		// 方法调用 pattern
		regexp.MustCompile(`\w+\([^)]*\)`),
		// 类型注解
		regexp.MustCompile(`:\s*\w+(\[\])?(\s|$)`),
		// 指针或引用
		regexp.MustCompile(`[*&]\w+`),
	}
	
	for _, pattern := range codePatterns {
		if pattern.MatchString(line) {
			// 但如果这行包含大量中文或其他自然语言，可能不是纯代码
			chineseCount := 0
			for _, char := range line {
				if char >= 0x4e00 && char <= 0x9fff {
					chineseCount++
				}
			}
			// 如果中文字符占比超过20%，可能是注释或说明，保留
			if float64(chineseCount)/float64(len(line)) < 0.2 {
				return true
			}
		}
	}
	
	// 检查是否是缩进很深的代码（行首有大量空格）
	leadingSpaces := 0
	for _, char := range line {
		if char == ' ' {
			leadingSpaces++
		} else if char == '\t' {
			leadingSpaces += 4 // 制表符算4个空格
		} else {
			break
		}
	}
	// 如果行首有超过8个空格，且包含代码特征，可能是代码
	if leadingSpaces > 8 && specialCharCount > 3 {
		return true
	}
	
	return false
}

// isFilePathLine 判断是否是文件路径行
func (s *SessionService) isFilePathLine(line string) bool {
	// 匹配文件路径模式：
	// 1. 绝对路径：/path/to/file 或 C:\path\to\file
	// 2. 相对路径：./path/to/file 或 ../path/to/file
	// 3. 文件路径 + 行号：file.go:123
	// 4. 包路径：github.com/user/repo
	
	filePathPatterns := []*regexp.Regexp{
		// Unix 路径
		regexp.MustCompile(`^[/~]?([\w\-_\.]+/)+[\w\-_\.]+(\.\w+)?(:?\d+)?$`),
		// Windows 路径
		regexp.MustCompile(`^[A-Za-z]:\\([\w\-_\.]+\\)+[\w\-_\.]+(\.\w+)?(:?\d+)?$`),
		// 相对路径
		regexp.MustCompile(`^\.\.?/[\w\-_\./]+(\.\w+)?(:?\d+)?$`),
		// 包路径（如 github.com/user/repo）
		regexp.MustCompile(`^[\w\-_\.]+/[\w\-_\./]+(\.\w+)?$`),
		// 文件路径 + 行号（如 file.go:123）
		regexp.MustCompile(`^[\w\-_\./]+\.\w+:\d+$`),
	}
	
	// 如果整行就是一个路径（没有其他文本），则认为是文件路径行
	for _, pattern := range filePathPatterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	
	return false
}

// removeInlineCode 移除行内代码标记
func (s *SessionService) removeInlineCode(line string) string {
	// 移除行内代码标记 `code`
	inlineCodeRegex := regexp.MustCompile("`([^`]+)`")
	// 保留代码内容，但移除标记
	line = inlineCodeRegex.ReplaceAllString(line, "$1")
	
	return line
}

// cleanupEmptyLines 清理多余的空行
func (s *SessionService) cleanupEmptyLines(text string) string {
	lines := strings.Split(text, "\n")
	var filteredLines []string
	emptyCount := 0
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			emptyCount++
			if emptyCount <= 2 {
				filteredLines = append(filteredLines, line)
			}
		} else {
			emptyCount = 0
			filteredLines = append(filteredLines, line)
		}
	}
	
	return strings.Join(filteredLines, "\n")
}

// sessionToComposerData 将 WorkspaceSession 转换为 ComposerData
func (s *SessionService) sessionToComposerData(session *storage.WorkspaceSession) domainCursor.ComposerData {
	return domainCursor.ComposerData{
		Type:                session.Type,
		ComposerID:          session.ComposerID,
		Name:                session.Name,
		CreatedAt:           session.CreatedAt,
		LastUpdatedAt:       session.LastUpdatedAt,
		UnifiedMode:         session.UnifiedMode,
		ContextUsagePercent: session.ContextUsagePercent,
		TotalLinesAdded:     session.TotalLinesAdded,
		TotalLinesRemoved:   session.TotalLinesRemoved,
		FilesChangedCount:   session.FilesChangedCount,
		Subtitle:            session.Subtitle,
		IsArchived:          session.IsArchived,
		CreatedOnBranch:     session.CreatedOnBranch,
	}
}
