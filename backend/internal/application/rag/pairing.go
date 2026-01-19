package rag

import (
	"fmt"
	"strings"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
)

// ConversationTurn 对话对
type ConversationTurn struct {
	UserMessages   []*domainCursor.Message
	AIMessages     []*domainCursor.Message
	UserText       string
	AIText         string
	CombinedText   string
	UserMessageIDs []string
	AIMessageIDs   []string
	Tools          []*domainCursor.ToolCall
	TurnIndex      int
	Timestamp      int64
	IsIncomplete   bool
}

// IndexedMessage 带索引信息的消息
type IndexedMessage struct {
	Message   *domainCursor.Message
	MessageID string
	Index     int
}

// PairMessages 将消息配对为对话对
func PairMessages(messages []*domainCursor.Message, sessionID string) []*ConversationTurn {
	var turns []*ConversationTurn
	var currentUserMsgs []*IndexedMessage
	var currentAIMsgs []*IndexedMessage

	for i, msg := range messages {
		messageID := fmt.Sprintf("%s-%d-%d", sessionID, i, msg.Timestamp)

		if msg.Type == domainCursor.MessageTypeUser {
			// 如果之前有未配对的 AI 消息，先创建对话对
			if len(currentAIMsgs) > 0 {
				turn := createTurn(currentUserMsgs, currentAIMsgs, len(turns))
				turns = append(turns, turn)
				currentUserMsgs = nil
				currentAIMsgs = nil
			}
			// 累积用户消息（可能连续多条）
			currentUserMsgs = append(currentUserMsgs, &IndexedMessage{
				Message:   msg,
				MessageID: messageID,
				Index:     i,
			})

		} else if msg.Type == domainCursor.MessageTypeAI {
			// 累积 AI 消息（可能连续多条）
			// 注意：即使有 tool call，也属于同一个对话对
			currentAIMsgs = append(currentAIMsgs, &IndexedMessage{
				Message:   msg,
				MessageID: messageID,
				Index:     i,
			})
		}
	}

	// 处理最后未配对的消息
	if len(currentUserMsgs) > 0 || len(currentAIMsgs) > 0 {
		turn := createTurn(currentUserMsgs, currentAIMsgs, len(turns))
		// 如果只有用户消息没有 AI 回复，标记为未完成
		if len(currentAIMsgs) == 0 {
			turn.IsIncomplete = true
		}
		turns = append(turns, turn)
	}

	return turns
}

// createTurn 创建对话对（合并多条连续消息）
func createTurn(userMsgs, aiMsgs []*IndexedMessage, turnIndex int) *ConversationTurn {
	// 合并用户消息文本
	userText := combineMessageTexts(userMsgs)

	// 合并 AI 消息文本
	aiText := combineMessageTexts(aiMsgs)

	// 收集所有 tool call（从所有 AI 消息中）
	var allTools []*domainCursor.ToolCall
	var allUserMessages []*domainCursor.Message
	var allAIMessages []*domainCursor.Message

	for _, indexedMsg := range userMsgs {
		allUserMessages = append(allUserMessages, indexedMsg.Message)
	}

	for _, indexedMsg := range aiMsgs {
		allAIMessages = append(allAIMessages, indexedMsg.Message)
		allTools = append(allTools, indexedMsg.Message.Tools...)
	}

	// 生成对话对文本（用于向量化）
	combinedText := fmt.Sprintf("用户: %s\n\nAI: %s", userText, aiText)

	// 生成消息 ID 列表
	userMessageIDs := make([]string, len(userMsgs))
	for i, indexedMsg := range userMsgs {
		userMessageIDs[i] = indexedMsg.MessageID
	}

	aiMessageIDs := make([]string, len(aiMsgs))
	for i, indexedMsg := range aiMsgs {
		aiMessageIDs[i] = indexedMsg.MessageID
	}

	// 获取时间戳（取第一条消息的时间戳）
	timestamp := int64(0)
	if len(userMsgs) > 0 {
		timestamp = userMsgs[0].Message.Timestamp
	} else if len(aiMsgs) > 0 {
		timestamp = aiMsgs[0].Message.Timestamp
	}

	return &ConversationTurn{
		UserMessages:   allUserMessages,
		AIMessages:     allAIMessages,
		UserText:       userText,
		AIText:         aiText,
		CombinedText:   combinedText,
		UserMessageIDs: userMessageIDs,
		AIMessageIDs:   aiMessageIDs,
		Tools:          allTools,
		TurnIndex:      turnIndex,
		Timestamp:      timestamp,
		IsIncomplete:   false,
	}
}

// combineMessageTexts 合并多条消息的文本
func combineMessageTexts(indexedMessages []*IndexedMessage) string {
	if len(indexedMessages) == 0 {
		return ""
	}

	var texts []string
	for _, indexedMsg := range indexedMessages {
		text := strings.TrimSpace(indexedMsg.Message.Text)
		if text != "" {
			texts = append(texts, text)
		}
	}

	return strings.Join(texts, "\n\n")
}

// ExtractMessageTexts 提取消息级别文本（用于向量化）
func ExtractMessageTexts(indexedMessages []*IndexedMessage) []string {
	texts := make([]string, len(indexedMessages))
	for i, indexedMsg := range indexedMessages {
		texts[i] = strings.TrimSpace(indexedMsg.Message.Text)
	}
	return texts
}

// ExtractTurnTexts 提取对话对级别文本（用于向量化）
func ExtractTurnTexts(turns []*ConversationTurn) []string {
	texts := make([]string, len(turns))
	for i, turn := range turns {
		texts[i] = turn.CombinedText
	}
	return texts
}
