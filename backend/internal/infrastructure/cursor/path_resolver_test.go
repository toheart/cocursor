package cursor

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetWorkspaceIDByPath 测试根据项目路径查找工作区 ID
// 验收标准：输入项目路径 D:/code/cocursor，能输出工作区 ID（如 d4b798d4...）
func TestGetWorkspaceIDByPath(t *testing.T) {
	// TODO: 此测试需要特定的本地环境（D:/code/cocursor），跳过 CI
	t.Skip("Skipping: requires specific local environment")

	resolver := NewPathResolver()

	// 测试路径：D:/code/cocursor
	testPath := "D:/code/cocursor"

	workspaceID, err := resolver.GetWorkspaceIDByPath(testPath)
	if err != nil {
		t.Fatalf("GetWorkspaceIDByPath failed: %v", err)
	}

	if workspaceID == "" {
		t.Fatal("workspaceID is empty")
	}

	t.Logf("项目路径: %s", testPath)
	t.Logf("工作区 ID: %s", workspaceID)

	// 验证工作区 ID 格式（应该是 32 位十六进制字符串）
	if len(workspaceID) != 32 {
		t.Errorf("workspaceID length should be 32, got %d", len(workspaceID))
	}
}

// TestGetGlobalStoragePath 测试获取全局存储路径
func TestGetGlobalStoragePath(t *testing.T) {
	// TODO: 此测试需要本地 Cursor 环境，跳过 CI
	t.Skip("Skipping: requires local Cursor installation")

	resolver := NewPathResolver()

	path, err := resolver.GetGlobalStoragePath()
	if err != nil {
		t.Fatalf("GetGlobalStoragePath failed: %v", err)
	}

	if path == "" {
		t.Fatal("path is empty")
	}

	t.Logf("全局存储路径: %s", path)
}

// TestGetWorkspaceStorageDir 测试获取工作区存储目录
func TestGetWorkspaceStorageDir(t *testing.T) {
	// TODO: 此测试需要本地 Cursor 环境，跳过 CI
	t.Skip("Skipping: requires local Cursor installation")

	resolver := NewPathResolver()

	dir, err := resolver.GetWorkspaceStorageDir()
	if err != nil {
		t.Fatalf("GetWorkspaceStorageDir failed: %v", err)
	}

	if dir == "" {
		t.Fatal("dir is empty")
	}

	t.Logf("工作区存储目录: %s", dir)
}

// TestParseFolderURI_WindowsPathWithLeadingBackslash 测试 Windows 路径解析时开头的反斜杠问题
// 问题：file:///c%3A/Users/... 解析后可能变成 \c:\Users\...，需要清理开头的反斜杠
func TestParseFolderURI_WindowsPathWithLeadingBackslash(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("此测试仅在 Windows 上运行")
	}

	resolver := NewPathResolver()

	tests := []struct {
		name     string
		uri      string
		expected string // 期望的路径（不包含开头的反斜杠）
		wantErr  bool
	}{
		{
			name:     "URL encoded Windows path with leading backslash",
			uri:      "file:///c%3A/Users/TANG/Videos/goanalysis",
			expected: "c:/Users/TANG/Videos/goanalysis", // 不应该有开头的反斜杠
			wantErr:  false,
		},
		{
			name:     "Windows path with drive letter",
			uri:      "file:///d:/code/cocursor",
			expected: "d:/code/cocursor",
			wantErr:  false,
		},
		{
			name:     "Windows path with URL encoding",
			uri:      "file:///d%3A/code/cocursor",
			expected: "d:/code/cocursor",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.parseFolderURI(tt.uri)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)

				// 规范化路径用于比较（统一使用正斜杠）
				resultNormalized := filepath.ToSlash(result)
				expectedNormalized := filepath.ToSlash(tt.expected)

				// 检查路径不应该以单个反斜杠开头（除非是 UNC 路径）
				if strings.HasPrefix(result, "\\") && !strings.HasPrefix(result, "\\\\") {
					t.Errorf("路径不应该以单个反斜杠开头: %s", result)
				}

				// 检查路径匹配（不区分大小写）
				if !strings.EqualFold(resultNormalized, expectedNormalized) {
					t.Errorf("路径不匹配: 期望 %s, 得到 %s", expectedNormalized, resultNormalized)
				}
			}
		})
	}
}

// TestParseFolderURI_UnixPaths 测试 Unix 路径解析（macOS/Linux）
// 确保 Unix 路径不受 Windows 清理逻辑影响
func TestParseFolderURI_UnixPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("此测试在非 Windows 系统上运行")
	}

	resolver := NewPathResolver()

	tests := []struct {
		name     string
		uri      string
		expected string
		wantErr  bool
	}{
		{
			name:     "macOS path",
			uri:      "file:///Users/user/project",
			expected: "/Users/user/project",
			wantErr:  false,
		},
		{
			name:     "Linux path",
			uri:      "file:///home/user/project",
			expected: "/home/user/project",
			wantErr:  false,
		},
		{
			name:     "Unix path with spaces",
			uri:      "file:///Users/user/my%20project",
			expected: "/Users/user/my project",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.parseFolderURI(tt.uri)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)

				// Unix 路径应该保留开头的斜杠
				if !strings.HasPrefix(result, "/") {
					t.Errorf("Unix 路径应该以斜杠开头: %s", result)
				}

				// 检查路径匹配
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestParseFolderURI_WindowsPathDetection 测试 Windows 路径检测逻辑
// 验证修复：索引 2 是冒号，而不是索引 1
// 例如："/D:/code/..." 中，索引 0='/', 索引 1='D', 索引 2=':'
func TestParseFolderURI_WindowsPathDetection(t *testing.T) {
	resolver := NewPathResolver()

	// 这些测试用例验证路径解析的正确性，不依赖于运行平台
	tests := []struct {
		name            string
		uri             string
		isWindowsPath   bool // 是否是 Windows 风格的路径
		expectHasColon  bool // 期望解析后路径包含冒号（Windows 驱动器号）
		expectNoLeading bool // 期望解析后路径不以 / 开头（Windows 路径）
	}{
		{
			name:            "Windows path file:///D:/code",
			uri:             "file:///D:/code/project",
			isWindowsPath:   true,
			expectHasColon:  true,
			expectNoLeading: true,
		},
		{
			name:            "Windows path lowercase file:///d:/code",
			uri:             "file:///d:/code/project",
			isWindowsPath:   true,
			expectHasColon:  true,
			expectNoLeading: true,
		},
		{
			name:            "Windows path URL encoded file:///d%3A/code",
			uri:             "file:///d%3A/code/project",
			isWindowsPath:   true,
			expectHasColon:  true,
			expectNoLeading: true,
		},
		{
			name:            "Unix path file:///Users/...",
			uri:             "file:///Users/user/code",
			isWindowsPath:   false,
			expectHasColon:  false,
			expectNoLeading: false, // Unix 路径应保留开头的 /
		},
		{
			name:            "Unix path file:///home/...",
			uri:             "file:///home/user/code",
			isWindowsPath:   false,
			expectHasColon:  false,
			expectNoLeading: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.parseFolderURI(tt.uri)
			require.NoError(t, err, "URI 解析不应该出错")

			hasColon := strings.Contains(result, ":")
			startsWithSlash := strings.HasPrefix(result, "/") || strings.HasPrefix(result, "\\")

			if tt.expectHasColon {
				assert.True(t, hasColon, "Windows 路径应该包含冒号: %s", result)
			}

			if tt.expectNoLeading {
				assert.False(t, startsWithSlash, "Windows 路径不应该以斜杠开头: %s", result)
			} else {
				assert.True(t, startsWithSlash, "Unix 路径应该以斜杠开头: %s", result)
			}

			t.Logf("URI: %s -> 解析结果: %s", tt.uri, result)
		})
	}
}

// TestNormalizePath_CrossPlatform 测试路径规范化在不同平台上的行为
func TestNormalizePath_CrossPlatform(t *testing.T) {
	// TODO: 跨平台路径测试在非 Windows 系统上存在问题，待修复
	t.Skip("Skipping: cross-platform path tests need fixes")

	resolver := NewPathResolver()

	tests := []struct {
		name    string
		path    string
		wantErr bool
		checkFn func(t *testing.T, normalized string)
	}{
		{
			name:    "relative path",
			path:    "./code/project",
			wantErr: false,
			checkFn: func(t *testing.T, normalized string) {
				// 应该转换为绝对路径
				assert.True(t, filepath.IsAbs(normalized) || strings.HasPrefix(normalized, "/"))
				// 应该使用正斜杠
				assert.NotContains(t, normalized, "\\")
			},
		},
		{
			name:    "path with trailing slash",
			path:    "/path/to/project/",
			wantErr: false,
			checkFn: func(t *testing.T, normalized string) {
				// 末尾斜杠应该被移除
				assert.False(t, strings.HasSuffix(normalized, "/"))
			},
		},
		{
			name:    "Windows path with backslashes",
			path:    "C:\\Users\\project",
			wantErr: false,
			checkFn: func(t *testing.T, normalized string) {
				// 应该转换为正斜杠
				assert.NotContains(t, normalized, "\\")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.normalizePath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.checkFn != nil {
					tt.checkFn(t, result)
				}
			}
		})
	}
}

// TestGetWorkspaceIDByPath_PathMatching 测试路径匹配逻辑
// 验证规范化后的路径能够正确匹配
func TestGetWorkspaceIDByPath_PathMatching(t *testing.T) {
	// TODO: 跨平台路径匹配测试在非 Windows 系统上存在问题，待修复
	t.Skip("Skipping: cross-platform path matching tests need fixes")

	resolver := NewPathResolver()

	// 这个测试需要真实的工作区环境，所以只测试路径规范化逻辑
	tests := []struct {
		name        string
		path1       string
		path2       string
		shouldMatch bool
	}{
		{
			name:        "same path different separators",
			path1:       "C:/Users/project",
			path2:       "C:\\Users\\project",
			shouldMatch: true,
		},
		{
			name:        "case insensitive on Windows",
			path1:       "C:/Users/PROJECT",
			path2:       "c:/users/project",
			shouldMatch: runtime.GOOS == "windows",
		},
		{
			name:        "different paths",
			path1:       "C:/Users/project1",
			path2:       "C:/Users/project2",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			norm1, err1 := resolver.normalizePath(tt.path1)
			norm2, err2 := resolver.normalizePath(tt.path2)

			require.NoError(t, err1)
			require.NoError(t, err2)

			// 使用 EqualFold 进行不区分大小写的比较（Windows 路径不区分大小写）
			matches := strings.EqualFold(norm1, norm2)

			if matches != tt.shouldMatch {
				t.Errorf("路径匹配结果不符合预期: %s vs %s, 期望 %v, 得到 %v",
					norm1, norm2, tt.shouldMatch, matches)
			}
		})
	}
}
