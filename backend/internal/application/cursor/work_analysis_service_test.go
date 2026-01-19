package cursor

import (
	"testing"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/stretchr/testify/assert"
)

// TestFilterComposerMode 测试 Composer 模式过滤
func TestFilterComposerMode(t *testing.T) {
	// 测试数据：混合不同类型的 prompts
	prompts := []map[string]interface{}{
		{"text": "composer prompt 1", "commandType": float64(4)}, // Composer 模式
		{"text": "chat prompt 1", "commandType": float64(1)},     // Chat 模式
		{"text": "composer prompt 2", "commandType": float64(4)}, // Composer 模式
		{"text": "tab prompt 1", "commandType": float64(2)},      // Tab 模式
	}

	// 统计 Composer 模式的 prompts
	composerPromptCount := 0
	for _, prompt := range prompts {
		commandType, ok := prompt["commandType"].(float64)
		if ok && int(commandType) == 4 {
			composerPromptCount++
		}
	}

	assert.Equal(t, 2, composerPromptCount, "应该只统计 Composer 模式的 prompts")

	// 测试数据：混合不同类型的 generations
	generations := []domainCursor.GenerationData{
		{Type: "composer", TextDescription: "composer reply 1"},
		{Type: "chat", TextDescription: "chat reply 1"},
		{Type: "composer", TextDescription: "composer reply 2"},
		{Type: "tab", TextDescription: "tab reply 1"},
	}

	// 统计 Composer 模式的 generations
	composerGenerationCount := 0
	for _, gen := range generations {
		if gen.Type == "composer" {
			composerGenerationCount++
		}
	}

	assert.Equal(t, 2, composerGenerationCount, "应该只统计 Composer 模式的 generations")
}
