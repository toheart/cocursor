package cursor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cocursor/backend/internal/infrastructure/config"
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

// ==================== 新增测试：PathNotFoundError ====================

// TestPathNotFoundError_Error 测试 PathNotFoundError 的 Error() 方法
func TestPathNotFoundError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *PathNotFoundError
		contains []string
	}{
		{
			name: "单个尝试路径",
			err: &PathNotFoundError{
				PathType:      "user_data_dir",
				AttemptedPath: "/path/to/cursor",
				Hint:          "路径不存在",
			},
			contains: []string{"user_data_dir", "/path/to/cursor", "路径不存在"},
		},
		{
			name: "多个尝试路径",
			err: &PathNotFoundError{
				PathType:       "user_data_dir",
				AttemptedPaths: []string{"/path1", "/path2"},
				Hint:           "自动检测失败",
			},
			contains: []string{"user_data_dir", "/path1", "/path2", "自动检测失败"},
		},
		{
			name: "仅提示信息",
			err: &PathNotFoundError{
				PathType: "projects_dir",
				Hint:     "HOME 环境变量未设置",
			},
			contains: []string{"projects_dir", "HOME 环境变量未设置"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			for _, substr := range tt.contains {
				assert.Contains(t, errMsg, substr, "错误信息应包含: %s", substr)
			}
		})
	}
}

// TestIsPathNotFoundError 测试错误类型检测
func TestIsPathNotFoundError(t *testing.T) {
	pnfErr := &PathNotFoundError{PathType: "test"}
	genericErr := os.ErrNotExist

	assert.True(t, IsPathNotFoundError(pnfErr))
	assert.False(t, IsPathNotFoundError(genericErr))
	assert.False(t, IsPathNotFoundError(nil))
}

// TestAsPathNotFoundError 测试错误类型转换
func TestAsPathNotFoundError(t *testing.T) {
	pnfErr := &PathNotFoundError{PathType: "test", Hint: "test hint"}

	// 成功转换
	converted, ok := AsPathNotFoundError(pnfErr)
	assert.True(t, ok)
	assert.Equal(t, "test", converted.PathType)
	assert.Equal(t, "test hint", converted.Hint)

	// 转换失败
	_, ok = AsPathNotFoundError(os.ErrNotExist)
	assert.False(t, ok)
}

// ==================== 新增测试：自定义路径配置 ====================

// TestNewPathResolverWithConfig 测试使用自定义配置创建 PathResolver
func TestNewPathResolverWithConfig(t *testing.T) {
	resolver := NewPathResolverWithConfig("/custom/user/data", "/custom/projects")

	assert.Equal(t, "/custom/user/data", resolver.customUserDataDir)
	assert.Equal(t, "/custom/projects", resolver.customProjectsDir)
}

// TestPathResolver_CustomUserDataDir 测试自定义 UserDataDir 优先级
func TestPathResolver_CustomUserDataDir(t *testing.T) {
	// 创建临时目录模拟 Cursor 用户数据目录
	tmpDir := t.TempDir()
	globalStorageDir := filepath.Join(tmpDir, "globalStorage")
	require.NoError(t, os.MkdirAll(globalStorageDir, 0755))

	resolver := NewPathResolverWithConfig(tmpDir, "")

	userDataDir, err := resolver.getUserDataDir()
	require.NoError(t, err)
	assert.Equal(t, tmpDir, userDataDir)
}

// TestPathResolver_CustomUserDataDir_NotExists 测试自定义路径不存在的情况
func TestPathResolver_CustomUserDataDir_NotExists(t *testing.T) {
	resolver := NewPathResolverWithConfig("/nonexistent/path", "")

	_, err := resolver.getUserDataDir()
	require.Error(t, err)
	assert.True(t, IsPathNotFoundError(err))

	pnfErr, ok := AsPathNotFoundError(err)
	require.True(t, ok)
	assert.Equal(t, "user_data_dir", pnfErr.PathType)
	assert.True(t, pnfErr.IsCustom)
	assert.Equal(t, "/nonexistent/path", pnfErr.AttemptedPath)
}

// TestPathResolver_CustomProjectsDir 测试自定义 ProjectsDir 优先级
func TestPathResolver_CustomProjectsDir(t *testing.T) {
	tmpDir := t.TempDir()

	resolver := NewPathResolverWithConfig("", tmpDir)

	projectsDir, err := resolver.GetCursorProjectsDir()
	require.NoError(t, err)
	assert.Equal(t, tmpDir, projectsDir)
}

// TestPathResolver_CustomProjectsDir_NotExists 测试自定义项目路径不存在的情况
func TestPathResolver_CustomProjectsDir_NotExists(t *testing.T) {
	resolver := NewPathResolverWithConfig("", "/nonexistent/projects")

	_, err := resolver.GetCursorProjectsDir()
	require.Error(t, err)
	assert.True(t, IsPathNotFoundError(err))

	pnfErr, ok := AsPathNotFoundError(err)
	require.True(t, ok)
	assert.Equal(t, "projects_dir", pnfErr.PathType)
	assert.True(t, pnfErr.IsCustom)
}

// ==================== 新增测试：全局配置 ====================

// TestGlobalCursorConfig 测试全局配置设置和获取
func TestGlobalCursorConfig(t *testing.T) {
	// 保存原始配置
	originalConfig := GetGlobalCursorConfig()
	defer SetGlobalCursorConfig(originalConfig)

	// 设置新配置
	newConfig := &config.CursorConfig{
		UserDataDir: "/global/user/data",
		ProjectsDir: "/global/projects",
	}
	SetGlobalCursorConfig(newConfig)

	// 验证配置已设置
	cfg := GetGlobalCursorConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, "/global/user/data", cfg.UserDataDir)
	assert.Equal(t, "/global/projects", cfg.ProjectsDir)
}

// TestPathResolver_GlobalConfig_UserDataDir 测试全局配置对 UserDataDir 的影响
func TestPathResolver_GlobalConfig_UserDataDir(t *testing.T) {
	// 保存原始配置
	originalConfig := GetGlobalCursorConfig()
	defer SetGlobalCursorConfig(originalConfig)

	// 创建临时目录
	tmpDir := t.TempDir()
	globalStorageDir := filepath.Join(tmpDir, "globalStorage")
	require.NoError(t, os.MkdirAll(globalStorageDir, 0755))

	// 设置全局配置
	SetGlobalCursorConfig(&config.CursorConfig{
		UserDataDir: tmpDir,
	})

	resolver := NewPathResolver()
	userDataDir, err := resolver.getUserDataDir()
	require.NoError(t, err)
	assert.Equal(t, tmpDir, userDataDir)
}

// TestPathResolver_GlobalConfig_ProjectsDir 测试全局配置对 ProjectsDir 的影响
func TestPathResolver_GlobalConfig_ProjectsDir(t *testing.T) {
	// 保存原始配置
	originalConfig := GetGlobalCursorConfig()
	defer SetGlobalCursorConfig(originalConfig)

	// 创建临时目录
	tmpDir := t.TempDir()

	// 设置全局配置
	SetGlobalCursorConfig(&config.CursorConfig{
		ProjectsDir: tmpDir,
	})

	resolver := NewPathResolver()
	projectsDir, err := resolver.GetCursorProjectsDir()
	require.NoError(t, err)
	assert.Equal(t, tmpDir, projectsDir)
}

// ==================== 新增测试：环境变量配置 ====================

// TestPathResolver_EnvVar_UserDataDir 测试环境变量 CURSOR_USER_DATA_DIR
func TestPathResolver_EnvVar_UserDataDir(t *testing.T) {
	// 保存原始值
	originalValue := os.Getenv("CURSOR_USER_DATA_DIR")
	originalConfig := GetGlobalCursorConfig()
	defer func() {
		if originalValue == "" {
			os.Unsetenv("CURSOR_USER_DATA_DIR")
		} else {
			os.Setenv("CURSOR_USER_DATA_DIR", originalValue)
		}
		SetGlobalCursorConfig(originalConfig)
	}()

	// 清除全局配置
	SetGlobalCursorConfig(nil)

	// 创建临时目录
	tmpDir := t.TempDir()
	globalStorageDir := filepath.Join(tmpDir, "globalStorage")
	require.NoError(t, os.MkdirAll(globalStorageDir, 0755))

	// 设置环境变量
	os.Setenv("CURSOR_USER_DATA_DIR", tmpDir)

	resolver := NewPathResolver()
	userDataDir, err := resolver.getUserDataDir()
	require.NoError(t, err)
	assert.Equal(t, tmpDir, userDataDir)
}

// TestPathResolver_EnvVar_UserDataDir_NotExists 测试环境变量指定的路径不存在
func TestPathResolver_EnvVar_UserDataDir_NotExists(t *testing.T) {
	// 保存原始值
	originalValue := os.Getenv("CURSOR_USER_DATA_DIR")
	originalConfig := GetGlobalCursorConfig()
	defer func() {
		if originalValue == "" {
			os.Unsetenv("CURSOR_USER_DATA_DIR")
		} else {
			os.Setenv("CURSOR_USER_DATA_DIR", originalValue)
		}
		SetGlobalCursorConfig(originalConfig)
	}()

	// 清除全局配置
	SetGlobalCursorConfig(nil)

	// 设置不存在的路径
	os.Setenv("CURSOR_USER_DATA_DIR", "/nonexistent/env/path")

	resolver := NewPathResolver()
	_, err := resolver.getUserDataDir()
	require.Error(t, err)
	assert.True(t, IsPathNotFoundError(err))

	pnfErr, ok := AsPathNotFoundError(err)
	require.True(t, ok)
	assert.True(t, pnfErr.IsCustom)
	assert.Contains(t, pnfErr.Hint, "CURSOR_USER_DATA_DIR")
}

// TestPathResolver_EnvVar_ProjectsDir 测试环境变量 CURSOR_PROJECTS_DIR
func TestPathResolver_EnvVar_ProjectsDir(t *testing.T) {
	// 保存原始值
	originalValue := os.Getenv("CURSOR_PROJECTS_DIR")
	originalConfig := GetGlobalCursorConfig()
	defer func() {
		if originalValue == "" {
			os.Unsetenv("CURSOR_PROJECTS_DIR")
		} else {
			os.Setenv("CURSOR_PROJECTS_DIR", originalValue)
		}
		SetGlobalCursorConfig(originalConfig)
	}()

	// 清除全局配置
	SetGlobalCursorConfig(nil)

	// 创建临时目录
	tmpDir := t.TempDir()

	// 设置环境变量
	os.Setenv("CURSOR_PROJECTS_DIR", tmpDir)

	resolver := NewPathResolver()
	projectsDir, err := resolver.GetCursorProjectsDir()
	require.NoError(t, err)
	assert.Equal(t, tmpDir, projectsDir)
}

// ==================== 新增测试：优先级测试 ====================

// TestPathResolver_Priority_UserDataDir 测试 UserDataDir 配置优先级
// 优先级：实例配置 > 全局配置 > 环境变量 > 自动检测
func TestPathResolver_Priority_UserDataDir(t *testing.T) {
	// 保存原始值
	originalEnv := os.Getenv("CURSOR_USER_DATA_DIR")
	originalConfig := GetGlobalCursorConfig()
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("CURSOR_USER_DATA_DIR")
		} else {
			os.Setenv("CURSOR_USER_DATA_DIR", originalEnv)
		}
		SetGlobalCursorConfig(originalConfig)
	}()

	// 创建多个临时目录
	instanceDir := t.TempDir()
	globalDir := t.TempDir()
	envDir := t.TempDir()

	// 设置环境变量
	os.Setenv("CURSOR_USER_DATA_DIR", envDir)

	// 设置全局配置
	SetGlobalCursorConfig(&config.CursorConfig{
		UserDataDir: globalDir,
	})

	// 测试1：实例配置优先
	resolver := NewPathResolverWithConfig(instanceDir, "")
	userDataDir, err := resolver.getUserDataDir()
	require.NoError(t, err)
	assert.Equal(t, instanceDir, userDataDir, "实例配置应优先")

	// 测试2：全局配置次之
	resolver = NewPathResolver()
	userDataDir, err = resolver.getUserDataDir()
	require.NoError(t, err)
	assert.Equal(t, globalDir, userDataDir, "全局配置应次之")

	// 测试3：环境变量再次之
	SetGlobalCursorConfig(nil)
	resolver = NewPathResolver()
	userDataDir, err = resolver.getUserDataDir()
	require.NoError(t, err)
	assert.Equal(t, envDir, userDataDir, "环境变量应再次之")
}

// ==================== 新增测试：GetPathStatus ====================

// TestPathResolver_GetPathStatus 测试获取路径状态
func TestPathResolver_GetPathStatus(t *testing.T) {
	// 保存原始配置
	originalConfig := GetGlobalCursorConfig()
	defer SetGlobalCursorConfig(originalConfig)

	// 创建临时目录
	userDataDir := t.TempDir()
	projectsDir := t.TempDir()

	// 创建 globalStorage 子目录
	globalStorageDir := filepath.Join(userDataDir, "globalStorage")
	require.NoError(t, os.MkdirAll(globalStorageDir, 0755))

	// 设置全局配置
	SetGlobalCursorConfig(&config.CursorConfig{
		UserDataDir: userDataDir,
		ProjectsDir: projectsDir,
	})

	resolver := NewPathResolver()
	status := resolver.GetPathStatus()

	// 验证状态
	assert.True(t, status.UserDataDirOK, "UserDataDir 应该正常")
	assert.True(t, status.ProjectsDirOK, "ProjectsDir 应该正常")
	assert.Equal(t, userDataDir, status.UserDataDir)
	assert.Equal(t, projectsDir, status.ProjectsDir)
	assert.Nil(t, status.UserDataDirError)
	assert.Nil(t, status.ProjectsDirError)

	// 验证配置信息
	assert.Equal(t, userDataDir, status.ConfigUserDataDir)
	assert.Equal(t, projectsDir, status.ConfigProjectsDir)
}

// TestPathResolver_GetPathStatus_WithErrors 测试路径不存在时的状态
func TestPathResolver_GetPathStatus_WithErrors(t *testing.T) {
	// 保存原始配置
	originalConfig := GetGlobalCursorConfig()
	originalEnvUserData := os.Getenv("CURSOR_USER_DATA_DIR")
	originalEnvProjects := os.Getenv("CURSOR_PROJECTS_DIR")
	defer func() {
		SetGlobalCursorConfig(originalConfig)
		if originalEnvUserData == "" {
			os.Unsetenv("CURSOR_USER_DATA_DIR")
		} else {
			os.Setenv("CURSOR_USER_DATA_DIR", originalEnvUserData)
		}
		if originalEnvProjects == "" {
			os.Unsetenv("CURSOR_PROJECTS_DIR")
		} else {
			os.Setenv("CURSOR_PROJECTS_DIR", originalEnvProjects)
		}
	}()

	// 清除环境变量
	os.Unsetenv("CURSOR_USER_DATA_DIR")
	os.Unsetenv("CURSOR_PROJECTS_DIR")

	// 设置不存在的路径
	SetGlobalCursorConfig(&config.CursorConfig{
		UserDataDir: "/nonexistent/user/data",
		ProjectsDir: "/nonexistent/projects",
	})

	resolver := NewPathResolver()
	status := resolver.GetPathStatus()

	// 验证状态
	assert.False(t, status.UserDataDirOK, "UserDataDir 应该失败")
	assert.False(t, status.ProjectsDirOK, "ProjectsDir 应该失败")
	assert.NotNil(t, status.UserDataDirError)
	assert.NotNil(t, status.ProjectsDirError)
	assert.Equal(t, "user_data_dir", status.UserDataDirError.PathType)
	assert.Equal(t, "projects_dir", status.ProjectsDirError.PathType)
}

// ==================== 新增测试：WSL 检测 ====================

// TestPathResolver_IsWSL 测试 WSL 检测
func TestPathResolver_IsWSL(t *testing.T) {
	resolver := NewPathResolver()
	isWSL := resolver.isWSL()

	// 在非 WSL 环境下应该返回 false
	// 这个测试主要验证方法不会 panic
	t.Logf("isWSL: %v", isWSL)

	// 如果设置了 WSL 环境变量，应该返回 true
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSLENV") != "" {
		assert.True(t, isWSL, "在 WSL 环境中应返回 true")
	}
}

// TestPathResolver_WindowsPathToWSL 测试 Windows 路径转换为 WSL 路径
func TestPathResolver_WindowsPathToWSL(t *testing.T) {
	resolver := NewPathResolver()

	tests := []struct {
		name        string
		windowsPath string
		expected    string
	}{
		{
			name:        "标准 Windows 路径",
			windowsPath: "C:\\Users\\test",
			expected:    "/mnt/c/Users/test",
		},
		{
			name:        "带正斜杠的 Windows 路径",
			windowsPath: "C:/Users/test",
			expected:    "/mnt/c/Users/test",
		},
		{
			name:        "D 盘路径",
			windowsPath: "D:\\code\\project",
			expected:    "/mnt/d/code/project",
		},
		{
			name:        "大写盘符",
			windowsPath: "E:\\data",
			expected:    "/mnt/e/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.windowsPathToWSL(tt.windowsPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== 新增测试：getPathConfigHint ====================

// TestPathResolver_GetPathConfigHint 测试路径配置提示
func TestPathResolver_GetPathConfigHint(t *testing.T) {
	resolver := NewPathResolver()

	// 测试 user_data_dir 提示
	hint := resolver.getPathConfigHint("user_data_dir")
	assert.NotEmpty(t, hint)
	assert.Contains(t, hint, "CURSOR_USER_DATA_DIR")

	// 测试 projects_dir 提示
	hint = resolver.getPathConfigHint("projects_dir")
	assert.NotEmpty(t, hint)
	assert.Contains(t, hint, "CURSOR_PROJECTS_DIR")
}

// ==================== 新增测试：GetCursorProjectsDirOrDefault ====================

// TestPathResolver_GetCursorProjectsDirOrDefault 测试获取项目目录（带默认值）
func TestPathResolver_GetCursorProjectsDirOrDefault(t *testing.T) {
	// 保存原始配置
	originalConfig := GetGlobalCursorConfig()
	defer SetGlobalCursorConfig(originalConfig)

	// 创建临时目录
	tmpDir := t.TempDir()

	// 设置全局配置
	SetGlobalCursorConfig(&config.CursorConfig{
		ProjectsDir: tmpDir,
	})

	resolver := NewPathResolver()
	projectsDir := resolver.GetCursorProjectsDirOrDefault()

	assert.Equal(t, tmpDir, projectsDir)
}

// TestPathResolver_GetCursorProjectsDirOrDefault_Fallback 测试回退到默认路径
func TestPathResolver_GetCursorProjectsDirOrDefault_Fallback(t *testing.T) {
	// 保存原始配置
	originalConfig := GetGlobalCursorConfig()
	originalEnv := os.Getenv("CURSOR_PROJECTS_DIR")
	defer func() {
		SetGlobalCursorConfig(originalConfig)
		if originalEnv == "" {
			os.Unsetenv("CURSOR_PROJECTS_DIR")
		} else {
			os.Setenv("CURSOR_PROJECTS_DIR", originalEnv)
		}
	}()

	// 清除配置
	SetGlobalCursorConfig(nil)
	os.Unsetenv("CURSOR_PROJECTS_DIR")

	resolver := NewPathResolver()
	projectsDir := resolver.GetCursorProjectsDirOrDefault()

	// 应该返回默认路径（不报错）
	// 默认路径格式：~/.cursor/projects
	homeDir, _ := os.UserHomeDir()
	expectedDefault := filepath.Join(homeDir, ".cursor", "projects")
	assert.Equal(t, expectedDefault, projectsDir)
}
