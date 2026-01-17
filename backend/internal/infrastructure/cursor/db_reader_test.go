package cursor

import (
	"testing"
)

// TestReadValueFromTable 测试从数据库读取值
// 验收标准：能成功读取 cursorAuth/cachedEmail 对应的 BLOB 数据（Email）
func TestReadValueFromTable(t *testing.T) {
	// 创建路径解析器和数据库读取器
	pathResolver := NewPathResolver()
	dbReader := NewDBReader()

	// 获取全局存储数据库路径
	dbPath, err := pathResolver.GetGlobalStoragePath()
	if err != nil {
		t.Fatalf("GetGlobalStoragePath failed: %v", err)
	}

	// 测试读取 cursorAuth/cachedEmail
	key := "cursorAuth/cachedEmail"
	value, err := dbReader.ReadValueFromTable(dbPath, key)
	if err != nil {
		t.Fatalf("ReadValueFromTable failed: %v", err)
	}

	if len(value) == 0 {
		t.Fatal("value is empty")
	}

	email := string(value)
	t.Logf("读取的键: %s", key)
	t.Logf("读取的值（Email）: %s", email)
}

// TestReadDailyStats 测试读取每日统计数据
func TestReadDailyStats(t *testing.T) {
	pathResolver := NewPathResolver()
	dbReader := NewDBReader()

	// 获取全局存储数据库路径
	dbPath, err := pathResolver.GetGlobalStoragePath()
	if err != nil {
		t.Fatalf("GetGlobalStoragePath failed: %v", err)
	}

	// 测试读取今日统计数据（需要根据实际日期调整）
	// 这里使用一个示例日期，实际使用时应该使用当前日期
	key := "aiCodeTracking.dailyStats.v1.5.2026-01-17"
	value, err := dbReader.ReadValueFromTable(dbPath, key)
	if err != nil {
		// 如果读取失败（可能是该日期没有数据），只记录警告
		t.Logf("无法读取统计数据（可能该日期没有数据）: %v", err)
		return
	}

	if len(value) == 0 {
		t.Fatal("value is empty")
	}

	t.Logf("读取的键: %s", key)
	t.Logf("读取的值: %s", string(value))
}
