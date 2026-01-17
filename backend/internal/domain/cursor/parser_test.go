package cursor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseComposerData(t *testing.T) {
	// 使用文档中的实际 JSON 示例
	rawJSON := `{
		"allComposers": [
			{
				"type": "head",
				"composerId": "03e41bc4-c700-491a-a01a-2048468bd7b8",
				"name": "Golang 后端服务框架",
				"lastUpdatedAt": 1768656147170,
				"createdAt": 1768650378012,
				"unifiedMode": "agent",
				"forceMode": "edit",
				"contextUsagePercent": 65.186,
				"totalLinesAdded": 1704,
				"totalLinesRemoved": 116,
				"filesChangedCount": 39,
				"subtitle": "DAEMON_MANAGER.md, extension.ts, daemonManager.ts, lock_test.go, SINGLETON_LOCK.md",
				"createdOnBranch": "main"
			},
			{
				"type": "head",
				"composerId": "f2dc90df-54c2-4bc6-abb0-8ac0c92d704c",
				"name": "图标优化",
				"lastUpdatedAt": 1768651830147,
				"createdAt": 1768651820874,
				"unifiedMode": "agent",
				"forceMode": "edit",
				"contextUsagePercent": 7.094,
				"totalLinesAdded": 20,
				"totalLinesRemoved": 5,
				"filesChangedCount": 1,
				"subtitle": "icon.svg",
				"createdOnBranch": "main"
			}
		]
	}`

	composers, err := ParseComposerData(rawJSON)
	require.NoError(t, err)
	require.Len(t, composers, 2)

	// 验证第一个会话的 filesChangedCount
	first := composers[0]
	assert.Equal(t, 39, first.FilesChangedCount, "应该正确提取 filesChangedCount")
	assert.Equal(t, "Golang 后端服务框架", first.Name)
	assert.Equal(t, 65.186, first.ContextUsagePercent)
	assert.Equal(t, 1704, first.TotalLinesAdded)
	assert.Equal(t, 116, first.TotalLinesRemoved)

	// 验证第二个会话的 filesChangedCount
	second := composers[1]
	assert.Equal(t, 1, second.FilesChangedCount, "应该正确提取 filesChangedCount")
	assert.Equal(t, "图标优化", second.Name)
	assert.Equal(t, 7.094, second.ContextUsagePercent)
	assert.Equal(t, 20, second.TotalLinesAdded)
	assert.Equal(t, 5, second.TotalLinesRemoved)
}

func TestParseComposerData_EmptyJSON(t *testing.T) {
	_, err := ParseComposerData("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "raw JSON is empty")
}

func TestParseComposerData_InvalidJSON(t *testing.T) {
	_, err := ParseComposerData("invalid json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
}

func TestGetActiveComposer(t *testing.T) {
	composers := []ComposerData{
		{
			ComposerID:    "1",
			LastUpdatedAt: 1000,
			IsArchived:    true,
		},
		{
			ComposerID:    "2",
			LastUpdatedAt: 2000,
			IsArchived:    false,
		},
		{
			ComposerID:    "3",
			LastUpdatedAt: 3000,
			IsArchived:    false,
		},
	}

	active := GetActiveComposer(composers)
	require.NotNil(t, active)
	assert.Equal(t, "3", active.ComposerID, "应该返回最近更新的活跃会话")
}

func TestGetActiveComposer_AllArchived(t *testing.T) {
	composers := []ComposerData{
		{
			ComposerID:    "1",
			LastUpdatedAt: 1000,
			IsArchived:    true,
		},
	}

	active := GetActiveComposer(composers)
	assert.Nil(t, active, "如果所有会话都已归档，应该返回 nil")
}

func TestGetActiveComposer_Empty(t *testing.T) {
	active := GetActiveComposer([]ComposerData{})
	assert.Nil(t, active, "如果列表为空，应该返回 nil")
}
