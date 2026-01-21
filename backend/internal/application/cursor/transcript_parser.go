package cursor

import (
	"log"
	"strings"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
)

// transcriptParser 对话记录解析器
// 使用状态机模式，将复杂的解析逻辑拆分为独立的处理函数
type transcriptParser struct {
	lines         []string
	messages      []*domainCursor.Message
	baseTimestamp int64
	messageIndex  int

	// 当前消息状态
	currentRole  string
	currentText  strings.Builder
	currentTools []*domainCursor.ToolCall

	// 依赖
	service *SessionService
}

// parseResult 行处理结果
type parseResult struct {
	skipLines int  // 需要额外跳过的行数（不包括当前行）
	handled   bool // 是否已处理该行
}

// newTranscriptParser 创建解析器实例
func newTranscriptParser(service *SessionService, content string, baseTimestamp int64) *transcriptParser {
	return &transcriptParser{
		lines:         strings.Split(content, "\n"),
		messages:      make([]*domainCursor.Message, 0),
		baseTimestamp: baseTimestamp,
		service:       service,
	}
}

// parse 执行解析，返回消息列表
func (p *transcriptParser) parse() ([]*domainCursor.Message, error) {
	for i := 0; i < len(p.lines); i++ {
		line := p.lines[i]

		// 1. 检测角色切换
		if result := p.handleRoleSwitch(line); result.handled {
			continue
		}

		// 2. 处理工具调用（只在 assistant 角色下）
		if result := p.handleToolCall(i); result.handled {
			i += result.skipLines
			continue
		}

		// 3. 跳过工具结果
		if result := p.handleToolResult(i); result.handled {
			i += result.skipLines
			continue
		}

		// 4. 处理 XML 标签
		if result := p.handleXMLTags(i); result.handled {
			i += result.skipLines
			continue
		}

		// 5. 累积文本
		p.appendText(line)
	}

	// 保存最后一条消息
	p.saveCurrentMessage()

	log.Printf("[parseTranscript] 解析了 %d 条消息", len(p.messages))
	return p.messages, nil
}

// handleRoleSwitch 处理角色切换（user:/assistant:）
func (p *transcriptParser) handleRoleSwitch(line string) parseResult {
	switch line {
	case "user:":
		p.saveCurrentMessage()
		p.currentRole = "user"
		p.currentText.Reset()
		p.currentTools = nil
		return parseResult{handled: true}
	case "assistant:":
		p.saveCurrentMessage()
		p.currentRole = "assistant"
		p.currentText.Reset()
		p.currentTools = nil
		return parseResult{handled: true}
	}
	return parseResult{handled: false}
}

// handleToolCall 处理工具调用 [Tool call]
func (p *transcriptParser) handleToolCall(lineIndex int) parseResult {
	line := p.lines[lineIndex]

	// 只在 assistant 消息中处理工具调用
	if !strings.HasPrefix(line, "[Tool call]") || p.currentRole != "assistant" {
		return parseResult{handled: false}
	}

	toolName := strings.TrimSpace(strings.TrimPrefix(line, "[Tool call]"))
	if toolName == "" {
		return parseResult{handled: true, skipLines: 0}
	}

	// 解析工具参数
	args, skipLines := p.parseToolArguments(lineIndex + 1)

	// 创建工具调用对象
	toolCall := &domainCursor.ToolCall{
		Name:      toolName,
		Arguments: args,
	}
	p.currentTools = append(p.currentTools, toolCall)

	return parseResult{handled: true, skipLines: skipLines}
}

// toolArgParser 工具参数解析器的内部状态
type toolArgParser struct {
	args            map[string]string
	currentArgKey   string
	currentArgValue strings.Builder
}

// newToolArgParser 创建工具参数解析器
func newToolArgParser() *toolArgParser {
	return &toolArgParser{
		args: make(map[string]string),
	}
}

// saveCurrentArg 保存当前参数
func (tap *toolArgParser) saveCurrentArg() {
	if tap.currentArgKey != "" {
		tap.args[tap.currentArgKey] = strings.TrimSpace(tap.currentArgValue.String())
		tap.currentArgValue.Reset()
	}
}

// parseColonFormat 解析 key: value 格式
func (tap *toolArgParser) parseColonFormat(trimmed string) {
	tap.saveCurrentArg()
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) == 2 {
		tap.currentArgKey = strings.TrimSpace(parts[0])
		tap.currentArgValue.WriteString(strings.TrimSpace(parts[1]))
	}
}

// parseEqualsFormat 解析 key=value 格式
func (tap *toolArgParser) parseEqualsFormat(trimmed string) {
	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) == 2 {
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" {
			tap.args[key] = value
		}
	}
}

// appendToCurrentValue 追加到当前参数值
func (tap *toolArgParser) appendToCurrentValue(trimmed string) {
	if tap.currentArgValue.Len() > 0 {
		tap.currentArgValue.WriteString(" ")
	}
	tap.currentArgValue.WriteString(trimmed)
}

// parseToolArguments 解析工具参数，返回参数 map 和跳过的行数
func (p *transcriptParser) parseToolArguments(startIndex int) (map[string]string, int) {
	tap := newToolArgParser()
	skipLines := 0

	for j := startIndex; j < len(p.lines); j++ {
		nextLine := p.lines[j]

		// 检查终止条件
		if p.isTerminatorLine(nextLine) {
			tap.saveCurrentArg()
			skipLines = p.calculateSkipLines(j, startIndex)
			break
		}

		trimmed := strings.TrimSpace(nextLine)
		p.processToolArgLine(tap, nextLine, trimmed)

		// 处理到文件末尾的情况
		if j == len(p.lines)-1 {
			tap.saveCurrentArg()
			skipLines = j - startIndex
		}
	}

	return tap.args, skipLines
}

// processToolArgLine 处理单行工具参数
func (p *transcriptParser) processToolArgLine(tap *toolArgParser, line, trimmed string) {
	switch {
	case p.isNewParameter(line, trimmed):
		tap.parseColonFormat(trimmed)
	case tap.currentArgKey != "":
		tap.appendToCurrentValue(trimmed)
	case strings.Contains(trimmed, "="):
		tap.parseEqualsFormat(trimmed)
	}
}

// calculateSkipLines 计算需要跳过的行数
func (p *transcriptParser) calculateSkipLines(currentIndex, startIndex int) int {
	skipLines := currentIndex - startIndex - 1
	if skipLines < 0 {
		return 0
	}
	return skipLines
}

// isTerminatorLine 检查是否是终止行
func (p *transcriptParser) isTerminatorLine(line string) bool {
	return line == "" ||
		strings.HasPrefix(line, "[Tool call]") ||
		strings.HasPrefix(line, "[Tool result]") ||
		line == "user:" ||
		line == "assistant:"
}

// isNewParameter 检查是否是新的参数行
func (p *transcriptParser) isNewParameter(line, trimmed string) bool {
	return strings.Contains(trimmed, ":") &&
		!strings.HasPrefix(line, " ") &&
		!strings.HasPrefix(line, "\t")
}

// handleToolResult 跳过工具结果 [Tool result]
func (p *transcriptParser) handleToolResult(lineIndex int) parseResult {
	line := p.lines[lineIndex]

	if !strings.HasPrefix(line, "[Tool result]") {
		return parseResult{handled: false}
	}

	skipLines := p.skipUntilTerminator(lineIndex + 1)
	return parseResult{handled: true, skipLines: skipLines}
}

// skipUntilTerminator 跳过直到遇到终止行，返回跳过的行数
func (p *transcriptParser) skipUntilTerminator(startIndex int) int {
	for j := startIndex; j < len(p.lines); j++ {
		if p.isTerminatorLine(p.lines[j]) {
			// 返回跳过的行数（不包括终止行）
			skipLines := j - startIndex - 1
			if skipLines < 0 {
				skipLines = 0
			}
			return skipLines
		}
		if j == len(p.lines)-1 {
			return j - startIndex
		}
	}
	return 0
}

// handleXMLTags 处理 XML 标签（<user_query>, <think>）
func (p *transcriptParser) handleXMLTags(lineIndex int) parseResult {
	line := p.lines[lineIndex]

	// 处理 <user_query> 标签：提取内容
	if strings.Contains(line, "<user_query>") {
		content, skipLines := p.extractTagContent(lineIndex, "<user_query>", "</user_query>")
		if content != "" {
			p.currentText.WriteString(content)
		}
		return parseResult{handled: true, skipLines: skipLines}
	}

	// 处理 <think> 标签：跳过内容
	if strings.Contains(line, "<think>") {
		skipLines := p.skipTagContent(lineIndex, "</think>")
		return parseResult{handled: true, skipLines: skipLines}
	}

	return parseResult{handled: false}
}

// extractTagContent 提取标签内容，返回内容和需要跳过的行数
func (p *transcriptParser) extractTagContent(lineIndex int, openTag, closeTag string) (string, int) {
	line := p.lines[lineIndex]
	startIdx := strings.Index(line, openTag)
	if startIdx == -1 {
		return "", 0
	}

	contentStart := startIdx + len(openTag)
	endIdx := strings.Index(line[contentStart:], closeTag)

	// 在同一行找到闭合标签
	if endIdx != -1 {
		return strings.TrimSpace(line[contentStart : contentStart+endIdx]), 0
	}

	// 跨行提取
	var content strings.Builder
	content.WriteString(line[contentStart:])

	for j := lineIndex + 1; j < len(p.lines); j++ {
		if strings.Contains(p.lines[j], closeTag) {
			idx := strings.Index(p.lines[j], closeTag)
			content.WriteString("\n")
			content.WriteString(p.lines[j][:idx])
			return strings.TrimSpace(content.String()), j - lineIndex
		}
		content.WriteString("\n")
		content.WriteString(p.lines[j])
	}

	return strings.TrimSpace(content.String()), len(p.lines) - 1 - lineIndex
}

// skipTagContent 跳过标签内容直到找到闭合标签，返回跳过的行数
func (p *transcriptParser) skipTagContent(lineIndex int, closeTag string) int {
	for j := lineIndex + 1; j < len(p.lines); j++ {
		if strings.Contains(p.lines[j], closeTag) {
			return j - lineIndex
		}
	}
	return len(p.lines) - 1 - lineIndex
}

// appendText 累积文本到当前消息
func (p *transcriptParser) appendText(line string) {
	if p.currentRole == "" {
		return
	}
	if p.currentText.Len() > 0 {
		p.currentText.WriteString("\n")
	}
	p.currentText.WriteString(line)
}

// saveCurrentMessage 保存当前消息到列表
func (p *transcriptParser) saveCurrentMessage() {
	if p.currentRole == "" || p.currentText.Len() == 0 {
		return
	}

	text := strings.TrimSpace(p.currentText.String())
	if text == "" {
		return
	}

	// 过滤消息文本
	text = p.service.filterMessageText(text)
	if text == "" {
		return
	}

	// 计算时间戳（每条消息间隔1秒）
	timestamp := p.baseTimestamp + int64(p.messageIndex*1000)

	msg := &domainCursor.Message{
		Type:      getMessageType(p.currentRole),
		Text:      text,
		Timestamp: timestamp,
	}

	// AI 消息额外处理
	if msg.Type == domainCursor.MessageTypeAI {
		msg.CodeBlocks = p.service.extractCodeBlocks(text)
		if len(p.currentTools) > 0 {
			msg.Tools = make([]*domainCursor.ToolCall, len(p.currentTools))
			copy(msg.Tools, p.currentTools)
		}
	}

	p.messages = append(p.messages, msg)
	p.messageIndex++
	p.currentTools = nil
}
