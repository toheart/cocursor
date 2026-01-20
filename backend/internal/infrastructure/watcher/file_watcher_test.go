package watcher

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cocursor/backend/internal/domain/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileWatcher_ParseSessionFilePath(t *testing.T) {
	fw := &FileWatcher{}

	tests := []struct {
		name           string
		path           string
		wantSessionID  string
		wantProjectKey string
	}{
		{
			name:           "valid session file",
			path:           "/Users/test/.cursor/projects/Users-test-code-myproject/agent-transcripts/abc123.txt",
			wantSessionID:  "abc123",
			wantProjectKey: "Users-test-code-myproject",
		},
		{
			name:           "session file with uuid",
			path:           "/home/user/.cursor/projects/home-user-workspace/agent-transcripts/550e8400-e29b-41d4-a716-446655440000.txt",
			wantSessionID:  "550e8400-e29b-41d4-a716-446655440000",
			wantProjectKey: "home-user-workspace",
		},
		{
			name:           "non-txt file",
			path:           "/Users/test/.cursor/projects/myproject/agent-transcripts/abc123.json",
			wantSessionID:  "",
			wantProjectKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID, projectKey := fw.parseSessionFilePath(tt.path)
			assert.Equal(t, tt.wantSessionID, sessionID)
			assert.Equal(t, tt.wantProjectKey, projectKey)
		})
	}
}

func TestFileWatcher_IsSessionFile(t *testing.T) {
	fw := &FileWatcher{}

	tests := []struct {
		path     string
		expected bool
	}{
		{"/path/agent-transcripts/session.txt", true},
		{"/path/agent-transcripts/session.json", false},
		{"/path/other/session.txt", false},
		{"/path/agent-transcripts/", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, fw.isSessionFile(tt.path))
		})
	}
}

func TestFileWatcher_Debounce(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "watcher-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建 agent-transcripts 目录
	transcriptsDir := filepath.Join(tmpDir, "test-project", "agent-transcripts")
	require.NoError(t, os.MkdirAll(transcriptsDir, 0755))

	// 创建事件总线
	bus := NewEventBus()
	defer bus.Close()

	// 记录接收到的事件
	var eventCount atomic.Int32
	bus.Subscribe(events.SessionFileModified, events.HandlerFunc(func(event events.Event) error {
		eventCount.Add(1)
		return nil
	}))

	// 创建 FileWatcher
	config := WatchConfig{
		SessionDir:        tmpDir,
		DebounceDelay:     100 * time.Millisecond,
		FullScanThreshold: 24 * time.Hour,
	}

	fw, err := NewFileWatcher(config, bus)
	require.NoError(t, err)

	// 启动监听
	require.NoError(t, fw.Start())
	defer fw.Stop()

	// 等待监听就绪
	time.Sleep(50 * time.Millisecond)

	// 创建测试文件
	testFile := filepath.Join(transcriptsDir, "test-session.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("initial"), 0644))

	// 快速多次写入（应该被防抖合并）
	for i := 0; i < 5; i++ {
		time.Sleep(20 * time.Millisecond)
		require.NoError(t, os.WriteFile(testFile, []byte("update"), 0644))
	}

	// 等待防抖完成
	time.Sleep(300 * time.Millisecond)

	// 应该只收到 1-2 个事件（创建 + 修改被合并）
	count := eventCount.Load()
	assert.LessOrEqual(t, count, int32(2), "events should be debounced")
}

func TestScanMetadata_Persistence(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "metadata-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建元数据管理器
	sm := &ScanMetadata{
		filePath: filepath.Join(tmpDir, "scan_metadata.json"),
	}

	// 设置时间
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	sm.SetLastScanTime(testTime)

	// 创建新实例加载
	sm2 := &ScanMetadata{
		filePath: filepath.Join(tmpDir, "scan_metadata.json"),
	}
	sm2.load()

	// 验证时间相同
	loaded := sm2.GetLastScanTime()
	assert.True(t, loaded.Equal(testTime), "loaded time should match saved time")
}
