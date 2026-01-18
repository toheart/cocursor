package cursor

import (
	"testing"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/stretchr/testify/assert"
)

// TestCountWorkspacePromptsAndGenerations 测试 countWorkspacePromptsAndGenerations 方法
func TestCountWorkspacePromptsAndGenerations(t *testing.T) {
	service := &WorkAnalysisService{
		dbReader: infraCursor.NewDBReader(),
	}

	startDate := time.Now().Add(-7 * 24 * time.Hour).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	// 测试：数据库不存在时返回 0, 0
	prompts, generations := service.countWorkspacePromptsAndGenerations("nonexistent.db", startDate, endDate)
	assert.Equal(t, 0, prompts, "数据库不存在时 prompts 应为 0")
	assert.Equal(t, 0, generations, "数据库不存在时 generations 应为 0")
}

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
