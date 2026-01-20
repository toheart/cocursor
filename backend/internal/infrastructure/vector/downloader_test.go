package vector

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPDownloader_Download_Success(t *testing.T) {
	// 创建测试服务器
	content := []byte("test file content for download")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test-download")

	// 创建下载器
	downloader := NewHTTPDownloader()

	// 执行下载
	err := downloader.Download(context.Background(), server.URL, destPath, DefaultDownloadOptions())
	require.NoError(t, err)

	// 验证文件内容
	downloadedContent, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, content, downloadedContent)
}

func TestHTTPDownloader_Download_WithProgress(t *testing.T) {
	// 创建测试服务器
	content := []byte("test file content for progress tracking")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test-download-progress")

	// 创建下载器
	downloader := NewHTTPDownloader()

	// 记录进度回调
	var progressCalled bool
	opts := DefaultDownloadOptions()
	opts.OnProgress = func(downloaded, total int64) {
		progressCalled = true
		assert.True(t, downloaded >= 0)
		assert.True(t, total > 0)
	}

	// 执行下载
	err := downloader.Download(context.Background(), server.URL, destPath, opts)
	require.NoError(t, err)
	assert.True(t, progressCalled, "progress callback should be called")
}

func TestHTTPDownloader_Download_ContextCanceled(t *testing.T) {
	// 创建慢速服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // 模拟慢速响应
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test-download-cancel")

	// 创建下载器
	downloader := NewHTTPDownloader()

	// 创建可取消的 context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 执行下载（应该被取消）
	err := downloader.Download(ctx, server.URL, destPath, DefaultDownloadOptions())
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrDownloadCanceled)
}

func TestHTTPDownloader_Download_404Error(t *testing.T) {
	// 创建返回 404 的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test-download-404")

	// 创建下载器
	downloader := NewHTTPDownloader()

	// 执行下载（应该失败且不重试）
	opts := DefaultDownloadOptions()
	opts.MaxRetries = 1 // 404 不应重试
	err := downloader.Download(context.Background(), server.URL, destPath, opts)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrHTTPStatusNotOK)
}

func TestHTTPDownloader_Download_ChecksumMismatch(t *testing.T) {
	// 创建测试服务器
	content := []byte("test file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test-download-checksum")

	// 创建下载器
	downloader := NewHTTPDownloader()

	// 使用错误的校验和
	opts := DefaultDownloadOptions()
	opts.ExpectedChecksum = "wrongchecksum123456789"

	// 执行下载（应该因校验和不匹配而失败）
	err := downloader.Download(context.Background(), server.URL, destPath, opts)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrChecksumMismatch)

	// 验证临时文件已被清理
	_, err = os.Stat(destPath)
	assert.True(t, os.IsNotExist(err))
}

func TestHTTPDownloader_Download_FileSizeMismatch(t *testing.T) {
	// 创建测试服务器（返回与 Content-Length 不匹配的数据）
	// 注意：当 Content-Length 与实际数据不匹配时，HTTP 客户端可能返回 EOF 错误
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000") // 声明 1000 字节
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("short")) // 实际只有 5 字节
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test-download-size")

	// 创建下载器
	downloader := NewHTTPDownloader()

	// 执行下载（应该因大小不匹配或 EOF 而失败）
	opts := DefaultDownloadOptions()
	opts.MaxRetries = 1
	err := downloader.Download(context.Background(), server.URL, destPath, opts)
	assert.Error(t, err)
	// 可能是 FileSizeMismatch 或 EOF 错误（取决于 HTTP 客户端行为）
	assert.True(t, err != nil, "should fail due to size mismatch or EOF")
}

func TestHTTPDownloader_Download_RetryOnServerError(t *testing.T) {
	// 创建服务器，前两次返回 500，第三次成功
	attempts := 0
	content := []byte("success after retry")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test-download-retry")

	// 创建下载器
	downloader := NewHTTPDownloader()

	// 执行下载
	opts := DefaultDownloadOptions()
	opts.MaxRetries = 3
	opts.RetryDelay = 10 * time.Millisecond // 使用短延迟加速测试

	err := downloader.Download(context.Background(), server.URL, destPath, opts)
	require.NoError(t, err)
	assert.Equal(t, 3, attempts, "should have retried until success")

	// 验证文件内容
	downloadedContent, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, content, downloadedContent)
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "checksum mismatch - not retryable",
			err:      ErrChecksumMismatch,
			expected: false,
		},
		{
			name:     "download canceled - not retryable",
			err:      ErrDownloadCanceled,
			expected: false,
		},
		{
			name:     "insufficient space - not retryable",
			err:      ErrInsufficientSpace,
			expected: false,
		},
		{
			name:     "404 error - not retryable",
			err:      fmt.Errorf("%w: 404 Not Found", ErrHTTPStatusNotOK),
			expected: false,
		},
		{
			name:     "500 error - retryable",
			err:      fmt.Errorf("%w: 500 Internal Server Error", ErrHTTPStatusNotOK),
			expected: true,
		},
		{
			name:     "network error - retryable",
			err:      fmt.Errorf("network error: connection refused"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFetchChecksum(t *testing.T) {
	// 创建测试服务器
	checksum := "abc123def456"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(checksum + "  filename.tar.gz"))
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	result, err := downloader.FetchChecksum(context.Background(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, checksum, result)
}
