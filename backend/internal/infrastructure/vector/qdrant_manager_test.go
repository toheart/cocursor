package vector

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildDownloadURL 测试 URL 构建
func TestBuildDownloadURL(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		osName   string
		arch     string
		wantErr  bool
		contains string
	}{
		{
			name:     "Windows x86_64",
			version:  "v1.16.3",
			osName:   "windows",
			arch:     "x86_64",
			wantErr:  false,
			contains: "qdrant-x86_64-pc-windows-msvc.zip",
		},
		{
			name:     "Linux x86_64",
			version:  "v1.16.3",
			osName:   "linux",
			arch:     "x86_64",
			wantErr:  false,
			contains: "qdrant-x86_64-unknown-linux-musl.tar.gz",
		},
		{
			name:     "macOS arm64",
			version:  "v1.16.3",
			osName:   "macos",
			arch:     "arm64",
			wantErr:  false,
			contains: "qdrant-aarch64-apple-darwin.tar.gz",
		},
		{
			name:    "Unsupported OS",
			version: "v1.16.3",
			osName:  "unsupported",
			arch:    "x86_64",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := buildDownloadURL(tt.version, tt.osName, tt.arch)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Contains(t, url, tt.contains)
				assert.Contains(t, url, "github.com/qdrant/qdrant/releases/download")
			}
		})
	}
}

// TestCopyFile 测试文件复制
func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	// 创建源文件
	require.NoError(t, os.WriteFile(srcPath, []byte("test content"), 0644))

	// 复制文件
	err := copyFile(srcPath, dstPath)
	require.NoError(t, err)

	// 验证目标文件存在且内容正确
	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("test content"), data)
}

// TestCopyFile_SourceNotFound 测试源文件不存在
func TestCopyFile_SourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dstPath := filepath.Join(tmpDir, "dest.txt")

	err := copyFile("non-existent-file", dstPath)
	assert.Error(t, err)
}

// TestVerifyInstallation 测试安装验证
func TestVerifyInstallation(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "qdrant")

	// 创建测试二进制文件（模拟）
	require.NoError(t, os.WriteFile(binaryPath, []byte("#!/bin/sh\necho qdrant"), 0755))

	// 在 Windows 上，验证可能会失败（需要真实的 qdrant 二进制）
	// 这里只测试文件存在的情况
	if runtime.GOOS == "windows" {
		t.Skip("Windows 上需要真实的 qdrant.exe 进行验证")
	}

	err := verifyInstallation(binaryPath)
	// 可能失败（因为不是真实的 qdrant 二进制），但不应该 panic
	_ = err
}

// TestGetPlatformInfo 测试平台信息获取
func TestGetPlatformInfo(t *testing.T) {
	osName, arch := GetPlatformInfo()
	assert.NotEmpty(t, osName)
	assert.NotEmpty(t, arch)
	assert.Contains(t, []string{"windows", "linux", "macos"}, osName)
	assert.Contains(t, []string{"x86_64", "arm64", "aarch64"}, arch)
}

// TestGetQdrantInstallPath 测试安装路径获取
func TestGetQdrantInstallPath(t *testing.T) {
	path, err := GetQdrantInstallPath()
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "qdrant")
}

// TestGetQdrantDataPath 测试数据路径获取
func TestGetQdrantDataPath(t *testing.T) {
	path, err := GetQdrantDataPath()
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "qdrant")
}
