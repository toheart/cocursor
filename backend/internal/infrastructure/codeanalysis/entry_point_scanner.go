package codeanalysis

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cocursor/backend/internal/domain/codeanalysis"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// EntryPointScanner 入口函数扫描器实现
type EntryPointScanner struct {
	logger *slog.Logger
}

// NewEntryPointScanner 创建入口函数扫描器
func NewEntryPointScanner() *EntryPointScanner {
	return &EntryPointScanner{
		logger: log.NewModuleLogger("codeanalysis", "entry_point_scanner"),
	}
}

// ScanEntryPoints 扫描项目中的入口函数候选
func (s *EntryPointScanner) ScanEntryPoints(ctx context.Context, projectPath string) ([]codeanalysis.EntryPointCandidate, error) {
	var candidates []codeanalysis.EntryPointCandidate

	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 策略 1: 扫描 cmd/*/main.go
	cmdPattern := filepath.Join(absPath, "cmd", "*", "main.go")
	cmdFiles, _ := filepath.Glob(cmdPattern)
	for _, f := range cmdFiles {
		relPath, _ := filepath.Rel(absPath, f)
		candidates = append(candidates, codeanalysis.EntryPointCandidate{
			File:        relPath,
			Function:    "main",
			Type:        "cmd",
			Priority:    1,
			Recommended: true,
		})
	}

	// 策略 2: 根目录 main.go
	rootMain := filepath.Join(absPath, "main.go")
	if _, err := os.Stat(rootMain); err == nil {
		candidates = append(candidates, codeanalysis.EntryPointCandidate{
			File:        "main.go",
			Function:    "main",
			Type:        "root",
			Priority:    2,
			Recommended: len(cmdFiles) == 0, // 如果没有 cmd，则推荐
		})
	}

	// 策略 3: 扫描 cmd/ 下的其他模式（如 cmd/tool/tool.go）
	cmdDirs, _ := filepath.Glob(filepath.Join(absPath, "cmd", "*"))
	for _, dir := range cmdDirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			dirName := filepath.Base(dir)
			// 检查是否有同名 .go 文件包含 main 函数
			mainFile := filepath.Join(dir, dirName+".go")
			if hasMainFunc(mainFile) {
				relPath, _ := filepath.Rel(absPath, mainFile)
				// 避免重复
				if !containsCandidate(candidates, relPath) {
					candidates = append(candidates, codeanalysis.EntryPointCandidate{
						File:        relPath,
						Function:    "main",
						Type:        "cmd",
						Priority:    1,
						Recommended: true,
					})
				}
			}
		}
	}

	// 如果没有找到任何入口函数，提供"分析所有导出函数"选项
	if len(candidates) == 0 {
		candidates = append(candidates, codeanalysis.EntryPointCandidate{
			File:        "*",
			Function:    "*",
			Type:        "all_exported",
			Priority:    3,
			Recommended: true,
		})
	}

	return candidates, nil
}

// GoModuleValidation Go 模块验证结果
type GoModuleValidation struct {
	Valid      bool   `json:"valid"`
	ModulePath string `json:"module_path,omitempty"`
	GoModPath  string `json:"go_mod_path,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ValidateGoModule 验证目录是否是有效的 Go 模块
func (s *EntryPointScanner) ValidateGoModule(_ context.Context, projectPath string) *GoModuleValidation {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return &GoModuleValidation{
			Valid: false,
			Error: fmt.Sprintf("invalid path: %v", err),
		}
	}

	// 检查目录是否存在
	info, err := os.Stat(absPath)
	if err != nil {
		return &GoModuleValidation{
			Valid: false,
			Error: fmt.Sprintf("directory does not exist: %v", err),
		}
	}
	if !info.IsDir() {
		return &GoModuleValidation{
			Valid: false,
			Error: "path is not a directory",
		}
	}

	// 查找 go.mod 文件
	goModPath := filepath.Join(absPath, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		// 向上查找，但记录实际的 go.mod 位置
		foundGoMod := ""
		for dir := filepath.Dir(absPath); dir != "/" && dir != "."; dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, "go.mod")
			if _, err := os.Stat(candidate); err == nil {
				foundGoMod = candidate
				break
			}
		}

		if foundGoMod != "" {
			// go.mod 在父目录，需要告知用户
			return &GoModuleValidation{
				Valid: false,
				Error: fmt.Sprintf("go.mod not found in project directory. Found go.mod at parent directory: %s. Please use the module root directory, or ensure the project has its own go.mod file.", foundGoMod),
			}
		}

		return &GoModuleValidation{
			Valid: false,
			Error: "go.mod not found. This directory is not a Go module. Please run 'go mod init' to initialize a Go module.",
		}
	}

	// 读取 go.mod 获取模块路径
	file, err := os.Open(goModPath)
	if err != nil {
		return &GoModuleValidation{
			Valid:     false,
			GoModPath: goModPath,
			Error:     fmt.Sprintf("failed to read go.mod: %v", err),
		}
	}
	defer file.Close()

	modulePath := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			modulePath = strings.TrimPrefix(line, "module ")
			break
		}
	}

	if modulePath == "" {
		return &GoModuleValidation{
			Valid:     false,
			GoModPath: goModPath,
			Error:     "go.mod exists but 'module' directive not found. The go.mod file may be corrupted.",
		}
	}

	// 检查是否有 Go 源文件
	hasGoFiles := false
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略错误，继续遍历
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			// 排除 vendor 目录
			if !strings.Contains(path, "/vendor/") {
				hasGoFiles = true
				return filepath.SkipAll // 找到一个就够了
			}
		}
		return nil
	})

	if !hasGoFiles {
		return &GoModuleValidation{
			Valid:      false,
			ModulePath: modulePath,
			GoModPath:  goModPath,
			Error:      "go.mod found but no Go source files (.go) in the project. This may not be a valid Go project.",
		}
	}

	return &GoModuleValidation{
		Valid:      true,
		ModulePath: modulePath,
		GoModPath:  goModPath,
	}
}

// GetModulePath 获取项目的 go.mod 模块路径
func (s *EntryPointScanner) GetModulePath(_ context.Context, projectPath string) (string, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", err
	}

	// 查找 go.mod 文件
	goModPath := filepath.Join(absPath, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		// 向上查找
		for dir := absPath; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
			goModPath = filepath.Join(dir, "go.mod")
			if _, err := os.Stat(goModPath); err == nil {
				break
			}
		}
	}

	file, err := os.Open(goModPath)
	if err != nil {
		return "", fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}

	return "", fmt.Errorf("module directive not found in go.mod")
}

// GetRemoteURL 获取项目的 remote URL
func (s *EntryPointScanner) GetRemoteURL(ctx context.Context, projectPath string) (string, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", err
	}

	// 使用 git 命令获取 remote URL
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "git", "-C", absPath, "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		// 尝试从 .git/config 读取
		return s.readRemoteFromGitConfig(absPath)
	}

	return strings.TrimSpace(string(output)), nil
}

// readRemoteFromGitConfig 从 .git/config 读取 remote URL
func (s *EntryPointScanner) readRemoteFromGitConfig(projectPath string) (string, error) {
	gitConfigPath := filepath.Join(projectPath, ".git", "config")
	file, err := os.Open(gitConfigPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inRemoteOrigin := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "[remote \"origin\"]") {
			inRemoteOrigin = true
			continue
		}
		if strings.HasPrefix(line, "[") && inRemoteOrigin {
			break
		}
		if inRemoteOrigin && strings.HasPrefix(line, "url") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "", fmt.Errorf("remote origin not found")
}

// hasMainFunc 检查文件是否包含 main 函数
func hasMainFunc(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "func main()") {
			return true
		}
	}
	return false
}

// containsCandidate 检查候选列表中是否已包含指定文件
func containsCandidate(candidates []codeanalysis.EntryPointCandidate, file string) bool {
	for _, c := range candidates {
		if c.File == file {
			return true
		}
	}
	return false
}

// 确保实现接口
var _ codeanalysis.EntryPointScanner = (*EntryPointScanner)(nil)
