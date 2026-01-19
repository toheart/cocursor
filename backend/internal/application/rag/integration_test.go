package rag

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIncrementalUpdate 测试增量更新逻辑（12.7）
func TestIncrementalUpdate(t *testing.T) {
	// 这个测试需要真实的文件系统和数据库
	// 在实际环境中运行，这里提供测试框架

	t.Run("文件修改时间检测", func(t *testing.T) {
		// 创建临时文件
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		// 写入初始内容
		err := os.WriteFile(testFile, []byte("initial content"), 0644)
		require.NoError(t, err)

		// 获取初始修改时间
		info1, err := os.Stat(testFile)
		require.NoError(t, err)
		initialMtime := info1.ModTime()

		// 等待一小段时间确保时间戳不同
		time.Sleep(100 * time.Millisecond)

		// 修改文件
		err = os.WriteFile(testFile, []byte("updated content"), 0644)
		require.NoError(t, err)

		// 获取新修改时间
		info2, err := os.Stat(testFile)
		require.NoError(t, err)
		newMtime := info2.ModTime()

		// 验证修改时间已更新
		assert.True(t, newMtime.After(initialMtime), "文件修改时间应该更新")
	})

	t.Run("内容哈希检测", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		// 写入初始内容
		err := os.WriteFile(testFile, []byte("content1"), 0644)
		require.NoError(t, err)

		// 计算初始哈希
		scheduler := NewScanScheduler(nil, nil, nil, &ScanConfig{})
		hash1 := scheduler.calculateFileHash(testFile)
		assert.NotEmpty(t, hash1)

		// 修改内容
		err = os.WriteFile(testFile, []byte("content2"), 0644)
		require.NoError(t, err)

		// 计算新哈希
		hash2 := scheduler.calculateFileHash(testFile)
		assert.NotEmpty(t, hash2)
		assert.NotEqual(t, hash1, hash2, "内容改变后哈希应该不同")
	})
}

// TestIncompleteTurnHandling 测试未完成对话对的处理（12.6）
func TestIncompleteTurnHandling(t *testing.T) {
	t.Run("检测未完成对话对", func(t *testing.T) {
		messages := []*domainCursor.Message{
			{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
			{Type: domainCursor.MessageTypeAI, Text: "回答1", Timestamp: 2000},
			{Type: domainCursor.MessageTypeUser, Text: "问题2", Timestamp: 3000},
			// 没有 AI 回复，应该标记为未完成
		}

		turns := PairMessages(messages, "session-1")

		require.Len(t, turns, 2, "应该有 2 个对话对")
		assert.False(t, turns[0].IsIncomplete, "第一个对话对应该已完成")
		assert.True(t, turns[1].IsIncomplete, "最后一个对话对应该未完成")
		assert.Len(t, turns[1].AIMessages, 0, "未完成的对话对不应该有 AI 消息")
	})

	t.Run("后续补充未完成对话对", func(t *testing.T) {
		// 第一次：只有用户消息
		messages1 := []*domainCursor.Message{
			{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
		}

		turns1 := PairMessages(messages1, "session-1")
		require.Len(t, turns1, 1)
		assert.True(t, turns1[0].IsIncomplete, "应该标记为未完成")

		// 第二次：添加了 AI 回复
		messages2 := []*domainCursor.Message{
			{Type: domainCursor.MessageTypeUser, Text: "问题1", Timestamp: 1000},
			{Type: domainCursor.MessageTypeAI, Text: "回答1", Timestamp: 2000},
		}

		turns2 := PairMessages(messages2, "session-1")
		require.Len(t, turns2, 1)
		assert.False(t, turns2[0].IsIncomplete, "添加 AI 回复后应该标记为完成")
		assert.Len(t, turns2[0].AIMessages, 1, "应该有 1 条 AI 消息")
	})
}

// TestEndToEndIndexAndSearch 端到端索引和搜索流程测试（12.5）
// 注意：这是一个集成测试框架，需要真实的 Qdrant 和 Embedding API
func TestEndToEndIndexAndSearch(t *testing.T) {
	// 这个测试需要：
	// 1. 真实的 Qdrant 实例（或嵌入式模式）
	// 2. 真实的 Embedding API（或 mock）
	// 3. 真实的文件系统

	t.Skip("集成测试需要真实环境，跳过单元测试")

	// 测试步骤：
	// 1. 创建测试会话文件
	// 2. 索引会话
	// 3. 执行搜索
	// 4. 验证搜索结果
	// 5. 更新会话内容
	// 6. 重新索引
	// 7. 验证增量更新
}
