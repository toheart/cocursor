package vector

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"log/slog"

	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/qdrant/go-client/qdrant"
)

// QdrantManager Qdrant 管理器
type QdrantManager struct {
	binaryPath string
	dataPath   string
	grpcPort   int
	httpPort   int
	cmd        *exec.Cmd
	client     *qdrant.Client
	configPath string // 临时配置文件路径（如果使用）
	logger     *slog.Logger
}

// NewQdrantManager 创建 Qdrant 管理器
func NewQdrantManager(binaryPath, dataPath string) *QdrantManager {
	return &QdrantManager{
		binaryPath: binaryPath,
		dataPath:   dataPath,
		grpcPort:   6334,
		httpPort:   6333,
		logger:     log.NewModuleLogger("qdrant", "manager"),
	}
}

// GetBinaryPath 获取 Qdrant 二进制路径
func (q *QdrantManager) GetBinaryPath() string {
	return q.binaryPath
}

// GetDataPath 获取数据存储路径
func (q *QdrantManager) GetDataPath() string {
	return q.dataPath
}

// Start 启动 Qdrant 服务
func (q *QdrantManager) Start() error {
	// 检查是否已在运行
	if q.IsRunning() {
		// 如果已有客户端，直接返回
		if q.client != nil {
			return nil
		}
		// 如果服务在运行但没有客户端，尝试连接
		client, err := qdrant.NewClient(&qdrant.Config{
			Host:                   "localhost",
			Port:                   q.grpcPort,
			SkipCompatibilityCheck: true,
		})
		if err == nil {
			q.client = client
			return nil
		}
	}

	// 确保数据目录存在
	if err := os.MkdirAll(q.dataPath, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// 检查二进制文件是否存在
	if _, err := os.Stat(q.binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("qdrant binary not found at %s", q.binaryPath)
	}

	// 创建临时配置文件
	configPath, err := q.createConfigFile()
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	q.configPath = configPath // 保存路径，在 Stop() 时删除

	// 构建启动命令（使用配置文件）
	args := []string{
		"--config-path", configPath,
	}

	q.cmd = exec.Command(q.binaryPath, args...)
	q.cmd.Stdout = os.Stdout
	q.cmd.Stderr = os.Stderr

	// 设置环境变量（作为备用方案，环境变量会覆盖配置文件中的设置）
	q.cmd.Env = append(os.Environ(),
		fmt.Sprintf("QDRANT__STORAGE__STORAGE_PATH=%s", q.dataPath),
		fmt.Sprintf("QDRANT__SERVICE__HTTP_PORT=%d", q.httpPort),
		fmt.Sprintf("QDRANT__SERVICE__GRPC_PORT=%d", q.grpcPort),
	)

	// 启动进程
	if err := q.cmd.Start(); err != nil {
		os.Remove(configPath) // 启动失败时立即删除配置文件
		q.configPath = ""
		return fmt.Errorf("failed to start qdrant: %w", err)
	}

	// 等待服务就绪
	if err := q.waitForReady(10 * time.Second); err != nil {
		q.Stop()
		return fmt.Errorf("qdrant failed to become ready: %w", err)
	}

	// 创建客户端连接
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:                   "localhost",
		Port:                   q.grpcPort,
		SkipCompatibilityCheck: true, // 跳过版本检查，避免在服务未就绪时产生警告
	})
	if err != nil {
		q.Stop()
		return fmt.Errorf("failed to connect to qdrant: %w", err)
	}

	q.client = client

	return nil
}

// Stop 停止 Qdrant 服务
func (q *QdrantManager) Stop() error {
	if q.cmd != nil && q.cmd.Process != nil {
		if err := q.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill qdrant process: %w", err)
		}
		q.cmd.Wait()
		q.cmd = nil
	}

	if q.client != nil {
		// 关闭连接
		q.client.Close()
		q.client = nil
	}

	// 删除临时配置文件
	if q.configPath != "" {
		os.Remove(q.configPath)
		q.configPath = ""
	}

	return nil
}

// GetClient 获取 Qdrant 客户端
func (q *QdrantManager) GetClient() *qdrant.Client {
	return q.client
}

// IsRunning 检查 Qdrant 服务是否已在运行
func (q *QdrantManager) IsRunning() bool {
	// 如果已有客户端连接，说明服务在运行
	if q.client != nil {
		// 测试连接是否有效
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := q.client.ListCollections(ctx)
		return err == nil
	}

	// 检查端口是否被占用（简单检查）
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", q.grpcPort), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// waitForReady 等待 Qdrant 服务就绪
func (q *QdrantManager) waitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// 尝试连接 Qdrant（跳过版本检查避免警告）
		client, err := qdrant.NewClient(&qdrant.Config{
			Host:                   "localhost",
			Port:                   q.grpcPort,
			SkipCompatibilityCheck: true, // 跳过版本检查，避免在服务未就绪时产生警告
		})
		if err == nil {
			// 测试连接：尝试列出集合
			_, err = client.ListCollections(context.Background())
			if err == nil {
				client.Close()
				return nil
			}
			client.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for qdrant to be ready")
}

// createConfigFile 创建临时配置文件
func (q *QdrantManager) createConfigFile() (string, error) {
	// 创建临时配置文件
	tmpFile, err := os.CreateTemp("", "qdrant-config-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp config file: %w", err)
	}
	configPath := tmpFile.Name()
	tmpFile.Close()

	// 写入配置内容
	configContent := fmt.Sprintf(`storage:
  storage_path: %s

service:
  http_port: %d
  grpc_port: %d
  host: "127.0.0.1"
`, q.dataPath, q.httpPort, q.grpcPort)

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		os.Remove(configPath)
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return configPath, nil
}

// EnsureCollections 确保集合存在
func (q *QdrantManager) EnsureCollections(vectorSize uint64) error {
	if q.client == nil {
		return fmt.Errorf("qdrant client not initialized")
	}

	collections := []string{"cursor_sessions_messages", "cursor_sessions_turns"}
	ctx := context.Background()

	// 获取现有集合列表
	existingCollections, err := q.client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	// 检查每个集合是否存在
	collectionMap := make(map[string]bool)
	for _, name := range existingCollections {
		collectionMap[name] = true
	}

	for _, collectionName := range collections {
		if !collectionMap[collectionName] {
			// 创建集合
			err := q.client.CreateCollection(ctx, &qdrant.CreateCollection{
				CollectionName: collectionName,
				VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
					Size:     vectorSize,
					Distance: qdrant.Distance_Cosine,
				}),
			})
			if err != nil {
				return fmt.Errorf("failed to create collection %s: %w", collectionName, err)
			}
		}
	}

	return nil
}

// ClearCollections 清空所有集合中的数据（通过删除并重新创建集合）
func (q *QdrantManager) ClearCollections() error {
	if q.client == nil {
		return fmt.Errorf("qdrant client not initialized")
	}

	collections := []string{"cursor_sessions_messages", "cursor_sessions_turns"}
	ctx := context.Background()

	// 获取现有集合列表
	existingCollections, err := q.client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	// 检查每个集合是否存在
	collectionMap := make(map[string]bool)
	for _, col := range existingCollections {
		collectionMap[col] = true
	}

	for _, collectionName := range collections {
		if collectionMap[collectionName] {
			// 删除集合
			err := q.client.DeleteCollection(ctx, collectionName)
			if err != nil {
				q.logger.Warn("Failed to delete collection", "collection", collectionName, "error", err)
				continue
			}
			q.logger.Info("Collection deleted successfully", "collection", collectionName)
		}
	}

	return nil
}

// GetPlatformInfo 获取平台信息（用于下载）
func GetPlatformInfo() (osName, arch string) {
	osName = runtime.GOOS
	arch = runtime.GOARCH

	// 标准化 OS 名称
	switch osName {
	case "darwin":
		osName = "macos"
	case "windows":
		osName = "windows"
	case "linux":
		osName = "linux"
	}

	// 标准化架构名称
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "arm64"
	}

	return osName, arch
}

// GetQdrantInstallPath 获取 Qdrant 安装路径
func GetQdrantInstallPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	osName, _ := GetPlatformInfo()
	binaryName := "qdrant"
	if osName == "windows" {
		binaryName = "qdrant.exe"
	}

	installPath := filepath.Join(homeDir, ".cocursor", "bin", "qdrant", binaryName)
	return installPath, nil
}

// GetQdrantDataPath 获取 Qdrant 数据路径
func GetQdrantDataPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	dataPath := filepath.Join(homeDir, ".cocursor", "data", "qdrant")
	return dataPath, nil
}

// GitHubRelease GitHub Release 信息
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// DownloadQdrant 下载并安装 Qdrant
// version: 版本号，如 "v1.16.3"，如果为空则使用最新稳定版
func DownloadQdrant(version string) (string, error) {
	// 如果版本为空，使用固定版本 v1.16.3
	if version == "" {
		version = "v1.16.3"
	}

	fmt.Printf("[Qdrant Download] Starting download process for version %s\n", version)

	// 获取平台信息
	osName, arch := GetPlatformInfo()
	fmt.Printf("[Qdrant Download] Platform: %s/%s\n", osName, arch)

	// 构建下载 URL
	downloadURL, err := buildDownloadURL(version, osName, arch)
	if err != nil {
		return "", fmt.Errorf("failed to build download URL: %w", err)
	}
	fmt.Printf("[Qdrant Download] Download URL: %s\n", downloadURL)

	// 获取安装路径
	installPath, err := GetQdrantInstallPath()
	if err != nil {
		return "", fmt.Errorf("failed to get install path: %w", err)
	}
	fmt.Printf("[Qdrant Download] Install path: %s\n", installPath)

	// 检查是否已安装
	if _, err := os.Stat(installPath); err == nil {
		// 已安装，检查版本
		installedVersion, err := getInstalledVersion(installPath)
		if err == nil && installedVersion == version {
			fmt.Printf("[Qdrant Download] Already installed with version %s\n", installedVersion)
			return installPath, nil
		}
		fmt.Printf("[Qdrant Download] Installed version %s differs from requested %s, will reinstall\n", installedVersion, version)
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "qdrant-download-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Printf("[Qdrant Download] Warning: failed to remove temp directory %s: %v\n", tmpDir, err)
		}
	}()
	fmt.Printf("[Qdrant Download] Temp directory: %s\n", tmpDir)

	// 下载文件
	downloadPath := filepath.Join(tmpDir, filepath.Base(downloadURL))
	fmt.Printf("[Qdrant Download] Starting download to %s\n", downloadPath)
	if err := downloadFile(downloadURL, downloadPath); err != nil {
		return "", fmt.Errorf("failed to download qdrant from %s: %w", downloadURL, err)
	}

	// 验证文件完整性（如果有 SHA256 文件）
	fmt.Printf("[Qdrant Download] Verifying checksum...\n")
	if err := verifyChecksum(downloadURL, downloadPath); err != nil {
		// 验证失败，但不阻止安装（某些版本可能没有 checksum 文件）
		fmt.Printf("[Qdrant Download] Warning: failed to verify checksum: %v (continuing anyway)\n", err)
	} else {
		fmt.Printf("[Qdrant Download] Checksum verified successfully\n")
	}

	// 解压文件
	fmt.Printf("[Qdrant Download] Extracting archive...\n")
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := extractArchive(downloadPath, extractDir, osName); err != nil {
		return "", fmt.Errorf("failed to extract archive: %w", err)
	}
	fmt.Printf("[Qdrant Download] Archive extracted successfully\n")

	// 查找二进制文件
	binaryName := "qdrant"
	if osName == "windows" {
		binaryName = "qdrant.exe"
	}
	fmt.Printf("[Qdrant Download] Looking for binary: %s\n", binaryName)
	binaryPath := findBinaryInExtracted(extractDir, binaryName)
	if binaryPath == "" {
		return "", fmt.Errorf("binary %s not found in extracted archive (searched in %s)", binaryName, extractDir)
	}
	fmt.Printf("[Qdrant Download] Binary found at: %s\n", binaryPath)

	// 确保安装目录存在
	installDir := filepath.Dir(installPath)
	fmt.Printf("[Qdrant Download] Creating install directory: %s\n", installDir)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create install directory %s: %w", installDir, err)
	}

	// 复制二进制文件
	fmt.Printf("[Qdrant Download] Copying binary to %s\n", installPath)
	if err := copyFile(binaryPath, installPath); err != nil {
		return "", fmt.Errorf("failed to copy binary from %s to %s: %w", binaryPath, installPath, err)
	}

	// 设置执行权限（非 Windows）
	if osName != "windows" {
		fmt.Printf("[Qdrant Download] Setting executable permissions\n")
		if err := os.Chmod(installPath, 0755); err != nil {
			return "", fmt.Errorf("failed to set executable permission: %w", err)
		}
	}

	// 验证安装
	fmt.Printf("[Qdrant Download] Verifying installation...\n")
	if err := verifyInstallation(installPath); err != nil {
		return "", fmt.Errorf("failed to verify installation: %w", err)
	}
	fmt.Printf("[Qdrant Download] Installation verified successfully\n")

	fmt.Printf("[Qdrant Download] Qdrant %s installed successfully at %s\n", version, installPath)
	return installPath, nil
}

// buildDownloadURL 构建下载 URL
func buildDownloadURL(version, osName, arch string) (string, error) {
	// Qdrant GitHub Releases URL 格式
	// https://github.com/qdrant/qdrant/releases/download/v1.16.3/qdrant-x86_64-pc-windows-msvc.zip
	// 注意：文件名中不包含版本号，版本号只在 URL 路径中
	baseURL := "https://github.com/qdrant/qdrant/releases/download"

	// 构建文件名（不包含版本号）
	var filename string

	switch osName {
	case "windows":
		if arch == "x86_64" {
			filename = "qdrant-x86_64-pc-windows-msvc.zip"
		} else {
			return "", fmt.Errorf("unsupported architecture for Windows: %s", arch)
		}
	case "macos":
		if arch == "x86_64" {
			filename = "qdrant-x86_64-apple-darwin.zip"
		} else if arch == "arm64" {
			filename = "qdrant-aarch64-apple-darwin.zip"
		} else {
			return "", fmt.Errorf("unsupported architecture for macOS: %s", arch)
		}
	case "linux":
		if arch == "x86_64" {
			filename = "qdrant-x86_64-unknown-linux-musl.tar.gz"
		} else if arch == "arm64" {
			filename = "qdrant-aarch64-unknown-linux-musl.tar.gz"
		} else {
			return "", fmt.Errorf("unsupported architecture for Linux: %s", arch)
		}
	default:
		return "", fmt.Errorf("unsupported OS: %s", osName)
	}

	return fmt.Sprintf("%s/%s/%s", baseURL, version, filename), nil
}

// downloadFile 下载文件（带超时和重试）
func downloadFile(url, destPath string) error {
	const maxRetries = 3
	const timeout = 5 * time.Minute // 5分钟超时

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			fmt.Printf("Retrying download (attempt %d/%d)...\n", attempt, maxRetries)
			time.Sleep(time.Duration(attempt) * time.Second) // 递增延迟
		}

		// 创建带超时的 HTTP 客户端
		client := &http.Client{
			Timeout: timeout,
		}

		resp, err := client.Get(url)
		if err != nil {
			lastErr = fmt.Errorf("failed to download file: %w", err)
			fmt.Printf("Download attempt %d failed: %v\n", attempt, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("failed to download file: status code %d", resp.StatusCode)
			fmt.Printf("Download attempt %d failed: HTTP %d\n", attempt, resp.StatusCode)
			continue
		}

		// 获取文件大小（用于进度显示）
		contentLength := resp.ContentLength
		if contentLength > 0 {
			fmt.Printf("Downloading %s (%.2f MB)...\n", filepath.Base(destPath), float64(contentLength)/(1024*1024))
		}

		// 创建目标文件
		out, err := os.Create(destPath)
		if err != nil {
			lastErr = fmt.Errorf("failed to create file: %w", err)
			fmt.Printf("Download attempt %d failed: %v\n", attempt, err)
			continue
		}

		// 复制数据
		written, err := io.Copy(out, resp.Body)
		if err != nil {
			out.Close()
			os.Remove(destPath) // 删除不完整的文件
			lastErr = fmt.Errorf("failed to write file: %w", err)
			fmt.Printf("Download attempt %d failed: %v (written: %d bytes)\n", attempt, err, written)
			continue
		}

		if err := out.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close file: %w", err)
			fmt.Printf("Download attempt %d failed: %v\n", attempt, err)
			continue
		}

		// 验证文件大小
		if contentLength > 0 && written != contentLength {
			os.Remove(destPath)
			lastErr = fmt.Errorf("file size mismatch: expected %d bytes, got %d bytes", contentLength, written)
			fmt.Printf("Download attempt %d failed: %v\n", attempt, lastErr)
			continue
		}

		fmt.Printf("Download completed successfully (%d bytes)\n", written)
		return nil
	}

	return fmt.Errorf("download failed after %d attempts: %w", maxRetries, lastErr)
}

// verifyChecksum 验证文件校验和
func verifyChecksum(downloadURL, filePath string) error {
	// 尝试下载 SHA256 文件
	checksumURL := downloadURL + ".sha256"
	resp, err := http.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("checksum file not available: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksum file not found")
	}

	checksumData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksum: %w", err)
	}

	// 解析校验和（格式：hash  filename）
	checksumParts := strings.Fields(string(checksumData))
	if len(checksumParts) == 0 {
		return fmt.Errorf("invalid checksum format")
	}
	expectedHash := checksumParts[0]

	// 计算文件哈希
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	actualHash := hex.EncodeToString(hash.Sum(nil))

	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

// extractArchive 解压归档文件
func extractArchive(archivePath, destDir string, osName string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}

	// 根据文件扩展名选择解压方式
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, destDir)
	} else if strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz") {
		return extractTarGz(archivePath, destDir)
	}

	return fmt.Errorf("unsupported archive format")
}

// extractZip 解压 ZIP 文件
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.FileInfo().Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		out, err := os.Create(path)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}

		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}

		os.Chmod(path, f.FileInfo().Mode())
	}

	return nil
}

// extractTarGz 解压 tar.gz 文件（纯 Go 实现，跨平台）
func extractTarGz(tarGzPath, destDir string) error {
	// 打开 tar.gz 文件
	file, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	// 创建 gzip 读取器
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// 创建 tar 读取器
	tarReader := tar.NewReader(gzReader)

	// 遍历 tar 文件中的所有条目
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // 文件结束
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// 构建目标路径
		targetPath := filepath.Join(destDir, header.Name)

		// 检查路径安全性（防止路径遍历攻击）
		// 使用 filepath.Clean 和 filepath.EvalSymlinks 来规范化路径
		cleanDest := filepath.Clean(destDir)
		cleanTarget := filepath.Clean(targetPath)
		if !strings.HasPrefix(cleanTarget, cleanDest+string(os.PathSeparator)) && cleanTarget != cleanDest {
			return fmt.Errorf("invalid path in tar archive: %s (potential path traversal)", header.Name)
		}

		// 根据文件类型处理
		switch header.Typeflag {
		case tar.TypeDir:
			// 创建目录
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			// 创建文件
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for file: %w", err)
			}

			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			// 复制文件内容
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract file content: %w", err)
			}

			outFile.Close()

		case tar.TypeSymlink:
			// 处理符号链接（可选，某些平台可能不支持）
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				// 在某些系统上符号链接可能失败，记录但不阻止解压
				fmt.Printf("Warning: failed to create symlink %s -> %s: %v\n", targetPath, header.Linkname, err)
			}

		default:
			// 忽略其他类型的条目（如硬链接等）
			fmt.Printf("Warning: unsupported tar entry type %c for %s\n", header.Typeflag, header.Name)
		}
	}

	return nil
}

// findBinaryInExtracted 在解压目录中查找二进制文件
func findBinaryInExtracted(extractDir, binaryName string) string {
	var foundPath string
	filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == binaryName {
			foundPath = path
			return filepath.SkipAll
		}
		return nil
	})
	return foundPath
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// verifyInstallation 验证安装
func verifyInstallation(binaryPath string) error {
	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to verify installation: %w", err)
	}

	// 检查输出是否包含版本信息
	if len(output) == 0 {
		return fmt.Errorf("invalid version output")
	}

	return nil
}

// getInstalledVersion 获取已安装的版本
func getInstalledVersion(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// 解析版本号（输出格式：qdrant 1.16.3）
	outputStr := strings.TrimSpace(string(output))
	parts := strings.Fields(outputStr)
	if len(parts) >= 2 {
		return "v" + parts[1], nil
	}

	return "", fmt.Errorf("failed to parse version")
}
