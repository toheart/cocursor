package vector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"log/slog"

	"github.com/cocursor/backend/internal/infrastructure/log"
)

// 下载相关错误
var (
	ErrDownloadCanceled   = errors.New("download canceled")
	ErrChecksumMismatch   = errors.New("checksum mismatch")
	ErrFileSizeMismatch   = errors.New("file size mismatch")
	ErrInsufficientSpace  = errors.New("insufficient disk space")
	ErrDownloadFailed     = errors.New("download failed")
	ErrHTTPStatusNotOK    = errors.New("HTTP status not OK")
)

// ProgressCallback 下载进度回调函数
// downloaded: 已下载字节数
// total: 总字节数（如果未知则为 -1）
type ProgressCallback func(downloaded, total int64)

// DownloadOptions 下载选项
type DownloadOptions struct {
	// OnProgress 进度回调函数，每秒至少调用一次
	OnProgress ProgressCallback
	// ExpectedSize 预期文件大小（用于校验，0 表示不校验）
	ExpectedSize int64
	// ExpectedChecksum SHA256 校验和（空字符串表示不校验）
	ExpectedChecksum string
	// MaxRetries 最大重试次数（默认 3）
	MaxRetries int
	// RetryDelay 重试延迟基数（默认 1s，使用指数退避）
	RetryDelay time.Duration
}

// DefaultDownloadOptions 返回默认下载选项
func DefaultDownloadOptions() DownloadOptions {
	return DownloadOptions{
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// Downloader 文件下载器接口
type Downloader interface {
	// Download 下载文件到指定路径
	// ctx: 用于取消下载的 context
	// url: 下载 URL
	// destPath: 目标文件路径
	// opts: 下载选项
	Download(ctx context.Context, url, destPath string, opts DownloadOptions) error
}

// HTTPDownloader HTTP 文件下载器实现
type HTTPDownloader struct {
	client *http.Client
	logger *slog.Logger
}

// NewHTTPDownloader 创建新的 HTTP 下载器
func NewHTTPDownloader() *HTTPDownloader {
	// 创建自定义 Transport，分离各种超时
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment, // 支持环境变量代理配置
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second, // 连接超时
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second, // TLS 握手超时
		ResponseHeaderTimeout: 30 * time.Second, // 响应头超时
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		// 注意：不设置整体 Timeout，由 context 控制
	}

	return &HTTPDownloader{
		client: &http.Client{
			Transport: transport,
			// 不设置 Timeout，让 context 控制超时
		},
		logger: log.NewModuleLogger("vector", "downloader"),
	}
}

// Download 实现 Downloader 接口
func (d *HTTPDownloader) Download(ctx context.Context, url, destPath string, opts DownloadOptions) error {
	// 设置默认值
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = 3
	}
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = time.Second
	}

	var lastErr error
	for attempt := 1; attempt <= opts.MaxRetries; attempt++ {
		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %v", ErrDownloadCanceled, ctx.Err())
		default:
		}

		if attempt > 1 {
			d.logger.Info("retrying download",
				"attempt", attempt,
				"max_retries", opts.MaxRetries,
				"url", url)
			
			// 指数退避等待
			waitTime := opts.RetryDelay * time.Duration(1<<(attempt-2))
			select {
			case <-ctx.Done():
				return fmt.Errorf("%w: %v", ErrDownloadCanceled, ctx.Err())
			case <-time.After(waitTime):
			}
		}

		err := d.downloadOnce(ctx, url, destPath, opts)
		if err == nil {
			return nil
		}

		lastErr = err
		d.logger.Warn("download attempt failed",
			"attempt", attempt,
			"error", err)

		// 检查是否为不可重试错误
		if !isRetryableError(err) {
			return err
		}
	}

	return fmt.Errorf("%w after %d attempts: %v", ErrDownloadFailed, opts.MaxRetries, lastErr)
}

// downloadOnce 执行单次下载尝试
func (d *HTTPDownloader) downloadOnce(ctx context.Context, url, destPath string, opts DownloadOptions) error {
	// 确保目标目录存在
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// 使用临时文件下载，成功后重命名
	tmpPath := destPath + ".tmp"
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// 创建带 context 的请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置 User-Agent
	req.Header.Set("User-Agent", "cocursor-downloader/1.0")

	// 发送请求
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d %s", ErrHTTPStatusNotOK, resp.StatusCode, resp.Status)
	}

	// 获取文件大小
	contentLength := resp.ContentLength
	if contentLength > 0 {
		d.logger.Info("downloading file",
			"url", url,
			"size_mb", float64(contentLength)/(1024*1024))
	}

	// 创建目标文件
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// 创建进度报告 reader
	var reader io.Reader = resp.Body

	// 如果有进度回调，包装一个进度报告器
	if opts.OnProgress != nil {
		reader = &progressReader{
			reader:     resp.Body,
			total:      contentLength,
			onProgress: opts.OnProgress,
		}
	}

	// 复制数据
	written, err := io.Copy(out, reader)
	if err != nil {
		// 检查是否是 context 取消
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %v", ErrDownloadCanceled, ctx.Err())
		}
		return fmt.Errorf("failed to write file: %w", err)
	}

	// 关闭文件以确保数据刷新
	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	// 验证文件大小
	if contentLength > 0 && written != contentLength {
		return fmt.Errorf("%w: expected %d bytes, got %d bytes",
			ErrFileSizeMismatch, contentLength, written)
	}
	if opts.ExpectedSize > 0 && written != opts.ExpectedSize {
		return fmt.Errorf("%w: expected %d bytes, got %d bytes",
			ErrFileSizeMismatch, opts.ExpectedSize, written)
	}

	// 验证校验和
	if opts.ExpectedChecksum != "" {
		checksum, err := calculateFileChecksum(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to calculate checksum: %w", err)
		}
		if !strings.EqualFold(checksum, opts.ExpectedChecksum) {
			return fmt.Errorf("%w: expected %s, got %s",
				ErrChecksumMismatch, opts.ExpectedChecksum, checksum)
		}
		d.logger.Info("checksum verified", "checksum", checksum)
	}

	// 重命名临时文件为目标文件
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	success = true
	d.logger.Info("download completed",
		"path", destPath,
		"size_bytes", written)

	return nil
}

// progressReader 包装 io.Reader 以报告进度
type progressReader struct {
	reader       io.Reader
	total        int64
	downloaded   int64
	onProgress   ProgressCallback
	lastReport   time.Time
	reportPeriod time.Duration
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.downloaded += int64(n)

	// 每秒至少报告一次进度
	now := time.Now()
	if pr.reportPeriod == 0 {
		pr.reportPeriod = time.Second
	}
	if now.Sub(pr.lastReport) >= pr.reportPeriod || err == io.EOF {
		pr.onProgress(pr.downloaded, pr.total)
		pr.lastReport = now
	}

	return n, err
}

// calculateFileChecksum 计算文件的 SHA256 校验和
func calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// isRetryableError 判断错误是否可重试
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 不可重试的错误
	if errors.Is(err, ErrChecksumMismatch) ||
		errors.Is(err, ErrDownloadCanceled) ||
		errors.Is(err, ErrInsufficientSpace) {
		return false
	}

	// HTTP 状态码错误：只有服务器错误（5xx）可重试
	if errors.Is(err, ErrHTTPStatusNotOK) {
		errStr := err.Error()
		// 4xx 错误不可重试
		if strings.Contains(errStr, "400") ||
			strings.Contains(errStr, "401") ||
			strings.Contains(errStr, "403") ||
			strings.Contains(errStr, "404") {
			return false
		}
	}

	// 其他错误（网络超时、连接重置等）可重试
	return true
}

// FetchChecksum 从远程获取校验和文件内容
func (d *HTTPDownloader) FetchChecksum(ctx context.Context, checksumURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum file not available: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 解析校验和（格式：hash  filename 或仅 hash）
	checksumStr := strings.TrimSpace(string(data))
	parts := strings.Fields(checksumStr)
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid checksum format")
	}

	return parts[0], nil
}
