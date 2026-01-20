package rag

import (
	"testing"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/stretchr/testify/assert"
)

func TestContentExtractor_ExtractFromTurn(t *testing.T) {
	extractor := NewContentExtractor()

	t.Run("基本提取", func(t *testing.T) {
		turn := &ConversationTurn{
			UserMessages: []*domainCursor.Message{
				{Text: "如何实现用户认证？", Type: domainCursor.MessageTypeUser},
			},
			AIMessages: []*domainCursor.Message{
				{Text: "用户认证可以通过 JWT 实现。首先需要安装相关依赖。", Type: domainCursor.MessageTypeAI},
			},
		}

		result := extractor.ExtractFromTurn(turn)

		assert.Equal(t, "如何实现用户认证？", result.UserQuery)
		assert.Contains(t, result.AIResponseCore, "JWT")
		assert.Contains(t, result.VectorText, "问题:")
		assert.Contains(t, result.VectorText, "回答:")
	})

	t.Run("移除代码块", func(t *testing.T) {
		turn := &ConversationTurn{
			UserMessages: []*domainCursor.Message{
				{Text: "写一个函数", Type: domainCursor.MessageTypeUser},
			},
			AIMessages: []*domainCursor.Message{
				{
					Text: "这是函数实现：\n```go\nfunc Hello() {\n    fmt.Println(\"hello\")\n}\n```\n这个函数会打印 hello。",
					Type: domainCursor.MessageTypeAI,
				},
			},
		}

		result := extractor.ExtractFromTurn(turn)

		// 验证代码块被移除
		assert.NotContains(t, result.AIResponseCore, "```")
		assert.NotContains(t, result.AIResponseCore, "fmt.Println")
		// 验证自然语言保留
		assert.Contains(t, result.AIResponseCore, "这是函数实现")
		assert.Contains(t, result.AIResponseCore, "打印 hello")
		// 验证代码语言被提取
		assert.Contains(t, result.CodeLanguages, "go")
		assert.True(t, result.HasCode)
	})

	t.Run("提取工具信息", func(t *testing.T) {
		turn := &ConversationTurn{
			UserMessages: []*domainCursor.Message{
				{Text: "创建一个文件", Type: domainCursor.MessageTypeUser},
			},
			AIMessages: []*domainCursor.Message{
				{
					Text: "我来创建文件",
					Type: domainCursor.MessageTypeAI,
					Tools: []*domainCursor.ToolCall{
						{Name: "Write", Arguments: map[string]string{"path": "/path/to/file.go"}},
						{Name: "Read", Arguments: map[string]string{"path": "/other/file.txt"}},
					},
				},
			},
		}

		result := extractor.ExtractFromTurn(turn)

		assert.Contains(t, result.ToolsUsed, "Write")
		assert.Contains(t, result.ToolsUsed, "Read")
		assert.Contains(t, result.FilesModified, "file.go")
		// Read 不应该加入 FilesModified
		assert.NotContains(t, result.FilesModified, "file.txt")
	})

	t.Run("移除系统标签", func(t *testing.T) {
		turn := &ConversationTurn{
			UserMessages: []*domainCursor.Message{
				{Text: "问题", Type: domainCursor.MessageTypeUser},
			},
			AIMessages: []*domainCursor.Message{
				{
					Text: "<think>这是思考过程</think>这是实际回答<context>上下文信息</context>。",
					Type: domainCursor.MessageTypeAI,
				},
			},
		}

		result := extractor.ExtractFromTurn(turn)

		assert.NotContains(t, result.AIResponseCore, "<think>")
		assert.NotContains(t, result.AIResponseCore, "思考过程")
		assert.NotContains(t, result.AIResponseCore, "<context>")
		assert.Contains(t, result.AIResponseCore, "这是实际回答")
	})

	t.Run("截断长文本", func(t *testing.T) {
		// 创建超长文本
		longText := ""
		for i := 0; i < 200; i++ {
			longText += "这是一段很长的文本内容。"
		}

		turn := &ConversationTurn{
			UserMessages: []*domainCursor.Message{
				{Text: longText, Type: domainCursor.MessageTypeUser},
			},
			AIMessages: []*domainCursor.Message{
				{Text: "回答", Type: domainCursor.MessageTypeAI},
			},
		}

		result := extractor.ExtractFromTurn(turn)

		// 验证用户问题被截断
		assert.LessOrEqual(t, len(result.UserQuery), extractor.MaxUserQueryLen+10) // 允许少量超出
	})
}

func TestContentExtractor_RemoveCodeBlocks(t *testing.T) {
	extractor := NewContentExtractor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "单个代码块",
			input:    "前文\n```go\ncode\n```\n后文",
			expected: "前文\n\n后文",
		},
		{
			name:     "多个代码块",
			input:    "```python\ncode1\n```中间```js\ncode2\n```结束",
			expected: "中间结束",
		},
		{
			name:     "无代码块",
			input:    "普通文本",
			expected: "普通文本",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.removeCodeBlocks(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContentExtractor_TruncateAtSentence(t *testing.T) {
	extractor := NewContentExtractor()

	tests := []struct {
		name     string
		input    string
		maxLen   int
		contains string
	}{
		{
			name:     "不需要截断",
			input:    "短文本",
			maxLen:   100,
			contains: "短文本",
		},
		{
			name:     "在句号处截断",
			input:    "第一句。第二句。第三句很长很长很长很长很长。",
			maxLen:   50, // 中文字符占 3 字节，需要更大的 maxLen
			contains: "第一句。",
		},
		{
			name:     "英文句号截断",
			input:    "First sentence. Second sentence. Third sentence is very long.",
			maxLen:   20,
			contains: "First sentence.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.truncateAtSentence(tt.input, tt.maxLen)
			assert.Contains(t, result, tt.contains)
			if len(tt.input) > tt.maxLen {
				assert.LessOrEqual(t, len(result), tt.maxLen+10)
			}
		})
	}
}

func TestContentExtractor_BuildVectorText(t *testing.T) {
	extractor := NewContentExtractor()

	result := &ExtractionResult{
		UserQuery:      "如何配置数据库？",
		AIResponseCore: "可以通过配置文件设置连接参数。",
		ToolsUsed:      []string{"Read", "Write"},
		FilesModified:  []string{"config.yaml", "db.go"},
	}

	vectorText := extractor.buildVectorText(result)

	assert.Contains(t, vectorText, "问题: 如何配置数据库？")
	assert.Contains(t, vectorText, "回答: 可以通过配置文件设置连接参数。")
	assert.Contains(t, vectorText, "操作: Read, Write")
	assert.Contains(t, vectorText, "文件: config.yaml, db.go")
}
