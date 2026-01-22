package cursor

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/cocursor/backend/internal/infrastructure/config"
)

// PathResolver 路径解析器，用于定位 Cursor 数据库和工作区
type PathResolver struct {
	// 用户自定义的路径配置（可选）
	customUserDataDir string
	customProjectsDir string
}

// 全局路径配置（通过环境变量或配置文件设置）
var (
	globalConfig     *config.CursorConfig
	globalConfigOnce sync.Once
	globalConfigMu   sync.RWMutex
)

// SetGlobalCursorConfig 设置全局 Cursor 路径配置
// 通常在应用启动时调用一次
func SetGlobalCursorConfig(cfg *config.CursorConfig) {
	globalConfigMu.Lock()
	defer globalConfigMu.Unlock()
	globalConfig = cfg
}

// GetGlobalCursorConfig 获取全局 Cursor 路径配置
func GetGlobalCursorConfig() *config.CursorConfig {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	return globalConfig
}

// NewPathResolver 创建路径解析器实例
func NewPathResolver() *PathResolver {
	return &PathResolver{}
}

// NewPathResolverWithConfig 使用自定义配置创建路径解析器
func NewPathResolverWithConfig(userDataDir, projectsDir string) *PathResolver {
	return &PathResolver{
		customUserDataDir: userDataDir,
		customProjectsDir: projectsDir,
	}
}

// PathNotFoundError 路径未找到错误
// 用于向前端提供详细的错误信息，帮助用户配置正确的路径
type PathNotFoundError struct {
	// PathType 路径类型: "user_data_dir", "projects_dir", "global_storage", "workspace_storage"
	PathType string `json:"path_type"`

	// AttemptedPath 尝试访问的单个路径（用于自定义配置场景）
	AttemptedPath string `json:"attempted_path,omitempty"`

	// AttemptedPaths 尝试访问的所有路径（用于自动检测场景）
	AttemptedPaths []string `json:"attempted_paths,omitempty"`

	// IsCustom 是否是用户自定义的路径
	IsCustom bool `json:"is_custom"`

	// IsWSL 是否在 WSL 环境中
	IsWSL bool `json:"is_wsl"`

	// Hint 给用户的提示信息
	Hint string `json:"hint"`
}

// Error 实现 error 接口
func (e *PathNotFoundError) Error() string {
	if e.AttemptedPath != "" {
		return fmt.Sprintf("Cursor %s not found at %s: %s", e.PathType, e.AttemptedPath, e.Hint)
	}
	if len(e.AttemptedPaths) > 0 {
		return fmt.Sprintf("Cursor %s not found (tried: %v): %s", e.PathType, e.AttemptedPaths, e.Hint)
	}
	return fmt.Sprintf("Cursor %s not found: %s", e.PathType, e.Hint)
}

// IsPathNotFoundError 检查错误是否是 PathNotFoundError
func IsPathNotFoundError(err error) bool {
	_, ok := err.(*PathNotFoundError)
	return ok
}

// AsPathNotFoundError 将错误转换为 PathNotFoundError
func AsPathNotFoundError(err error) (*PathNotFoundError, bool) {
	pnf, ok := err.(*PathNotFoundError)
	return pnf, ok
}

// getPathConfigHint 获取路径配置提示
func (p *PathResolver) getPathConfigHint(pathType string) string {
	if p.isWSL() {
		switch pathType {
		case "user_data_dir":
			return "在 WSL 环境中，无法自动检测 Cursor 数据目录。请设置环境变量 CURSOR_USER_DATA_DIR，例如: export CURSOR_USER_DATA_DIR=/mnt/c/Users/<你的Windows用户名>/AppData/Roaming/Cursor/User"
		case "projects_dir":
			return "在 WSL 环境中，无法自动检测 Cursor 项目目录。请设置环境变量 CURSOR_PROJECTS_DIR，例如: export CURSOR_PROJECTS_DIR=/mnt/c/Users/<你的Windows用户名>/.cursor/projects"
		}
	}

	switch pathType {
	case "user_data_dir":
		return "Cursor 用户数据目录不存在，请确保 Cursor 已安装并至少运行过一次。如需手动配置，请设置环境变量 CURSOR_USER_DATA_DIR"
	case "projects_dir":
		return "Cursor 项目目录不存在，请确保 Cursor 已安装并至少运行过一次。如需手动配置，请设置环境变量 CURSOR_PROJECTS_DIR"
	}
	return "路径不存在，请检查 Cursor 是否已正确安装"
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
// 优先级：1. 实例自定义路径 2. 全局配置 3. 环境变量 CURSOR_USER_DATA_DIR 4. 自动检测
// Windows: %APPDATA%/Cursor/User
// macOS: ~/Library/Application Support/Cursor/User
// Linux: ~/.config/Cursor/User
// WSL: /mnt/c/Users/<username>/AppData/Roaming/Cursor/User（访问 Windows 宿主机）
func (p *PathResolver) getUserDataDir() (string, error) {
	// 1. 检查实例自定义路径
	if p.customUserDataDir != "" {
		if _, err := os.Stat(p.customUserDataDir); err == nil {
			return p.customUserDataDir, nil
		}
		return "", &PathNotFoundError{
			PathType:    "user_data_dir",
			AttemptedPath: p.customUserDataDir,
			IsCustom:    true,
			Hint:        "配置的 Cursor 用户数据目录不存在，请检查路径是否正确",
		}
	}

	// 2. 检查全局配置
	if cfg := GetGlobalCursorConfig(); cfg != nil && cfg.UserDataDir != "" {
		if _, err := os.Stat(cfg.UserDataDir); err == nil {
			return cfg.UserDataDir, nil
		}
		return "", &PathNotFoundError{
			PathType:    "user_data_dir",
			AttemptedPath: cfg.UserDataDir,
			IsCustom:    true,
			Hint:        "配置的 Cursor 用户数据目录不存在，请检查路径是否正确",
		}
	}

	// 3. 检查环境变量
	if envPath := os.Getenv("CURSOR_USER_DATA_DIR"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
		return "", &PathNotFoundError{
			PathType:    "user_data_dir",
			AttemptedPath: envPath,
			IsCustom:    true,
			Hint:        "环境变量 CURSOR_USER_DATA_DIR 指定的路径不存在",
		}
	}

	// 4. 自动检测
	return p.autoDetectUserDataDir()
}

// autoDetectUserDataDir 自动检测 Cursor 用户数据目录
func (p *PathResolver) autoDetectUserDataDir() (string, error) {
	var userDataDir string
	var attemptedPaths []string

	switch runtime.GOOS {
	case "windows":
		// Windows
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", &PathNotFoundError{
				PathType: "user_data_dir",
				Hint:     "APPDATA 环境变量未设置",
			}
		}
		userDataDir = filepath.Join(appData, "Cursor", "User")
		attemptedPaths = append(attemptedPaths, userDataDir)
	case "darwin":
		// macOS
		home := os.Getenv("HOME")
		if home == "" {
			return "", &PathNotFoundError{
				PathType: "user_data_dir",
				Hint:     "HOME 环境变量未设置",
			}
		}
		userDataDir = filepath.Join(home, "Library", "Application Support", "Cursor", "User")
		attemptedPaths = append(attemptedPaths, userDataDir)
	default:
		// Linux（包括 WSL）
		// 首先检查是否在 WSL 环境中
		if p.isWSL() {
			// WSL 环境：尝试访问 Windows 宿主机的 Cursor 数据
			windowsUserDataDir, err := p.getWindowsUserDataDirFromWSL()
			if err == nil {
				return windowsUserDataDir, nil
			}
			// 记录尝试的 Windows 路径
			attemptedPaths = append(attemptedPaths, "/mnt/c/Users/<username>/AppData/Roaming/Cursor/User")
		}

		// 原生 Linux 或 WSL 回退
		home := os.Getenv("HOME")
		if home == "" {
			return "", &PathNotFoundError{
				PathType:       "user_data_dir",
				AttemptedPaths: attemptedPaths,
				IsWSL:          p.isWSL(),
				Hint:           "HOME 环境变量未设置",
			}
		}
		userDataDir = filepath.Join(home, ".config", "Cursor", "User")
		attemptedPaths = append(attemptedPaths, userDataDir)
	}

	// 检查路径是否存在
	if _, err := os.Stat(userDataDir); err != nil {
		return "", &PathNotFoundError{
			PathType:       "user_data_dir",
			AttemptedPaths: attemptedPaths,
			IsWSL:          p.isWSL(),
			Hint:           p.getPathConfigHint("user_data_dir"),
		}
	}

	return userDataDir, nil
}

// isWSL 检测是否在 WSL 环境中运行
func (p *PathResolver) isWSL() bool {
	// 方法1：检查 /proc/version 是否包含 Microsoft 或 WSL
	if data, err := os.ReadFile("/proc/version"); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "microsoft") || strings.Contains(content, "wsl") {
			return true
		}
	}

	// 方法2：检查 WSL 特有的环境变量
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSLENV") != "" {
		return true
	}

	return false
}

// getWindowsUserDataDirFromWSL 在 WSL 环境中获取 Windows Cursor 用户数据目录
func (p *PathResolver) getWindowsUserDataDirFromWSL() (string, error) {
	// 方法1：通过 USERPROFILE 环境变量（如果设置了 WSLENV）
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		// 将 Windows 路径转换为 WSL 路径
		// C:\Users\xxx -> /mnt/c/Users/xxx
		wslPath := p.windowsPathToWSL(userProfile)
		cursorPath := filepath.Join(wslPath, "AppData", "Roaming", "Cursor", "User")
		if _, err := os.Stat(cursorPath); err == nil {
			return cursorPath, nil
		}
	}

	// 方法2：扫描 /mnt/c/Users/ 目录查找 Cursor 配置
	usersDir := "/mnt/c/Users"
	if entries, err := os.ReadDir(usersDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			// 跳过系统用户目录
			name := entry.Name()
			if name == "Public" || name == "Default" || name == "Default User" || name == "All Users" {
				continue
			}

			cursorPath := filepath.Join(usersDir, name, "AppData", "Roaming", "Cursor", "User")
			if _, err := os.Stat(cursorPath); err == nil {
				return cursorPath, nil
			}
		}
	}

	return "", fmt.Errorf("cannot find Windows Cursor data directory from WSL")
}

// windowsPathToWSL 将 Windows 路径转换为 WSL 路径
// 例如: C:\Users\xxx -> /mnt/c/Users/xxx
func (p *PathResolver) windowsPathToWSL(windowsPath string) string {
	// 替换反斜杠为正斜杠
	path := strings.ReplaceAll(windowsPath, "\\", "/")

	// 处理盘符: C:/... -> /mnt/c/...
	if len(path) >= 2 && path[1] == ':' {
		driveLetter := strings.ToLower(string(path[0]))
		path = "/mnt/" + driveLetter + path[2:]
	}

	return path
}

// GetCursorProjectsDir 获取 Cursor 项目目录（存放 agent-transcripts）
// 优先级：1. 实例自定义路径 2. 全局配置 3. 环境变量 CURSOR_PROJECTS_DIR 4. 自动检测
// Windows: %USERPROFILE%\.cursor\projects
// macOS: ~/.cursor/projects
// Linux: ~/.cursor/projects
// WSL: /mnt/c/Users/<username>/.cursor/projects（访问 Windows 宿主机）
func (p *PathResolver) GetCursorProjectsDir() (string, error) {
	// 1. 检查实例自定义路径
	if p.customProjectsDir != "" {
		if _, err := os.Stat(p.customProjectsDir); err == nil {
			return p.customProjectsDir, nil
		}
		return "", &PathNotFoundError{
			PathType:      "projects_dir",
			AttemptedPath: p.customProjectsDir,
			IsCustom:      true,
			Hint:          "配置的 Cursor 项目目录不存在，请检查路径是否正确",
		}
	}

	// 2. 检查全局配置
	if cfg := GetGlobalCursorConfig(); cfg != nil && cfg.ProjectsDir != "" {
		if _, err := os.Stat(cfg.ProjectsDir); err == nil {
			return cfg.ProjectsDir, nil
		}
		return "", &PathNotFoundError{
			PathType:      "projects_dir",
			AttemptedPath: cfg.ProjectsDir,
			IsCustom:      true,
			Hint:          "配置的 Cursor 项目目录不存在，请检查路径是否正确",
		}
	}

	// 3. 检查环境变量
	if envPath := os.Getenv("CURSOR_PROJECTS_DIR"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
		return "", &PathNotFoundError{
			PathType:      "projects_dir",
			AttemptedPath: envPath,
			IsCustom:      true,
			Hint:          "环境变量 CURSOR_PROJECTS_DIR 指定的路径不存在",
		}
	}

	// 4. 自动检测
	return p.autoDetectProjectsDir()
}

// autoDetectProjectsDir 自动检测 Cursor 项目目录
func (p *PathResolver) autoDetectProjectsDir() (string, error) {
	var attemptedPaths []string

	// 在 WSL 环境中，尝试使用 Windows 宿主机的路径
	if runtime.GOOS == "linux" && p.isWSL() {
		windowsProjectsDir, err := p.getWindowsProjectsDirFromWSL()
		if err == nil {
			return windowsProjectsDir, nil
		}
		// 记录尝试的路径
		attemptedPaths = append(attemptedPaths, "/mnt/c/Users/<username>/.cursor/projects")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", &PathNotFoundError{
			PathType:       "projects_dir",
			AttemptedPaths: attemptedPaths,
			IsWSL:          p.isWSL(),
			Hint:           "无法获取用户主目录",
		}
	}

	projectsDir := filepath.Join(homeDir, ".cursor", "projects")
	attemptedPaths = append(attemptedPaths, projectsDir)

	// 检查目录是否存在
	if _, err := os.Stat(projectsDir); err != nil {
		return "", &PathNotFoundError{
			PathType:       "projects_dir",
			AttemptedPaths: attemptedPaths,
			IsWSL:          p.isWSL(),
			Hint:           p.getPathConfigHint("projects_dir"),
		}
	}

	return projectsDir, nil
}

// GetCursorProjectsDirOrDefault 获取 Cursor 项目目录，如果不存在返回默认路径（不报错）
// 适用于需要容错的场景
func (p *PathResolver) GetCursorProjectsDirOrDefault() string {
	// 尝试使用 GetCursorProjectsDir，如果成功则返回
	if dir, err := p.GetCursorProjectsDir(); err == nil {
		return dir
	}

	// 回退到默认路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".cursor", "projects")
}

// getWindowsProjectsDirFromWSL 在 WSL 环境中获取 Windows Cursor 项目目录
func (p *PathResolver) getWindowsProjectsDirFromWSL() (string, error) {
	// 方法1：通过 USERPROFILE 环境变量（如果设置了 WSLENV）
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		wslPath := p.windowsPathToWSL(userProfile)
		projectsDir := filepath.Join(wslPath, ".cursor", "projects")
		if _, err := os.Stat(projectsDir); err == nil {
			return projectsDir, nil
		}
	}

	// 方法2：扫描 /mnt/c/Users/ 目录查找 .cursor/projects
	usersDir := "/mnt/c/Users"
	if entries, err := os.ReadDir(usersDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			// 跳过系统用户目录
			name := entry.Name()
			if name == "Public" || name == "Default" || name == "Default User" || name == "All Users" {
				continue
			}

			projectsDir := filepath.Join(usersDir, name, ".cursor", "projects")
			if _, err := os.Stat(projectsDir); err == nil {
				return projectsDir, nil
			}
		}
	}

	return "", fmt.Errorf("cannot find Windows Cursor projects directory from WSL")
}

// GetPathStatus 获取当前路径配置状态（用于前端展示）
func (p *PathResolver) GetPathStatus() *PathStatus {
	status := &PathStatus{
		IsWSL: p.isWSL(),
	}

	// 检查 UserDataDir
	if userDataDir, err := p.getUserDataDir(); err == nil {
		status.UserDataDir = userDataDir
		status.UserDataDirOK = true
	} else {
		if pnf, ok := AsPathNotFoundError(err); ok {
			status.UserDataDirError = pnf
		} else {
			status.UserDataDirError = &PathNotFoundError{
				PathType: "user_data_dir",
				Hint:     err.Error(),
			}
		}
	}

	// 检查 ProjectsDir
	if projectsDir, err := p.GetCursorProjectsDir(); err == nil {
		status.ProjectsDir = projectsDir
		status.ProjectsDirOK = true
	} else {
		if pnf, ok := AsPathNotFoundError(err); ok {
			status.ProjectsDirError = pnf
		} else {
			status.ProjectsDirError = &PathNotFoundError{
				PathType: "projects_dir",
				Hint:     err.Error(),
			}
		}
	}

	// 检查环境变量配置
	status.EnvUserDataDir = os.Getenv("CURSOR_USER_DATA_DIR")
	status.EnvProjectsDir = os.Getenv("CURSOR_PROJECTS_DIR")

	// 检查全局配置
	if cfg := GetGlobalCursorConfig(); cfg != nil {
		status.ConfigUserDataDir = cfg.UserDataDir
		status.ConfigProjectsDir = cfg.ProjectsDir
	}

	return status
}

// PathStatus 路径配置状态
type PathStatus struct {
	// 当前检测到的路径
	UserDataDir   string `json:"user_data_dir,omitempty"`
	UserDataDirOK bool   `json:"user_data_dir_ok"`
	ProjectsDir   string `json:"projects_dir,omitempty"`
	ProjectsDirOK bool   `json:"projects_dir_ok"`

	// 错误信息
	UserDataDirError *PathNotFoundError `json:"user_data_dir_error,omitempty"`
	ProjectsDirError *PathNotFoundError `json:"projects_dir_error,omitempty"`

	// 环境变量配置
	EnvUserDataDir string `json:"env_user_data_dir,omitempty"`
	EnvProjectsDir string `json:"env_projects_dir,omitempty"`

	// 全局配置
	ConfigUserDataDir string `json:"config_user_data_dir,omitempty"`
	ConfigProjectsDir string `json:"config_projects_dir,omitempty"`

	// 环境信息
	IsWSL bool `json:"is_wsl"`
}
