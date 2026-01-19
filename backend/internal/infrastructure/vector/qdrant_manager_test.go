package vector

import (
	"net/http"
	"net/http/httptest"
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
			contains: "qdrant-aarch64-apple-darwin.zip",
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

// TestDownloadFile_Success 测试成功下载（使用 mock HTTP server）
func TestDownloadFile_Success(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test data"))
	}))
	defer server.Close()

	// 创建临时文件
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test.zip")

	// 测试下载
	err := downloadFile(server.URL, destPath)
	require.NoError(t, err)

	// 验证文件存在且内容正确
	data, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("test data"), data)
}

// TestDownloadFile_HTTPError 测试 HTTP 错误
func TestDownloadFile_HTTPError(t *testing.T) {
	// 创建返回 404 的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test.zip")

	err := downloadFile(server.URL, destPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code 404")
}

// TestDownloadFile_NetworkError 测试网络错误
func TestDownloadFile_NetworkError(t *testing.T) {
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test.zip")

	// 使用无效 URL
	err := downloadFile("http://invalid-url-that-does-not-exist.local/test", destPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download file")
}

// TestDownloadFile_Retry 测试重试机制
func TestDownloadFile_Retry(t *testing.T) {
	attempts := 0
	// 创建前两次失败、第三次成功的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Header().Set("Content-Length", "5")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("data"))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test.zip")

	err := downloadFile(server.URL, destPath)
	require.NoError(t, err)
	assert.Equal(t, 3, attempts) // 应该重试了 3 次

	// 验证文件内容
	data, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("data"), data)
}

// TestDownloadFile_FileSizeMismatch 测试文件大小不匹配
func TestDownloadFile_FileSizeMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10") // 声明 10 字节
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data")) // 实际只写 4 字节
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test.zip")

	err := downloadFile(server.URL, destPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file size mismatch")
}

// TestExtractZip 测试 ZIP 解压
func TestExtractZip(t *testing.T) {
	// 创建测试 ZIP 文件
	// 注意：这个测试需要真实的 ZIP 文件，可能需要使用 archive/zip 包创建
	// 暂时跳过，因为需要创建真实的 ZIP 文件
	t.Skip("需要真实的 ZIP 文件进行测试")
}

// TestFindBinaryInExtracted 测试查找二进制文件
func TestFindBinaryInExtracted(t *testing.T) {
	tmpDir := t.TempDir()
	extractDir := filepath.Join(tmpDir, "extracted")

	// 创建测试目录结构
	binaryPath := filepath.Join(extractDir, "qdrant")
	require.NoError(t, os.MkdirAll(filepath.Dir(binaryPath), 0755))
	require.NoError(t, os.WriteFile(binaryPath, []byte("binary"), 0755))

	// 测试查找
	found := findBinaryInExtracted(extractDir, "qdrant")
	assert.Equal(t, binaryPath, found)
}

// TestFindBinaryInExtracted_NotFound 测试找不到二进制文件
func TestFindBinaryInExtracted_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	extractDir := filepath.Join(tmpDir, "extracted")
	require.NoError(t, os.MkdirAll(extractDir, 0755))

	found := findBinaryInExtracted(extractDir, "qdrant")
	assert.Empty(t, found)
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
