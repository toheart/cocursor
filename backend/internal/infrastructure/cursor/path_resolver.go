package cursor

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// PathResolver 路径解析器，用于定位 Cursor 数据库和工作区
type PathResolver struct{}

// NewPathResolver 创建路径解析器实例
func NewPathResolver() *PathResolver {
	return &PathResolver{}
}

// GetGlobalStoragePath 获取全局存储数据库路径
// Windows: %APPDATA%/Cursor/User/globalStorage/state.vscdb
// macOS: ~/Library/Application Support/Cursor/User/globalStorage/state.vscdb
// Linux: ~/.config/Cursor/User/globalStorage/state.vscdb
func (p *PathResolver) GetGlobalStoragePath() (string, error) {
	userDataDir, err := p.getUserDataDir()
	if err != nil {
		return "", err
	}

	basePath := filepath.Join(userDataDir, "globalStorage", "state.vscdb")

	// 检查文件是否存在
	if _, err := os.Stat(basePath); err != nil {
		return "", fmt.Errorf("global storage database not found at %s: %w", basePath, err)
	}

	return basePath, nil
}

// GetWorkspaceStorageDir 获取工作区存储根目录
// Windows: %APPDATA%/Cursor/User/workspaceStorage
// macOS: ~/Library/Application Support/Cursor/User/workspaceStorage
// Linux: ~/.config/Cursor/User/workspaceStorage
func (p *PathResolver) GetWorkspaceStorageDir() (string, error) {
	userDataDir, err := p.getUserDataDir()
	if err != nil {
		return "", err
	}

	workspaceDir := filepath.Join(userDataDir, "workspaceStorage")

	// 检查目录是否存在
	if _, err := os.Stat(workspaceDir); err != nil {
		return "", fmt.Errorf("workspace storage directory not found at %s: %w", workspaceDir, err)
	}

	return workspaceDir, nil
}

// GetWorkspaceIDByPath 根据项目路径查找工作区 ID
// projectPath: 项目路径，如 "D:/code/cocursor" 或 "D:\\code\\cocursor"
// 返回: 工作区 ID（哈希值），如 "d4b798d47e9a14d74eb7965f996e8739"
func (p *PathResolver) GetWorkspaceIDByPath(projectPath string) (string, error) {
	// 规范化项目路径
	normalizedPath, err := p.normalizePath(projectPath)
	if err != nil {
		return "", fmt.Errorf("failed to normalize path: %w", err)
	}

	// 获取工作区存储目录
	workspaceDir, err := p.GetWorkspaceStorageDir()
	if err != nil {
		return "", err
	}

	// 遍历所有子目录
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return "", fmt.Errorf("failed to read workspace storage directory: %w", err)
	}

	// 读取每个子目录里的 workspace.json
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workspaceID := entry.Name()
		workspaceJSONPath := filepath.Join(workspaceDir, workspaceID, "workspace.json")

		// 读取 workspace.json
		data, err := os.ReadFile(workspaceJSONPath)
		if err != nil {
			// 如果文件不存在，跳过
			continue
		}

		// 解析 JSON
		var workspace struct {
			Folder string `json:"folder"`
		}
		if err := json.Unmarshal(data, &workspace); err != nil {
			continue
		}

		// 解析 folder URI 并转换为文件系统路径
		folderPath, err := p.parseFolderURI(workspace.Folder)
		if err != nil {
			continue
		}

		// 规范化 folder 路径
		normalizedFolderPath, err := p.normalizePath(folderPath)
		if err != nil {
			continue
		}

		// 比较路径（不区分大小写，因为 Windows 路径不区分大小写）
		if strings.EqualFold(normalizedPath, normalizedFolderPath) {
			return workspaceID, nil
		}
	}

	return "", fmt.Errorf("workspace not found for path: %s", projectPath)
}

// normalizePath 规范化路径
// 将路径转换为统一格式，便于比较
func (p *PathResolver) normalizePath(path string) (string, error) {
	// 转换为绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// 统一使用正斜杠（Windows 也支持）
	normalized := filepath.ToSlash(absPath)

	// 移除末尾的斜杠
	normalized = strings.TrimSuffix(normalized, "/")

	return normalized, nil
}

// parseFolderURI 解析 folder URI 为文件系统路径
// 输入: "file:///d%3A/code/cocursor"
// 输出: "D:/code/cocursor" 或 "D:\\code\\cocursor" (取决于系统)
func (p *PathResolver) parseFolderURI(uri string) (string, error) {
	// 解析 URI
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("failed to parse URI: %w", err)
	}

	// 检查协议
	if parsedURL.Scheme != "file" {
		return "", fmt.Errorf("unsupported scheme: %s", parsedURL.Scheme)
	}

	// 获取路径部分（url.Parse 会自动解码 %3A 为 :）
	path := parsedURL.Path

	// Windows 路径格式: file:///d%3A/code/cocursor
	// url.Parse 会解析为: Path = "/d:/code/cocursor" (如果 %3A 被解码)
	// 或者: Path = "/d%3A/code/cocursor" (如果 %3A 没有被解码)

	// 手动处理 URL 编码（以防 url.Parse 没有完全解码）
	decodedPath, err := url.PathUnescape(path)
	if err != nil {
		// 如果解码失败，使用原始路径
		decodedPath = path
	}

	// Windows 路径格式: file:///d:/code/cocursor -> /d:/code/cocursor
	// 需要移除开头的斜杠（如果存在）
	if len(decodedPath) > 0 && decodedPath[0] == '/' {
		decodedPath = decodedPath[1:]
	}

	// 转换为系统路径格式（Windows 会使用反斜杠）
	systemPath := filepath.FromSlash(decodedPath)

	return systemPath, nil
}

// GetWorkspaceDBPath 获取工作区数据库路径
// workspaceID: 工作区 ID（哈希值）
// 返回: 工作区数据库路径，如 %APPDATA%/Cursor/User/workspaceStorage/{workspaceId}/state.vscdb
func (p *PathResolver) GetWorkspaceDBPath(workspaceID string) (string, error) {
	workspaceDir, err := p.GetWorkspaceStorageDir()
	if err != nil {
		return "", err
	}

	dbPath := filepath.Join(workspaceDir, workspaceID, "state.vscdb")

	// 检查文件是否存在
	if _, err := os.Stat(dbPath); err != nil {
		return "", fmt.Errorf("workspace database not found at %s: %w", dbPath, err)
	}

	return dbPath, nil
}

// getUserDataDir 获取 Cursor 用户数据目录
// Windows: %APPDATA%/Cursor/User
// macOS: ~/Library/Application Support/Cursor/User
// Linux: ~/.config/Cursor/User
func (p *PathResolver) getUserDataDir() (string, error) {
	var userDataDir string

	switch runtime.GOOS {
	case "windows":
		// Windows
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		userDataDir = filepath.Join(appData, "Cursor", "User")
	case "darwin":
		// macOS
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("HOME environment variable not set")
		}
		userDataDir = filepath.Join(home, "Library", "Application Support", "Cursor", "User")
	default:
		// Linux
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("HOME environment variable not set")
		}
		userDataDir = filepath.Join(home, ".config", "Cursor", "User")
	}
	return userDataDir, nil
}
