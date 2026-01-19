package rag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMergeResults_Basic 测试基本合并逻辑
func TestMergeResults_Basic(t *testing.T) {
	service := &SearchService{}

	turnResults := []*SearchResult{
		{Type: "turn", SessionID: "session-1", TurnIndex: 0, Score: 0.8},
		{Type: "turn", SessionID: "session-2", TurnIndex: 0, Score: 0.7},
	}

	messageResults := []*SearchResult{
		{Type: "message", SessionID: "session-1", MessageID: "msg-1", Score: 0.9},
		{Type: "message", SessionID: "session-3", MessageID: "msg-2", Score: 0.6},
	}

	merged := service.mergeResults(turnResults, messageResults, 10)

	// 验证结果数量
	assert.LessOrEqual(t, len(merged), 10)

	// 验证对话对分数被加权（0.8 * 1.2 ≈ 0.96）
	foundTurn := false
	for _, result := range merged {
		if result.Type == "turn" && result.SessionID == "session-1" {
			// 使用近似比较，因为浮点数精度问题
			assert.InDelta(t, 0.96, result.Score, 0.001, "Turn score should be weighted")
			foundTurn = true
		}
	}
	assert.True(t, foundTurn, "Should find weighted turn result")
}

// TestMergeResults_Deduplication 测试去重逻辑
func TestMergeResults_Deduplication(t *testing.T) {
	service := &SearchService{}

	turnResults := []*SearchResult{
		{Type: "turn", SessionID: "session-1", TurnIndex: 0, Score: 0.8},
	}

	messageResults := []*SearchResult{
		{Type: "message", SessionID: "session-1", MessageID: "msg-1", Score: 0.9},
		{Type: "message", SessionID: "session-1", MessageID: "msg-1", Score: 0.85}, // 重复
	}

	merged := service.mergeResults(turnResults, messageResults, 10)

	// 验证去重
	messageCount := 0
	for _, result := range merged {
		if result.Type == "message" && result.MessageID == "msg-1" {
			messageCount++
		}
	}
	assert.Equal(t, 1, messageCount, "Duplicate messages should be removed")
}

// TestMergeResults_Sorting 测试排序逻辑
func TestMergeResults_Sorting(t *testing.T) {
	service := &SearchService{}

	turnResults := []*SearchResult{
		{Type: "turn", SessionID: "session-1", TurnIndex: 0, Score: 0.5},
	}

	messageResults := []*SearchResult{
		{Type: "message", SessionID: "session-2", MessageID: "msg-1", Score: 0.9},
		{Type: "message", SessionID: "session-3", MessageID: "msg-2", Score: 0.7},
	}

	merged := service.mergeResults(turnResults, messageResults, 10)

	// 验证按分数降序排序
	for i := 0; i < len(merged)-1; i++ {
		assert.GreaterOrEqual(t, merged[i].Score, merged[i+1].Score,
			"Results should be sorted by score in descending order")
	}
}

// TestMergeResults_Limit 测试结果数量限制
func TestMergeResults_Limit(t *testing.T) {
	service := &SearchService{}

	// 创建超过限制的结果
	turnResults := make([]*SearchResult, 5)
	messageResults := make([]*SearchResult, 10)

	for i := 0; i < 5; i++ {
		turnResults[i] = &SearchResult{
			Type:      "turn",
			SessionID: "session-1",
			TurnIndex: i,
			Score:     float32(0.8 - float32(i)*0.1),
		}
	}

	for i := 0; i < 10; i++ {
		messageResults[i] = &SearchResult{
			Type:      "message",
			SessionID: "session-1",
			MessageID: "msg-" + string(rune(i)),
			Score:     float32(0.9 - float32(i)*0.1),
		}
	}

	merged := service.mergeResults(turnResults, messageResults, 5)

	// 验证结果数量不超过限制
	assert.LessOrEqual(t, len(merged), 5)
}

// TestGetResultKey 测试结果键生成
func TestGetResultKey(t *testing.T) {
	service := &SearchService{}

	// 测试对话对键
	turnResult := &SearchResult{
		Type:      "turn",
		SessionID: "session-1",
		TurnIndex: 5,
	}
	key1 := service.getResultKey(turnResult)
	assert.Equal(t, "session-1:turn:5", key1)

	// 测试消息键
	messageResult := &SearchResult{
		Type:      "message",
		SessionID: "session-2",
		MessageID: "msg-123",
	}
	key2 := service.getResultKey(messageResult)
	assert.Equal(t, "session-2:message:msg-123", key2)
}

// TestMergeResults_EmptyResults 测试空结果合并
func TestMergeResults_EmptyResults(t *testing.T) {
	service := &SearchService{}

	merged := service.mergeResults([]*SearchResult{}, []*SearchResult{}, 10)

	assert.Empty(t, merged)
}

// TestMergeResults_TurnWeighting 测试对话对加权
func TestMergeResults_TurnWeighting(t *testing.T) {
	service := &SearchService{}

	turnResults := []*SearchResult{
		{Type: "turn", SessionID: "session-1", TurnIndex: 0, Score: 0.5},
		{Type: "turn", SessionID: "session-2", TurnIndex: 0, Score: 1.0},
	}

	messageResults := []*SearchResult{}

	merged := service.mergeResults(turnResults, messageResults, 10)

	// 验证所有对话对都被加权 20%
	for _, result := range merged {
		if result.SessionID == "session-1" {
			assert.InDelta(t, 0.6, result.Score, 0.001, "0.5 * 1.2 ≈ 0.6")
		}
		if result.SessionID == "session-2" {
			assert.InDelta(t, 1.2, result.Score, 0.001, "1.0 * 1.2 = 1.2")
		}
	}
}
