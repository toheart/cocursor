package rag

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
)

// ContentExtractor 内容提取器
// 用于从对话消息中提取核心内容，过滤代码块和噪音
type ContentExtractor struct {
	// 配置参数
	MaxUserQueryLen    int // 用户问题最大长度
	MaxAIResponseLen   int // AI 回答最大长度
	MaxQueryPreviewLen int // 问题预览最大长度
}

// NewContentExtractor 创建内容提取器
func NewContentExtractor() *ContentExtractor {
	return &ContentExtractor{
		MaxUserQueryLen:    1000,
		MaxAIResponseLen:   2000,
		MaxQueryPreviewLen: 200,
	}
}

// ExtractionResult 提取结果
type ExtractionResult struct {
	UserQuery      string   // 用户问题
	AIResponseCore string   // AI 核心回答
	VectorText     string   // 组合后的向量化文本
	ToolsUsed      []string // 使用的工具
	FilesModified  []string // 修改的文件
	CodeLanguages  []string // 代码语言
	HasCode        bool     // 是否包含代码
}

// ExtractFromTurn 从对话对中提取内容
func (e *ContentExtractor) ExtractFromTurn(turn *ConversationTurn) *ExtractionResult {
	result := &ExtractionResult{}

	// 提取用户问题
	result.UserQuery = e.extractUserQuery(turn.UserMessages)

	// 提取工具信息
	result.ToolsUsed, result.FilesModified = e.extractToolInfo(turn.AIMessages)

	// 提取代码语言和检测代码
	result.CodeLanguages, result.HasCode = e.extractCodeInfo(turn.AIMessages)

	// 提取 AI 核心回答
	result.AIResponseCore = e.extractAICore(turn.AIMessages)

	// 组合向量化文本
	result.VectorText = e.buildVectorText(result)

	return result
}

// extractUserQuery 提取用户问题
func (e *ContentExtractor) extractUserQuery(messages []*domainCursor.Message) string {
	var parts []string

	for _, msg := range messages {
		text := strings.TrimSpace(msg.Text)
		if text != "" {
			parts = append(parts, text)
		}
	}

	combined := strings.Join(parts, "\n")
	return e.truncateAtSentence(combined, e.MaxUserQueryLen)
}

// extractAICore 提取 AI 核心回答
func (e *ContentExtractor) extractAICore(messages []*domainCursor.Message) string {
	var parts []string

	for _, msg := range messages {
		text := msg.Text

		// 移除代码块
		text = e.removeCodeBlocks(text)

		// 移除系统标签
		text = e.removeSystemTags(text)

		// 过滤非自然语言内容
		text = e.filterNonNaturalLanguage(text)

		text = strings.TrimSpace(text)
		if text != "" {
			parts = append(parts, text)
		}
	}

	combined := strings.Join(parts, "\n")
	return e.truncateAtSentence(combined, e.MaxAIResponseLen)
}

// codeBlockPattern 代码块正则（匹配 ``` 代码块格式）
var codeBlockPattern = regexp.MustCompile("(?s)(?:```)[\\w]*[\\s\\S]*?(?:```)")

// removeCodeBlocks 移除代码块
func (e *ContentExtractor) removeCodeBlocks(text string) string {
	return codeBlockPattern.ReplaceAllString(text, "")
}

// systemTagPatterns 系统标签正则
var systemTagPatterns = []*regexp.Regexp{
	regexp.MustCompile("(?s)<think>.*?</think>"),
	regexp.MustCompile("(?s)<context>.*?</context>"),
	regexp.MustCompile("(?s)<git_status>.*?</git_status>"),
	regexp.MustCompile("(?s)<system_reminder>.*?</system_reminder>"),
	regexp.MustCompile(`(?m)^\[Tool call\].*$`),
	regexp.MustCompile(`(?m)^\[Tool result\].*$`),
	regexp.MustCompile(`(?m)^\[Thinking\].*$`),
}

// removeSystemTags 移除系统标签
func (e *ContentExtractor) removeSystemTags(text string) string {
	for _, pattern := range systemTagPatterns {
		text = pattern.ReplaceAllString(text, "")
	}
	return text
}

// logLinePattern 日志行正则
var logLinePattern = regexp.MustCompile(`(?m)^\d{4}[-/]\d{2}[-/]\d{2}[T ]\d{2}:\d{2}.*$`)

// filePathPattern 文件路径正则
var filePathPattern = regexp.MustCompile(`(?m)^\s*(/[\w./\-]+|[A-Za-z]:[\][\w.\\\-]+)\s*$`)

// emptyLinesPattern 多余空行正则
var emptyLinesPattern = regexp.MustCompile("\n{3,}")

// filterNonNaturalLanguage 过滤非自然语言内容
func (e *ContentExtractor) filterNonNaturalLanguage(text string) string {
	// 移除日志行
	text = logLinePattern.ReplaceAllString(text, "")

	// 移除纯文件路径行
	text = filePathPattern.ReplaceAllString(text, "")

	// 移除多余的空行
	text = emptyLinesPattern.ReplaceAllString(text, "\n\n")

	return text
}

// extractToolInfo 提取工具信息
func (e *ContentExtractor) extractToolInfo(messages []*domainCursor.Message) (toolsUsed, filesModified []string) {
	toolSet := make(map[string]bool)
	fileSet := make(map[string]bool)

	for _, msg := range messages {
		if msg.Tools == nil {
			continue
		}

		for _, tool := range msg.Tools {
			toolName := tool.Name
			if toolName != "" {
				toolSet[toolName] = true
			}

			// 从 Write/StrReplace 工具中提取文件
			if toolName == "Write" || toolName == "StrReplace" || toolName == "Edit" {
				if path, ok := tool.Arguments["path"]; ok {
					// 只保留文件名
					fileName := filepath.Base(path)
					if fileName != "" && fileName != "." {
						fileSet[fileName] = true
					}
				}
			}
		}
	}

	for tool := range toolSet {
		toolsUsed = append(toolsUsed, tool)
	}
	for file := range fileSet {
		filesModified = append(filesModified, file)
	}

	return toolsUsed, filesModified
}

// codeBlockLangPattern 代码块语言提取正则
var codeBlockLangPattern = regexp.MustCompile("(?:```)([a-zA-Z]+)")

// extractCodeInfo 提取代码信息
func (e *ContentExtractor) extractCodeInfo(messages []*domainCursor.Message) (languages []string, hasCode bool) {
	langSet := make(map[string]bool)

	for _, msg := range messages {
		// 从代码块提取语言
		matches := codeBlockLangPattern.FindAllStringSubmatch(msg.Text, -1)
		for _, match := range matches {
			if len(match) > 1 && match[1] != "" {
				langSet[match[1]] = true
				hasCode = true
			}
		}

		// 从消息的 CodeBlocks 字段提取
		if msg.CodeBlocks != nil {
			for _, cb := range msg.CodeBlocks {
				if cb.Language != "" {
					langSet[cb.Language] = true
				}
				hasCode = true
			}
		}
	}

	for lang := range langSet {
		languages = append(languages, lang)
	}

	return languages, hasCode
}

// buildVectorText 组合向量化文本
func (e *ContentExtractor) buildVectorText(result *ExtractionResult) string {
	var parts []string

	// 问题部分
	if result.UserQuery != "" {
		parts = append(parts, "问题: "+result.UserQuery)
	}

	// 回答部分
	if result.AIResponseCore != "" {
		parts = append(parts, "回答: "+result.AIResponseCore)
	}

	// 操作部分
	if len(result.ToolsUsed) > 0 {
		parts = append(parts, "操作: "+strings.Join(result.ToolsUsed, ", "))
	}

	// 文件部分
	if len(result.FilesModified) > 0 {
		parts = append(parts, "文件: "+strings.Join(result.FilesModified, ", "))
	}

	return strings.Join(parts, "\n\n")
}

// truncateAtSentence 在句子边界截断
func (e *ContentExtractor) truncateAtSentence(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	// 在 maxLen 之前找到最后一个句子结束符
	truncated := text[:maxLen]

	// 中英文句子结束符
	sentenceEnds := []rune{'。', '！', '？', '.', '!', '?', '\n'}

	lastEnd := -1
	for i, r := range truncated {
		for _, end := range sentenceEnds {
			if r == end {
				lastEnd = i
				break
			}
		}
	}

	if lastEnd > maxLen/2 {
		// 在句子边界截断
		return text[:lastEnd+1]
	}

	// 找不到合适的句子边界，在最后一个空格处截断
	lastSpace := strings.LastIndexFunc(truncated, unicode.IsSpace)
	if lastSpace > maxLen/2 {
		return text[:lastSpace] + "..."
	}

	// 直接截断
	return truncated + "..."
}
