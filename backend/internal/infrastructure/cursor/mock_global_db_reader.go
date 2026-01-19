package cursor

import (
	"fmt"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
)

// MockGlobalDBReader Mock Global 数据库读取器（用于测试）
type MockGlobalDBReader struct{}

// NewMockGlobalDBReader 创建 Mock GlobalDBReader 实例
func NewMockGlobalDBReader() domainCursor.GlobalDBReader {
	return &MockGlobalDBReader{}
}

// ReadValue 从 Global 数据库读取指定键的值（Mock 实现，总是返回错误）
func (m *MockGlobalDBReader) ReadValue(key string) ([]byte, error) {
	// Mock 实现：返回 key not found 错误
	return nil, fmt.Errorf("key not found: %s", key)
}
