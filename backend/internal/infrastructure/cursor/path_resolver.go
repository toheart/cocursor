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
// 输入:
//   - Windows: "file:///d%3A/code/cocursor" 或 "file:///d:/code/cocursor"
//   - macOS/Linux: "file:///Users/xibaobao/code/cocursor"
//
// 输出:
//   - Windows: "D:/code/cocursor" 或 "D:\\code\\cocursor" (取决于系统)
//   - macOS/Linux: "/Users/xibaobao/code/cocursor"
func (p *PathResolver) parseFolderURI(uri string) (string, error) {
	// 检查 URI 是否为空
	if uri == "" {
		return "", fmt.Errorf("empty URI")
	}

	// 解析 URI
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("failed to parse URI: %w", err)
	}

	// 检查协议
	if parsedURL.Scheme == "" {
		return "", fmt.Errorf("missing scheme in URI: %s", uri)
	}
	if parsedURL.Scheme != "file" {
		return "", fmt.Errorf("unsupported scheme: %s", parsedURL.Scheme)
	}

	// 获取路径部分（url.Parse 会自动解码 %3A 为 :）
	path := parsedURL.Path

	// 手动处理 URL 编码（以防 url.Parse 没有完全解码）
	decodedPath, err := url.PathUnescape(path)
	if err != nil {
		// 如果解码失败，使用原始路径
		decodedPath = path
	}

	// 区分 Windows 和 Unix 路径
	// Windows 路径格式: file:///d:/code/cocursor -> Path = "/d:/code/cocursor"
	// macOS/Linux 路径格式: file:///Users/... -> Path = "/Users/..."
	//
	// Windows 路径特征: 第三个字符是 ':' (如 "/d:/..." 或 "/D:/...")
	// 索引: 0='/', 1='d', 2=':'
	// Unix 路径特征: 以 "/" 开头，第三个字符不是 ':'
	if len(decodedPath) > 2 && decodedPath[0] == '/' && decodedPath[2] == ':' {
		// Windows 路径: 移除开头的斜杠
		// file:///d:/code/cocursor -> /d:/code/cocursor -> d:/code/cocursor
		decodedPath = decodedPath[1:]
	}
	// macOS/Linux 路径: 保留开头的斜杠
	// file:///Users/... -> /Users/... (保持不变)

	// 转换为系统路径格式（Windows 会使用反斜杠，Unix 保持正斜杠）
	systemPath := filepath.FromSlash(decodedPath)

	// 清理路径：移除开头的单个反斜杠（仅 Windows，且非 UNC 路径）
	// Windows 路径问题：file:///c%3A/... 解析后可能变成 \c:\...
	// 例如：\c:\Users\... -> c:\Users\...
	// 但 UNC 路径 \\server\share 需要保留双反斜杠
	if runtime.GOOS == "windows" && len(systemPath) > 0 && systemPath[0] == '\\' {
		// 检查是否是 UNC 路径（\\server\share），如果是则保留
		if len(systemPath) <= 1 || systemPath[1] != '\\' {
			// 单个反斜杠，移除它
			systemPath = systemPath[1:]
		}
		// 如果是 UNC 路径（双反斜杠），保持不变
	}

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

// GetCursorProjectsDir 获取 Cursor 项目目录（存放 agent-transcripts）
// Windows: %USERPROFILE%\.cursor\projects
// macOS: ~/.cursor/projects
// Linux: ~/.cursor/projects
func (p *PathResolver) GetCursorProjectsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	projectsDir := filepath.Join(homeDir, ".cursor", "projects")

	// 检查目录是否存在
	if _, err := os.Stat(projectsDir); err != nil {
		return "", fmt.Errorf("cursor projects directory not found at %s: %w", projectsDir, err)
	}

	return projectsDir, nil
}

// GetCursorProjectsDirOrDefault 获取 Cursor 项目目录，如果不存在返回默认路径（不报错）
// 适用于需要容错的场景
func (p *PathResolver) GetCursorProjectsDirOrDefault() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".cursor", "projects")
}
