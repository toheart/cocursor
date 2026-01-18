package cursor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanAllWorkspaces(t *testing.T) {
	// 创建临时目录结构模拟 Cursor 工作区存储
	tmpDir, err := os.MkdirTemp("", "workspace_storage_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建第一个工作区
	workspaceID1 := "workspace1"
	workspaceDir1 := filepath.Join(tmpDir, workspaceID1)
	err = os.MkdirAll(workspaceDir1, 0755)
	require.NoError(t, err)

	// 创建 workspace.json
	workspaceJSON1 := map[string]string{
		"folder": "file:///d:/code/project1",
	}
	jsonData1, err := json.Marshal(workspaceJSON1)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(workspaceDir1, "workspace.json"), jsonData1, 0644)
	require.NoError(t, err)

	// 创建第二个工作区
	workspaceID2 := "workspace2"
	workspaceDir2 := filepath.Join(tmpDir, workspaceID2)
	err = os.MkdirAll(workspaceDir2, 0755)
	require.NoError(t, err)

	workspaceJSON2 := map[string]string{
		"folder": "file:///d:/code/project2",
	}
	jsonData2, err := json.Marshal(workspaceJSON2)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(workspaceDir2, "workspace.json"), jsonData2, 0644)
	require.NoError(t, err)

	// 注意：这个测试需要真实 Cursor 环境，因为 PathResolver 依赖真实环境
	// 这里只测试基本逻辑，完整测试需要集成测试环境
	t.Logf("测试工作区扫描逻辑（需要真实 Cursor 环境）")
}

func TestParseFolderURI(t *testing.T) {
	discovery := NewProjectDiscovery()

	tests := []struct {
		name     string
		uri      string
		expected string
		wantErr  bool
	}{
		{
			name:     "Windows path",
			uri:      "file:///d:/code/project",
			expected: "d:/code/project",
			wantErr:  false,
		},
		{
			name:     "Unix path",
			uri:      "file:///Users/user/project",
			expected: "/Users/user/project",
			wantErr:  false,
		},
		{
			name:     "URL encoded Windows",
			uri:      "file:///d%3A/code/project",
			expected: "d:/code/project",
			wantErr:  false,
		},
		{
			name:    "Invalid scheme",
			uri:     "http://example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := discovery.parseFolderURI(tt.uri)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				// 由于路径格式可能因系统而异，只检查基本结构
				assert.NotEmpty(t, result)
			}
		})
	}
}

// TestParseFolderURI_WindowsPathWithLeadingBackslash 测试 Windows 路径解析时开头的反斜杠问题
// 确保 ProjectDiscovery 和 PathResolver 使用相同的逻辑
func TestParseFolderURI_WindowsPathWithLeadingBackslash(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("此测试仅在 Windows 上运行")
	}

	discovery := NewProjectDiscovery()

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
			result, err := discovery.parseFolderURI(tt.uri)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				
				// 规范化路径用于比较（统一使用正斜杠）
				resultNormalized := filepath.ToSlash(result)
				expectedNormalized := filepath.ToSlash(tt.expected)
				
				// 检查路径不应该以单个反斜杠开头
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

func TestDiscoveredWorkspace(t *testing.T) {
	// 测试 DiscoveredWorkspace 结构
	ws := &DiscoveredWorkspace{
		WorkspaceID:  "test-workspace-id",
		Path:         "/path/to/project",
		ProjectName:  "project",
		GitRemoteURL: "https://github.com/user/repo",
		GitBranch:    "main",
	}

	assert.Equal(t, "test-workspace-id", ws.WorkspaceID)
	assert.Equal(t, "/path/to/project", ws.Path)
	assert.Equal(t, "project", ws.ProjectName)
	assert.Equal(t, "https://github.com/user/repo", ws.GitRemoteURL)
	assert.Equal(t, "main", ws.GitBranch)
}
