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
// 通过遍历项目所有 .go 文件，查找 package main + func main() 来发现入口函数，
// 不依赖固定的目录命名约定（如 cmd/、apps/）。
func (s *EntryPointScanner) ScanEntryPoints(ctx context.Context, projectPath string) ([]codeanalysis.EntryPointCandidate, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	var candidates []codeanalysis.EntryPointCandidate
	// 记录已发现的 main 包目录，同一目录下多个文件只生成一个候选
	seenDirs := make(map[string]bool)

	err = filepath.Walk(absPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil // 忽略单个文件错误，继续遍历
		}

		// 跳过不需要扫描的目录
		if info.IsDir() {
			dirName := info.Name()
			if dirName == "vendor" || dirName == "testdata" || dirName == "node_modules" || strings.HasPrefix(dirName, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// 只处理 .go 文件，排除测试文件
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// 检查文件是否同时包含 package main 和 func main()
		if !isMainPackageWithMainFunc(path) {
			return nil
		}

		dir := filepath.Dir(path)
		if seenDirs[dir] {
			return nil // 同一目录已经记录，跳过
		}
		seenDirs[dir] = true

		relPath, _ := filepath.Rel(absPath, path)
		entryType, priority := classifyEntryPoint(absPath, path)

		candidates = append(candidates, codeanalysis.EntryPointCandidate{
			File:        filepath.ToSlash(relPath),
			Function:    "main",
			Type:        entryType,
			Priority:    priority,
			Recommended: priority <= 2,
		})

		return nil
	})

	if err != nil {
		s.logger.Warn("error walking project directory", "error", err)
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

	// 按优先级排序
	sortCandidates(candidates)

	return candidates, nil
}

// isMainPackageWithMainFunc 检查文件是否属于 package main 且包含 func main()
func isMainPackageWithMainFunc(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	hasPackageMain := false
	hasFuncMain := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过注释行
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}

		if line == "package main" {
			hasPackageMain = true
		}
		if strings.HasPrefix(line, "func main()") {
			hasFuncMain = true
		}

		// 两者都找到就可以提前退出
		if hasPackageMain && hasFuncMain {
			return true
		}
	}

	return false
}

// classifyEntryPoint 根据文件在项目中的位置推断入口类型和优先级
// 优先级: 1(最高) - 常见入口目录(cmd/, apps/ 等)  2 - 根目录  3 - 其他位置
func classifyEntryPoint(projectRoot, filePath string) (entryType string, priority int) {
	relPath, _ := filepath.Rel(projectRoot, filePath)
	relPath = filepath.ToSlash(relPath)
	parts := strings.Split(relPath, "/")

	// 根目录下的 main.go
	if len(parts) == 1 {
		return "root", 2
	}

	// 第一级目录名用于判断类型
	topDir := strings.ToLower(parts[0])
	switch topDir {
	case "cmd", "apps", "app", "cmds", "command", "commands", "tools", "tool":
		return "cmd", 1
	case "bench", "benchmark", "benchmarks", "example", "examples", "hack", "test", "tests":
		return "auxiliary", 3
	}

	return "other", 2
}

// sortCandidates 按优先级排序候选列表（优先级数字越小越优先）
func sortCandidates(candidates []codeanalysis.EntryPointCandidate) {
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidates[j].Priority < candidates[j-1].Priority; j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}
}

// GoModuleValidation Go 模块验证结果
type GoModuleValidation struct {
	Valid      bool   `json:"valid"`
	ModulePath string `json:"module_path,omitempty"`
	GoModPath  string `json:"go_mod_path,omitempty"`
	// GoModDir go.mod 所在的目录（即实际的 Go 模块根目录）
	// 对于全栈项目，可能与传入的 projectPath 不同（如 go.mod 在 backend/ 子目录中）
	GoModDir string `json:"go_mod_dir,omitempty"`
	Error    string `json:"error,omitempty"`
}

// ValidateGoModule 验证目录是否是有效的 Go 模块
// 支持全栈项目：当根目录没有 go.mod 时，会自动向下搜索子目录（如 backend/）中的 go.mod
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
		// 1. 先向下搜索子目录中的 go.mod（支持全栈项目如 backend/go.mod）
		foundGoModInChild := s.findGoModInChildren(absPath)
		if foundGoModInChild != "" {
			goModDir := filepath.Dir(foundGoModInChild)
			return s.validateGoModAt(goModDir, foundGoModInChild)
		}

		// 2. 向上查找父目录中的 go.mod
		foundGoMod := ""
		prevDir := absPath
		for dir := filepath.Dir(absPath); dir != prevDir; dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, "go.mod")
			if _, err := os.Stat(candidate); err == nil {
				foundGoMod = candidate
				break
			}
			prevDir = dir
		}

		if foundGoMod != "" {
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

	// go.mod 就在当前目录
	return s.validateGoModAt(absPath, goModPath)
}

// findGoModInChildren 在子目录中查找 go.mod 文件
// 只搜索一级子目录（常见的全栈项目结构如 backend/、server/、go/）
// 如果找到多个，优先返回常见目录名（backend, server, go, api, service）
func (s *EntryPointScanner) findGoModInChildren(rootPath string) string {
	// 优先检查常见的 Go 子目录名
	commonDirs := []string{"backend", "server", "go", "api", "service", "services", "app", "src"}
	for _, dirName := range commonDirs {
		candidate := filepath.Join(rootPath, dirName, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			s.logger.Info("found go.mod in common subdirectory", "dir", dirName, "path", candidate)
			return candidate
		}
	}

	// 如果常见目录都没找到，遍历一级子目录
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// 跳过隐藏目录和不需要扫描的目录
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "frontend" || name == "web" || name == "dist" {
			continue
		}
		candidate := filepath.Join(rootPath, name, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			s.logger.Info("found go.mod in subdirectory", "dir", name, "path", candidate)
			return candidate
		}
	}

	return ""
}

// validateGoModAt 验证指定目录下的 go.mod 是否有效
func (s *EntryPointScanner) validateGoModAt(goModDir string, goModPath string) *GoModuleValidation {
	// 读取 go.mod 获取模块路径
	file, err := os.Open(goModPath)
	if err != nil {
		return &GoModuleValidation{
			Valid:     false,
			GoModPath: goModPath,
			GoModDir:  goModDir,
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
			GoModDir:  goModDir,
			Error:     "go.mod exists but 'module' directive not found. The go.mod file may be corrupted.",
		}
	}

	// 检查是否有 Go 源文件（在 go.mod 所在目录中查找）
	hasGoFiles := false
	_ = filepath.Walk(goModDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			dirName := info.Name()
			if dirName == "vendor" || dirName == "node_modules" || strings.HasPrefix(dirName, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			hasGoFiles = true
			return filepath.SkipAll
		}
		return nil
	})

	if !hasGoFiles {
		return &GoModuleValidation{
			Valid:      false,
			ModulePath: modulePath,
			GoModPath:  goModPath,
			GoModDir:   goModDir,
			Error:      "go.mod found but no Go source files (.go) in the project. This may not be a valid Go project.",
		}
	}

	return &GoModuleValidation{
		Valid:      true,
		ModulePath: modulePath,
		GoModPath:  goModPath,
		GoModDir:   goModDir,
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


// 确保实现接口
var _ codeanalysis.EntryPointScanner = (*EntryPointScanner)(nil)
