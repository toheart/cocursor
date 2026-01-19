package cursor

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
)

// ProjectDiscovery 项目发现器
type ProjectDiscovery struct {
	pathResolver *infraCursor.PathResolver
	gitReader    *infraCursor.GitReader
}

// NewProjectDiscovery 创建项目发现器实例
func NewProjectDiscovery() *ProjectDiscovery {
	return &ProjectDiscovery{
		pathResolver: infraCursor.NewPathResolver(),
		gitReader:    infraCursor.NewGitReader(),
	}
}

// DiscoveredWorkspace 发现的工作区信息
type DiscoveredWorkspace struct {
	WorkspaceID  string
	Path         string
	ProjectName  string
	GitRemoteURL string
	GitBranch    string
}

// ScanAllWorkspaces 扫描所有 Cursor 工作区
// 返回: 发现的工作区列表和错误
func (d *ProjectDiscovery) ScanAllWorkspaces() ([]*DiscoveredWorkspace, error) {
	// 获取工作区存储目录
	workspaceDir, err := d.pathResolver.GetWorkspaceStorageDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace storage directory: %w", err)
	}

	// 读取所有子目录
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace storage directory: %w", err)
	}

	var workspaces []*DiscoveredWorkspace

	// 遍历每个工作区目录
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
		// 使用与 PathResolver.GetWorkspaceIDByPath 相同的逻辑
		folderPath, err := d.parseFolderURI(workspace.Folder)
		if err != nil {
			continue
		}

		// 提取项目名（路径最后一个目录名）
		projectName := filepath.Base(folderPath)

		// 读取 Git 信息（如果存在）
		gitRemoteURL, _ := d.gitReader.ReadGitRemoteURL(folderPath)
		gitBranch, _ := d.gitReader.ReadGitBranch(folderPath)

		workspaces = append(workspaces, &DiscoveredWorkspace{
			WorkspaceID:  workspaceID,
			Path:         folderPath,
			ProjectName:  projectName,
			GitRemoteURL: gitRemoteURL,
			GitBranch:    gitBranch,
		})
	}

	return workspaces, nil
}

// parseFolderURI 解析 folder URI 为文件系统路径（与 PathResolver 逻辑相同）
func (d *ProjectDiscovery) parseFolderURI(uri string) (string, error) {
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

	// 获取路径部分
	path := parsedURL.Path

	// 手动处理 URL 编码
	decodedPath, err := url.PathUnescape(path)
	if err != nil {
		decodedPath = path
	}

	// 区分 Windows 和 Unix 路径
	if len(decodedPath) > 2 && decodedPath[1] == ':' {
		// Windows 路径: 移除开头的斜杠
		if len(decodedPath) > 0 && decodedPath[0] == '/' {
			decodedPath = decodedPath[1:]
		}
	}

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
