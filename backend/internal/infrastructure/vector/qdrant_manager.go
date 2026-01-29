package vector

import (
	"context"
	"fmt"
	"net"
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

	// 知识片段集合
	collections := []string{
		"cursor_knowledge",
	}
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
			q.logger.Info("Created collection", "collection", collectionName)
		}
	}

	return nil
}

// GetCollectionPointsCount 获取集合中的点数（实际索引数量）
func (q *QdrantManager) GetCollectionPointsCount(collectionName string) (uint64, error) {
	if q.client == nil {
		return 0, fmt.Errorf("qdrant client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := q.client.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		return 0, fmt.Errorf("failed to get collection info: %w", err)
	}

	if info.PointsCount == nil {
		return 0, nil
	}
	return *info.PointsCount, nil
}

// ClearCollections 清空所有集合中的数据（通过删除并重新创建集合）
func (q *QdrantManager) ClearCollections() error {
	if q.client == nil {
		return fmt.Errorf("qdrant client not initialized")
	}

	collections := []string{"cursor_knowledge"}
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

// QdrantDownloadOptions Qdrant 下载选项
type QdrantDownloadOptions struct {
	// Version 版本号，如 "v1.16.3"，空字符串使用默认版本
	Version string
	// OnProgress 进度回调函数
	OnProgress ProgressCallback
}

// DefaultQdrantVersion 默认 Qdrant 版本
const DefaultQdrantVersion = "v1.16.3"

// DownloadQdrant 下载并安装 Qdrant（向后兼容的简化版本）
// version: 版本号，如 "v1.16.3"，如果为空则使用默认版本
func DownloadQdrant(version string) (string, error) {
	return DownloadQdrantWithContext(context.Background(), QdrantDownloadOptions{
		Version: version,
	})
}

// DownloadQdrantWithContext 下载并安装 Qdrant（支持 Context 和进度回调）
// ctx: 用于取消下载的 context
// opts: 下载选项
func DownloadQdrantWithContext(ctx context.Context, opts QdrantDownloadOptions) (string, error) {
	logger := log.NewModuleLogger("qdrant", "download")

	// 设置默认版本
	version := opts.Version
	if version == "" {
		version = DefaultQdrantVersion
	}

	logger.Info("starting download", "version", version)

	// 获取平台信息
	osName, arch := GetPlatformInfo()
	logger.Info("platform detected", "os", osName, "arch", arch)

	// 构建下载 URL
	downloadURL, err := buildDownloadURL(version, osName, arch)
	if err != nil {
		return "", fmt.Errorf("failed to build download URL: %w", err)
	}
	logger.Info("download URL", "url", downloadURL)

	// 获取安装路径
	installPath, err := GetQdrantInstallPath()
	if err != nil {
		return "", fmt.Errorf("failed to get install path: %w", err)
	}
	logger.Info("install path", "path", installPath)

	// 检查是否已安装
	if _, err := os.Stat(installPath); err == nil {
		installedVersion, err := getInstalledVersion(installPath)
		if err == nil && installedVersion == version {
			logger.Info("already installed", "version", installedVersion)
			return installPath, nil
		}
		logger.Info("version mismatch, will reinstall",
			"installed", installedVersion,
			"requested", version)
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "qdrant-download-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			logger.Warn("failed to remove temp directory", "path", tmpDir, "error", err)
		}
	}()
	logger.Debug("temp directory created", "path", tmpDir)

	// 创建下载器和解压器
	downloader := NewHTTPDownloader()
	extractor := NewArchiveExtractor()

	// 准备下载选项
	downloadPath := filepath.Join(tmpDir, filepath.Base(downloadURL))
	downloadOpts := DefaultDownloadOptions()
	downloadOpts.OnProgress = opts.OnProgress

	// 尝试获取校验和（不阻止下载）
	checksumURL := downloadURL + ".sha256"
	if checksum, err := downloader.FetchChecksum(ctx, checksumURL); err == nil {
		downloadOpts.ExpectedChecksum = checksum
		logger.Info("checksum fetched", "checksum", checksum[:16]+"...")
	} else {
		logger.Warn("failed to fetch checksum", "error", err)
	}

	// 下载文件
	logger.Info("starting download", "dest", downloadPath)
	if err := downloader.Download(ctx, downloadURL, downloadPath, downloadOpts); err != nil {
		return "", fmt.Errorf("failed to download qdrant: %w", err)
	}
	logger.Info("download completed")

	// 解压文件
	extractDir := filepath.Join(tmpDir, "extracted")
	logger.Info("extracting archive", "dest", extractDir)
	if err := extractor.Extract(downloadPath, extractDir); err != nil {
		return "", fmt.Errorf("failed to extract archive: %w", err)
	}
	logger.Info("extraction completed")

	// 查找二进制文件
	binaryName := "qdrant"
	if osName == "windows" {
		binaryName = "qdrant.exe"
	}
	binaryPath, err := extractor.FindBinary(extractDir, binaryName)
	if err != nil {
		return "", fmt.Errorf("binary not found: %w", err)
	}
	logger.Info("binary found", "path", binaryPath)

	// 确保安装目录存在
	installDir := filepath.Dir(installPath)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create install directory: %w", err)
	}

	// 复制二进制文件
	if err := copyFile(binaryPath, installPath); err != nil {
		return "", fmt.Errorf("failed to copy binary: %w", err)
	}
	logger.Info("binary copied", "dest", installPath)

	// 设置执行权限（非 Windows）
	if osName != "windows" {
		if err := os.Chmod(installPath, 0755); err != nil {
			return "", fmt.Errorf("failed to set executable permission: %w", err)
		}
	}

	// 验证安装
	if err := verifyInstallation(installPath); err != nil {
		return "", fmt.Errorf("failed to verify installation: %w", err)
	}
	logger.Info("installation verified")

	logger.Info("Qdrant installed successfully", "version", version, "path", installPath)
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
		// macOS 使用 tar.gz 格式（不是 zip）
		switch arch {
		case "x86_64":
			filename = "qdrant-x86_64-apple-darwin.tar.gz"
		case "arm64":
			filename = "qdrant-aarch64-apple-darwin.tar.gz"
		default:
			return "", fmt.Errorf("unsupported architecture for macOS: %s", arch)
		}
	case "linux":
		switch arch {
		case "x86_64":
			filename = "qdrant-x86_64-unknown-linux-musl.tar.gz"
		case "arm64":
			filename = "qdrant-aarch64-unknown-linux-musl.tar.gz"
		default:
			return "", fmt.Errorf("unsupported architecture for Linux: %s", arch)
		}
	default:
		return "", fmt.Errorf("unsupported OS: %s", osName)
	}

	return fmt.Sprintf("%s/%s/%s", baseURL, version, filename), nil
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
