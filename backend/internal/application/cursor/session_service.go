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
)

// SessionService 会话查询服务
type SessionService struct {
	projectManager *ProjectManager
	pathResolver   *infraCursor.PathResolver
	dbReader       *infraCursor.DBReader
}

// NewSessionService 创建会话查询服务实例
func NewSessionService(projectManager *ProjectManager) *SessionService {
	return &SessionService{
		projectManager: projectManager,
		pathResolver:   infraCursor.NewPathResolver(),
		dbReader:       infraCursor.NewDBReader(),
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

	// 收集所有会话
	var allSessions []*domainCursor.ComposerData
	for _, workspaceID := range workspaceIDs {
		workspaceDBPath, err := s.pathResolver.GetWorkspaceDBPath(workspaceID)
		if err != nil {
			continue
		}

		composerDataValue, err := s.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
		if err != nil {
			continue
		}

		composers, err := domainCursor.ParseComposerData(string(composerDataValue))
		if err != nil {
			continue
		}

		// 转换为指针切片
		for i := range composers {
			allSessions = append(allSessions, &composers[i])
		}
	}

	// 搜索过滤
	if search != "" {
		filtered := []*domainCursor.ComposerData{}
		searchLower := strings.ToLower(search)
		for _, session := range allSessions {
			if strings.Contains(strings.ToLower(session.Name), searchLower) {
				filtered = append(filtered, session)
			}
		}
		allSessions = filtered
	}

	// 按最后更新时间排序（倒序）
	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].LastUpdatedAt > allSessions[j].LastUpdatedAt
	})

	total := len(allSessions)

	// 分页
	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var result []*domainCursor.ComposerData
	if start < total {
		result = allSessions[start:end]
	}

	hasMore := end < total

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

	// 1. 直接扫描 ~/.cursor/projects/ 下的所有项目目录，查找 transcript 文件
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("无法获取用户目录: %w", err)
	}

	projectsDir := filepath.Join(homeDir, ".cursor", "projects")
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
