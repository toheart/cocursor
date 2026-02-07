package codeanalysis

import (
	"bufio"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cocursor/backend/internal/domain/codeanalysis"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// DiffAnalyzer Git diff 分析器实现
type DiffAnalyzer struct {
	logger *slog.Logger
}

// NewDiffAnalyzer 创建 Git diff 分析器
func NewDiffAnalyzer() *DiffAnalyzer {
	return &DiffAnalyzer{
		logger: log.NewModuleLogger("codeanalysis", "diff_analyzer"),
	}
}

// diffHunk 表示一个 diff 块
type diffHunk struct {
	File      string
	OldStart  int
	OldCount  int
	NewStart  int
	NewCount  int
	IsDeleted bool
	IsAdded   bool
}

// AnalyzeDiff 分析 Git diff，返回变更的函数列表
func (a *DiffAnalyzer) AnalyzeDiff(ctx context.Context, projectPath string, commitRange string) (*codeanalysis.DiffAnalysisResult, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	a.logger.Info("analyzing git diff",
		"project", absPath,
		"commit_range", commitRange,
	)

	// 1. 获取 diff
	hunks, err := a.getDiffHunks(ctx, absPath, commitRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff hunks: %w", err)
	}

	// 2. 按文件分组
	fileHunks := make(map[string][]diffHunk)
	changedFiles := make([]string, 0)
	fileSet := make(map[string]bool)

	for _, hunk := range hunks {
		if !strings.HasSuffix(hunk.File, ".go") {
			continue
		}
		// 排除测试文件和 vendor
		if strings.HasSuffix(hunk.File, "_test.go") {
			continue
		}
		if strings.Contains(hunk.File, "vendor/") {
			continue
		}
		// 排除生成文件
		if strings.HasSuffix(hunk.File, ".pb.go") || strings.HasSuffix(hunk.File, "_gen.go") {
			continue
		}

		fileHunks[hunk.File] = append(fileHunks[hunk.File], hunk)
		if !fileSet[hunk.File] {
			fileSet[hunk.File] = true
			changedFiles = append(changedFiles, hunk.File)
		}
	}

	// 3. 获取模块路径
	modulePath, err := a.getModulePath(absPath)
	if err != nil {
		a.logger.Warn("failed to get module path", "error", err)
		modulePath = ""
	}

	// 4. 定位变更的函数
	var changedFunctions []codeanalysis.ChangedFunction
	for file, hunks := range fileHunks {
		filePath := filepath.Join(absPath, file)
		funcs, err := a.locateFunctionsInFile(filePath, file, hunks, modulePath)
		if err != nil {
			a.logger.Warn("failed to locate functions in file",
				"file", file,
				"error", err,
			)
			continue
		}
		changedFunctions = append(changedFunctions, funcs...)
	}

	// 去重
	changedFunctions = a.deduplicateFunctions(changedFunctions)

	a.logger.Info("diff analysis completed",
		"changed_files", len(changedFiles),
		"changed_functions", len(changedFunctions),
	)

	return &codeanalysis.DiffAnalysisResult{
		CommitRange:      commitRange,
		ChangedFunctions: changedFunctions,
		ChangedFiles:     changedFiles,
	}, nil
}

// GetCurrentCommit 获取当前 HEAD commit
func (a *DiffAnalyzer) GetCurrentCommit(ctx context.Context, projectPath string) (string, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "git", "-C", projectPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// GetCommitsBetween 获取两个 commit 之间的 commit 数量
func (a *DiffAnalyzer) GetCommitsBetween(ctx context.Context, projectPath string, fromCommit string, toCommit string) (int, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "git", "-C", projectPath, "rev-list", "--count", fromCommit+".."+toCommit)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var count int
	fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count)
	return count, nil
}

// getDiffHunks 获取 diff 块
// commitRange 支持以下值：
//   - "working" 或 ""：分析工作区未提交的改动（git diff HEAD）
//   - "HEAD~1..HEAD"：分析最近一次提交
//   - "main..HEAD"：分析分支对比
func (a *DiffAnalyzer) getDiffHunks(ctx context.Context, projectPath string, commitRange string) ([]diffHunk, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 构建 git diff 命令参数
	args := []string{"-C", projectPath, "diff", "--unified=0"}
	if commitRange == "" || commitRange == "working" {
		// 工作区未提交的改动：对比 HEAD
		args = append(args, "HEAD")
	} else {
		args = append(args, commitRange)
	}

	cmd := exec.CommandContext(cmdCtx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			a.logger.Warn("git diff failed", "stderr", string(exitErr.Stderr))
		}
		return nil, err
	}

	return a.parseDiffOutput(string(output))
}

// parseDiffOutput 解析 diff 输出
func (a *DiffAnalyzer) parseDiffOutput(output string) ([]diffHunk, error) {
	var hunks []diffHunk
	var currentFile string

	// 正则表达式
	fileRegex := regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	hunkRegex := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
	deletedRegex := regexp.MustCompile(`^deleted file mode`)
	newFileRegex := regexp.MustCompile(`^new file mode`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	var isDeleted, isAdded bool

	for scanner.Scan() {
		line := scanner.Text()

		// 检查是否是新文件
		if matches := fileRegex.FindStringSubmatch(line); matches != nil {
			currentFile = matches[2]
			isDeleted = false
			isAdded = false
			continue
		}

		// 检查文件删除
		if deletedRegex.MatchString(line) {
			isDeleted = true
			continue
		}

		// 检查文件新增
		if newFileRegex.MatchString(line) {
			isAdded = true
			continue
		}

		// 解析 hunk 头
		if matches := hunkRegex.FindStringSubmatch(line); matches != nil {
			oldStart, _ := strconv.Atoi(matches[1])
			oldCount := 1
			if matches[2] != "" {
				oldCount, _ = strconv.Atoi(matches[2])
			}
			newStart, _ := strconv.Atoi(matches[3])
			newCount := 1
			if matches[4] != "" {
				newCount, _ = strconv.Atoi(matches[4])
			}

			hunks = append(hunks, diffHunk{
				File:      currentFile,
				OldStart:  oldStart,
				OldCount:  oldCount,
				NewStart:  newStart,
				NewCount:  newCount,
				IsDeleted: isDeleted,
				IsAdded:   isAdded,
			})
		}
	}

	return hunks, nil
}

// locateFunctionsInFile 定位文件中变更的函数
func (a *DiffAnalyzer) locateFunctionsInFile(filePath string, relPath string, hunks []diffHunk, modulePath string) ([]codeanalysis.ChangedFunction, error) {
	// 检查文件是否存在（可能被删除）
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 文件被删除，无法解析 AST
		// 返回一个占位记录
		var result []codeanalysis.ChangedFunction
		for range hunks {
			result = append(result, codeanalysis.ChangedFunction{
				Name:       "unknown",
				File:       relPath,
				ChangeType: codeanalysis.ChangeTypeDeleted,
			})
		}
		return result, nil
	}

	// 解析 AST
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// 提取包路径
	pkgPath := modulePath
	if pkgPath != "" {
		// 从相对路径提取包路径
		dir := filepath.Dir(relPath)
		if dir != "." {
			pkgPath = modulePath + "/" + dir
		}
	}

	// 收集所有函数及其位置
	type funcInfo struct {
		name      string
		fullName  string
		startLine int
		endLine   int
	}
	var functions []funcInfo

	ast.Inspect(node, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			startPos := fset.Position(fn.Pos())
			endPos := fset.Position(fn.End())

			funcName := fn.Name.Name
			fullName := funcName

			// 处理方法接收者
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				recvType := fn.Recv.List[0].Type
				recvName := a.getTypeName(recvType)
				if recvName != "" {
					funcName = recvName + "." + fn.Name.Name
					fullName = funcName
				}
			}

			if pkgPath != "" {
				fullName = pkgPath + "." + funcName
			}

			functions = append(functions, funcInfo{
				name:      funcName,
				fullName:  fullName,
				startLine: startPos.Line,
				endLine:   endPos.Line,
			})
		}
		return true
	})

	// 匹配变更行到函数
	var result []codeanalysis.ChangedFunction
	matched := make(map[string]bool)

	for _, hunk := range hunks {
		for _, fn := range functions {
			// 检查 hunk 是否与函数重叠
			hunkStart := hunk.NewStart
			hunkEnd := hunk.NewStart + hunk.NewCount

			if hunk.IsDeleted || hunk.NewCount == 0 {
				// 删除的行，使用旧行号
				hunkStart = hunk.OldStart
				hunkEnd = hunk.OldStart + hunk.OldCount
			}

			// 检查重叠
			if hunkEnd >= fn.startLine && hunkStart <= fn.endLine {
				if matched[fn.fullName] {
					continue
				}
				matched[fn.fullName] = true

				changeType := codeanalysis.ChangeTypeModified
				if hunk.IsAdded {
					changeType = codeanalysis.ChangeTypeAdded
				} else if hunk.IsDeleted {
					changeType = codeanalysis.ChangeTypeDeleted
				}

				result = append(result, codeanalysis.ChangedFunction{
					Name:         fn.name,
					FullName:     fn.fullName,
					Package:      pkgPath,
					File:         relPath,
					LineStart:    fn.startLine,
					LineEnd:      fn.endLine,
					ChangeType:   changeType,
					LinesAdded:   hunk.NewCount,
					LinesRemoved: hunk.OldCount,
				})
			}
		}
	}

	return result, nil
}

// getTypeName 从 AST 类型表达式获取类型名
func (a *DiffAnalyzer) getTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return a.getTypeName(t.X)
	case *ast.SelectorExpr:
		return a.getTypeName(t.X) + "." + t.Sel.Name
	default:
		return ""
	}
}

// getModulePath 获取模块路径
func (a *DiffAnalyzer) getModulePath(projectPath string) (string, error) {
	goModPath := filepath.Join(projectPath, "go.mod")
	file, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}

	return "", fmt.Errorf("module directive not found")
}

// deduplicateFunctions 去重函数列表
func (a *DiffAnalyzer) deduplicateFunctions(functions []codeanalysis.ChangedFunction) []codeanalysis.ChangedFunction {
	seen := make(map[string]bool)
	var result []codeanalysis.ChangedFunction

	for _, fn := range functions {
		key := fn.FullName
		if key == "" {
			key = fn.File + ":" + fn.Name
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, fn)
		}
	}

	return result
}

// 确保实现接口
var _ codeanalysis.DiffAnalyzer = (*DiffAnalyzer)(nil)
