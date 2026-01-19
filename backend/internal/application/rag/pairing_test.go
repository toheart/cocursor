package rag

import (
	"testing"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
)

func TestPairMessages_StandardConversation(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
		{Type: domainCursor.MessageTypeAI, Text: "回答1", Timestamp: 2000},
		{Type: domainCursor.MessageTypeUser, Text: "问题2", Timestamp: 3000},
		{Type: domainCursor.MessageTypeAI, Text: "回答2", Timestamp: 4000},
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 2 {
		t.Fatalf("Expected 2 turns, got %d", len(turns))
	}

	// 验证第一个对话对
	if len(turns[0].UserMessages) != 1 || len(turns[0].AIMessages) != 1 {
		t.Errorf("Turn 0: expected 1 user + 1 AI message, got %d user + %d AI", len(turns[0].UserMessages), len(turns[0].AIMessages))
	}
	if turns[0].UserText != "问题1" {
		t.Errorf("Turn 0 user text: got %s, want 问题1", turns[0].UserText)
	}
	if turns[0].AIText != "回答1" {
		t.Errorf("Turn 0 AI text: got %s, want 回答1", turns[0].AIText)
	}

	// 验证第二个对话对
	if turns[1].UserText != "问题2" {
		t.Errorf("Turn 1 user text: got %s, want 问题2", turns[1].UserText)
	}
	if turns[1].AIText != "回答2" {
		t.Errorf("Turn 1 AI text: got %s, want 回答2", turns[1].AIText)
	}
}

func TestPairMessages_ConsecutiveUserMessages(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
		{Type: domainCursor.MessageTypeUser, Text: "补充问题", Timestamp: 1500},
		{Type: domainCursor.MessageTypeAI, Text: "回答", Timestamp: 2000},
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn, got %d", len(turns))
	}

	if len(turns[0].UserMessages) != 2 {
		t.Errorf("Expected 2 user messages, got %d", len(turns[0].UserMessages))
	}
	if turns[0].UserText != "问题1\n\n补充问题" {
		t.Errorf("User text: got %s, want 问题1\\n\\n补充问题", turns[0].UserText)
	}
}

func TestPairMessages_ConsecutiveAIMessages(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
		{Type: domainCursor.MessageTypeAI, Text: "回答1", Timestamp: 2000},
		{Type: domainCursor.MessageTypeAI, Text: "继续回答", Timestamp: 2500},
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn, got %d", len(turns))
	}

	if len(turns[0].AIMessages) != 2 {
		t.Errorf("Expected 2 AI messages, got %d", len(turns[0].AIMessages))
	}
	if turns[0].AIText != "回答1\n\n继续回答" {
		t.Errorf("AI text: got %s, want 回答1\\n\\n继续回答", turns[0].AIText)
	}
}

func TestPairMessages_IncompleteTurn(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
		{Type: domainCursor.MessageTypeAI, Text: "回答1", Timestamp: 2000},
		{Type: domainCursor.MessageTypeUser, Text: "问题2", Timestamp: 3000},
		// 没有 AI 回复
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 2 {
		t.Fatalf("Expected 2 turns, got %d", len(turns))
	}

	// 最后一个对话对应该是未完成的
	if !turns[1].IsIncomplete {
		t.Error("Expected last turn to be incomplete")
	}
	if len(turns[1].AIMessages) != 0 {
		t.Errorf("Expected 0 AI messages in incomplete turn, got %d", len(turns[1].AIMessages))
	}
}

func TestExtractMessageTexts(t *testing.T) {
	indexedMessages := []*IndexedMessage{
		{Message: &domainCursor.Message{Text: "消息1"}, MessageID: "msg-1", Index: 0},
		{Message: &domainCursor.Message{Text: "消息2"}, MessageID: "msg-2", Index: 1},
		{Message: &domainCursor.Message{Text: "消息3"}, MessageID: "msg-3", Index: 2},
	}

	texts := ExtractMessageTexts(indexedMessages)

	if len(texts) != 3 {
		t.Fatalf("Expected 3 texts, got %d", len(texts))
	}
	if texts[0] != "消息1" {
		t.Errorf("Text 0: got %s, want 消息1", texts[0])
	}
}

func TestExtractTurnTexts(t *testing.T) {
	turns := []*ConversationTurn{
		{CombinedText: "用户: 问题1\n\nAI: 回答1"},
		{CombinedText: "用户: 问题2\n\nAI: 回答2"},
	}

	texts := ExtractTurnTexts(turns)

	if len(texts) != 2 {
		t.Fatalf("Expected 2 texts, got %d", len(texts))
	}
	if texts[0] != "用户: 问题1\n\nAI: 回答1" {
		t.Errorf("Text 0 mismatch")
	}
}

// TestPairMessages_WithToolCalls 测试带 Tool Call 的消息配对
func TestPairMessages_WithToolCalls(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "帮我读取文件", Timestamp: 1000},
		{
			Type:      domainCursor.MessageTypeAI,
			Text:      "我来帮你读取文件",
			Timestamp: 2000,
			Tools: []*domainCursor.ToolCall{
				{Name: "read_file", Arguments: map[string]string{"path": "/path/to/file"}},
			},
		},
		{Type: domainCursor.MessageTypeAI, Text: "文件内容已读取", Timestamp: 2500},
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn, got %d", len(turns))
	}

	// 验证 Tool Call 被正确收集
	if len(turns[0].Tools) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(turns[0].Tools))
	}
	if turns[0].Tools[0].Name != "read_file" {
		t.Errorf("Tool name: got %s, want read_file", turns[0].Tools[0].Name)
	}

	// 验证文本内容不包含 Tool Call（应该在解析时已过滤）
	if turns[0].AIText != "我来帮你读取文件\n\n文件内容已读取" {
		t.Errorf("AI text: got %s, want 我来帮你读取文件\\n\\n文件内容已读取", turns[0].AIText)
	}
}

// TestPairMessages_ComplexScenario 测试复杂场景（混合连续消息和 Tool Call）
func TestPairMessages_ComplexScenario(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
		{Type: domainCursor.MessageTypeAI, Text: "回答1", Timestamp: 2000},
		{
			Type:      domainCursor.MessageTypeAI,
			Text:      "继续回答1",
			Timestamp: 2500,
			Tools: []*domainCursor.ToolCall{
				{Name: "codebase_search", Arguments: map[string]string{"query": "test"}},
			},
		},
		{Type: domainCursor.MessageTypeUser, Text: "问题2补充", Timestamp: 3000},
		{Type: domainCursor.MessageTypeAI, Text: "回答2", Timestamp: 4000},
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 2 {
		t.Fatalf("Expected 2 turns, got %d", len(turns))
	}

	// 验证第一个对话对
	if len(turns[0].AIMessages) != 2 {
		t.Errorf("Turn 0: expected 2 AI messages, got %d", len(turns[0].AIMessages))
	}
	if len(turns[0].Tools) != 1 {
		t.Errorf("Turn 0: expected 1 tool call, got %d", len(turns[0].Tools))
	}
	if turns[0].AIText != "回答1\n\n继续回答1" {
		t.Errorf("Turn 0 AI text mismatch: got %s", turns[0].AIText)
	}

	// 验证第二个对话对
	if turns[1].UserText != "问题2补充" {
		t.Errorf("Turn 1 user text mismatch: got %s", turns[1].UserText)
	}
}

// TestPairMessages_EmptyMessages 测试空消息列表
func TestPairMessages_EmptyMessages(t *testing.T) {
	messages := []*domainCursor.Message{}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 0 {
		t.Errorf("Expected 0 turns, got %d", len(turns))
	}
}

// TestPairMessages_OnlyUserMessages 测试只有用户消息
func TestPairMessages_OnlyUserMessages(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
		{Type: domainCursor.MessageTypeUser, Text: "问题2", Timestamp: 2000},
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn, got %d", len(turns))
	}

	if !turns[0].IsIncomplete {
		t.Error("Expected turn to be incomplete")
	}
	if len(turns[0].AIMessages) != 0 {
		t.Errorf("Expected 0 AI messages, got %d", len(turns[0].AIMessages))
	}
}

// TestPairMessages_OnlyAIMessages 测试只有 AI 消息（边界情况）
func TestPairMessages_OnlyAIMessages(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeAI, Text: "回答1", Timestamp: 1000},
		{Type: domainCursor.MessageTypeAI, Text: "回答2", Timestamp: 2000},
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn, got %d", len(turns))
	}

	if len(turns[0].UserMessages) != 0 {
		t.Errorf("Expected 0 user messages, got %d", len(turns[0].UserMessages))
	}
	if len(turns[0].AIMessages) != 2 {
		t.Errorf("Expected 2 AI messages, got %d", len(turns[0].AIMessages))
	}
}

// TestPairMessages_MessageIDs 测试消息 ID 生成
func TestPairMessages_MessageIDs(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
		{Type: domainCursor.MessageTypeAI, Text: "回答1", Timestamp: 2000},
	}

	turns := PairMessages(messages, "session-abc")

	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn, got %d", len(turns))
	}

	// 验证消息 ID 格式
	if len(turns[0].UserMessageIDs) != 1 {
		t.Errorf("Expected 1 user message ID, got %d", len(turns[0].UserMessageIDs))
	}
	if !contains(turns[0].UserMessageIDs, "session-abc-0-1000") {
		t.Errorf("User message ID not found: %v", turns[0].UserMessageIDs)
	}
	if !contains(turns[0].AIMessageIDs, "session-abc-1-2000") {
		t.Errorf("AI message ID not found: %v", turns[0].AIMessageIDs)
	}
}

// TestPairMessages_EmptyText 测试空文本消息
func TestPairMessages_EmptyText(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "", Timestamp: 1000},
		{Type: domainCursor.MessageTypeUser, Text: "   ", Timestamp: 1500}, // 只有空格
		{Type: domainCursor.MessageTypeAI, Text: "回答", Timestamp: 2000},
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn, got %d", len(turns))
	}

	// 空文本应该被过滤
	if turns[0].UserText != "" {
		t.Errorf("Expected empty user text after filtering, got: %s", turns[0].UserText)
	}
}

// TestPairMessages_MultipleToolCalls 测试多个 Tool Call
func TestPairMessages_MultipleToolCalls(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "帮我执行多个操作", Timestamp: 1000},
		{
			Type:      domainCursor.MessageTypeAI,
			Text:      "我来执行",
			Timestamp: 2000,
			Tools: []*domainCursor.ToolCall{
				{Name: "tool1", Arguments: map[string]string{"arg1": "value1"}},
				{Name: "tool2", Arguments: map[string]string{"arg2": "value2"}},
			},
		},
		{
			Type:      domainCursor.MessageTypeAI,
			Text:      "继续执行",
			Timestamp: 2500,
			Tools: []*domainCursor.ToolCall{
				{Name: "tool3", Arguments: map[string]string{"arg3": "value3"}},
			},
		},
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn, got %d", len(turns))
	}

	// 验证所有 Tool Call 都被收集
	if len(turns[0].Tools) != 3 {
		t.Errorf("Expected 3 tool calls, got %d", len(turns[0].Tools))
	}
}

// TestExtractMessageTexts_EmptyText 测试提取空文本
func TestExtractMessageTexts_EmptyText(t *testing.T) {
	indexedMessages := []*IndexedMessage{
		{Message: &domainCursor.Message{Text: "消息1"}, MessageID: "msg-1", Index: 0},
		{Message: &domainCursor.Message{Text: ""}, MessageID: "msg-2", Index: 1},
		{Message: &domainCursor.Message{Text: "   "}, MessageID: "msg-3", Index: 2},
		{Message: &domainCursor.Message{Text: "消息2"}, MessageID: "msg-4", Index: 3},
	}

	texts := ExtractMessageTexts(indexedMessages)

	if len(texts) != 4 {
		t.Fatalf("Expected 4 texts, got %d", len(texts))
	}

	// 验证空文本被处理（TrimSpace 后为空）
	if texts[1] != "" {
		t.Errorf("Expected empty text, got: %s", texts[1])
	}
	if texts[2] != "" {
		t.Errorf("Expected empty text after trim, got: %s", texts[2])
	}
}

// TestExtractMessageTexts_EmptyList 测试空列表
func TestExtractMessageTexts_EmptyList(t *testing.T) {
	indexedMessages := []*IndexedMessage{}

	texts := ExtractMessageTexts(indexedMessages)

	if len(texts) != 0 {
		t.Errorf("Expected 0 texts, got %d", len(texts))
	}
}

// TestExtractTurnTexts_EmptyList 测试空对话对列表
func TestExtractTurnTexts_EmptyList(t *testing.T) {
	turns := []*ConversationTurn{}

	texts := ExtractTurnTexts(turns)

	if len(texts) != 0 {
		t.Errorf("Expected 0 texts, got %d", len(texts))
	}
}

// TestPairMessages_TimestampOrder 测试时间戳顺序
func TestPairMessages_TimestampOrder(t *testing.T) {
	// 测试消息按时间戳顺序处理
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
		{Type: domainCursor.MessageTypeAI, Text: "回答1", Timestamp: 2000},
		{Type: domainCursor.MessageTypeUser, Text: "问题2", Timestamp: 3000},
		{Type: domainCursor.MessageTypeAI, Text: "回答2", Timestamp: 4000},
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 2 {
		t.Fatalf("Expected 2 turns, got %d", len(turns))
	}

	// 验证时间戳
	if turns[0].Timestamp != 1000 {
		t.Errorf("Turn 0 timestamp: got %d, want 1000", turns[0].Timestamp)
	}
	if turns[1].Timestamp != 3000 {
		t.Errorf("Turn 1 timestamp: got %d, want 3000", turns[1].Timestamp)
	}
}

// TestPairMessages_TurnIndex 测试对话对索引
func TestPairMessages_TurnIndex(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
		{Type: domainCursor.MessageTypeAI, Text: "回答1", Timestamp: 2000},
		{Type: domainCursor.MessageTypeUser, Text: "问题2", Timestamp: 3000},
		{Type: domainCursor.MessageTypeAI, Text: "回答2", Timestamp: 4000},
		{Type: domainCursor.MessageTypeUser, Text: "问题3", Timestamp: 5000},
		{Type: domainCursor.MessageTypeAI, Text: "回答3", Timestamp: 6000},
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 3 {
		t.Fatalf("Expected 3 turns, got %d", len(turns))
	}

	// 验证 TurnIndex
	for i, turn := range turns {
		if turn.TurnIndex != i {
			t.Errorf("Turn %d: expected TurnIndex %d, got %d", i, i, turn.TurnIndex)
		}
	}
}

// TestPairMessages_CombinedText 测试对话对文本生成
func TestPairMessages_CombinedText(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
		{Type: domainCursor.MessageTypeAI, Text: "回答1", Timestamp: 2000},
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn, got %d", len(turns))
	}

	expectedText := "用户: 问题1\n\nAI: 回答1"
	if turns[0].CombinedText != expectedText {
		t.Errorf("Combined text: got %s, want %s", turns[0].CombinedText, expectedText)
	}
}

// TestPairMessages_CombinedText_EmptyAI 测试 AI 文本为空的情况
func TestPairMessages_CombinedText_EmptyAI(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
		// 没有 AI 回复
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn, got %d", len(turns))
	}

	expectedText := "用户: 问题1\n\nAI: "
	if turns[0].CombinedText != expectedText {
		t.Errorf("Combined text: got %s, want %s", turns[0].CombinedText, expectedText)
	}
}

// TestPairMessages_CombinedText_EmptyUser 测试用户文本为空的情况
func TestPairMessages_CombinedText_EmptyUser(t *testing.T) {
	messages := []*domainCursor.Message{
		{Type: domainCursor.MessageTypeAI, Text: "回答1", Timestamp: 2000},
	}

	turns := PairMessages(messages, "session-1")

	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn, got %d", len(turns))
	}

	expectedText := "用户: \n\nAI: 回答1"
	if turns[0].CombinedText != expectedText {
		t.Errorf("Combined text: got %s, want %s", turns[0].CombinedText, expectedText)
	}
}

// contains 辅助函数检查切片是否包含元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
