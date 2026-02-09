package codeanalysis

import (
	"context"
	"fmt"
	"go/token"
	"go/types"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/cocursor/backend/internal/domain/codeanalysis"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/callgraph/vta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// SSAAnalyzer SSA 分析器实现
type SSAAnalyzer struct {
	logger *slog.Logger
}

// NewSSAAnalyzer 创建 SSA 分析器
func NewSSAAnalyzer() *SSAAnalyzer {
	return &SSAAnalyzer{
		logger: log.NewModuleLogger("codeanalysis", "ssa_analyzer"),
	}
}

// Analyze 分析项目，生成调用图
func (a *SSAAnalyzer) Analyze(ctx context.Context, projectPath string, entryPoints []string, algorithm codeanalysis.AlgorithmType) (*codeanalysis.AnalysisResult, error) {
	return a.AnalyzeWithProgress(ctx, projectPath, entryPoints, algorithm, nil)
}

// AnalyzeWithProgress 分析项目，生成调用图（带进度回调）
func (a *SSAAnalyzer) AnalyzeWithProgress(ctx context.Context, projectPath string, entryPoints []string, algorithm codeanalysis.AlgorithmType, onProgress codeanalysis.SSAProgressCallback) (*codeanalysis.AnalysisResult, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 安全的进度回调封装
	report := func(progress int, message string) {
		if onProgress != nil {
			onProgress(progress, message)
		}
	}

	a.logger.Info("starting SSA analysis",
		"project", absPath,
		"algorithm", algorithm,
		"entry_points", len(entryPoints),
	)

	// 1. 加载包（0% - 40%）
	report(0, "Loading Go packages...")
	prog, pkgs, modulePath, err := a.loadPackages(ctx, absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	a.logger.Info("packages loaded",
		"module", modulePath,
		"packages", len(pkgs),
	)
	report(40, fmt.Sprintf("Loaded %d packages, building call graph...", len(pkgs)))

	// 2. 构建调用图（40% - 80%）
	report(45, fmt.Sprintf("Building call graph with %s algorithm...", algorithm))
	cgResult, err := a.buildCallGraph(prog, pkgs, algorithm, entryPoints)
	if err != nil {
		return nil, fmt.Errorf("failed to build call graph: %w", err)
	}
	report(80, "Call graph built, extracting data...")

	// 3. 提取函数节点和调用边（80% - 100%）
	result, err := a.extractCallGraphData(cgResult.graph, modulePath, absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract call graph data: %w", err)
	}

	result.ModulePath = modulePath
	result.ActualAlgorithm = cgResult.actualAlgorithm
	result.Fallback = false
	result.FallbackReason = ""

	report(100, fmt.Sprintf("Analysis complete: %d functions, %d edges", len(result.FuncNodes), len(result.FuncEdges)))

	a.logger.Info("SSA analysis completed",
		"func_count", len(result.FuncNodes),
		"edge_count", len(result.FuncEdges),
		"actual_algorithm", cgResult.actualAlgorithm,
		"fallback", false,
	)

	return result, nil
}

// loadPackages 加载项目的所有包
func (a *SSAAnalyzer) loadPackages(_ context.Context, projectPath string) (*ssa.Program, []*ssa.Package, string, error) {
	// 配置包加载
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedTypesSizes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedDeps |
			packages.NeedModule,
		Dir:   projectPath,
		Tests: false, // 不加载测试
	}

	// 加载所有包
	initial, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to load packages: %w", err)
	}

	// 检查加载错误
	var errs []string
	for _, pkg := range initial {
		for _, err := range pkg.Errors {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		a.logger.Warn("package loading had errors", "errors", errs[:min(5, len(errs))])
	}

	// 检查是否加载到任何包
	if len(initial) == 0 {
		errMsg := "no Go packages found in the project"
		if len(errs) > 0 {
			errMsg = fmt.Sprintf("failed to load Go packages: %s", errs[0])
		}
		return nil, nil, "", fmt.Errorf("%s", errMsg)
	}

	// 检查是否是有效的 Go 模块（至少需要有一个包有有效的 PkgPath）
	hasValidPkg := false
	for _, pkg := range initial {
		if pkg.PkgPath != "" && pkg.PkgPath != "." && !strings.HasPrefix(pkg.PkgPath, "-") {
			hasValidPkg = true
			break
		}
	}
	if !hasValidPkg {
		errMsg := "the directory is not a valid Go module (missing go.mod or invalid configuration)"
		if len(errs) > 0 {
			// 提取更具体的错误信息
			for _, e := range errs {
				if strings.Contains(e, "go.mod") || strings.Contains(e, "main module") {
					errMsg = fmt.Sprintf("Go module error: %s", e)
					break
				}
			}
		}
		return nil, nil, "", fmt.Errorf("%s", errMsg)
	}

	// 获取模块路径
	modulePath := ""
	for _, pkg := range initial {
		if pkg.Module != nil {
			modulePath = pkg.Module.Path
			break
		}
	}

	if modulePath == "" && len(initial) > 0 {
		// 从包路径推断
		modulePath = initial[0].PkgPath
		if idx := strings.Index(modulePath, "/internal/"); idx > 0 {
			modulePath = modulePath[:idx]
		} else if idx := strings.Index(modulePath, "/pkg/"); idx > 0 {
			modulePath = modulePath[:idx]
		} else if idx := strings.Index(modulePath, "/cmd/"); idx > 0 {
			modulePath = modulePath[:idx]
		}
	}

	// 构建 SSA
	prog, ssaPkgs := ssautil.AllPackages(initial, ssa.SanityCheckFunctions)
	prog.Build()

	// 过滤有效的 SSA 包
	var validPkgs []*ssa.Package
	for _, pkg := range ssaPkgs {
		if pkg != nil {
			validPkgs = append(validPkgs, pkg)
		}
	}

	return prog, validPkgs, modulePath, nil
}

// buildCallGraphResult 调用图构建结果
type buildCallGraphResult struct {
	graph           *callgraph.Graph
	actualAlgorithm codeanalysis.AlgorithmType
}

// buildCallGraph 构建调用图
func (a *SSAAnalyzer) buildCallGraph(prog *ssa.Program, pkgs []*ssa.Package, algorithm codeanalysis.AlgorithmType, entryPoints []string) (*buildCallGraphResult, error) {
	switch algorithm {
	case codeanalysis.AlgorithmStatic:
		return &buildCallGraphResult{
			graph:           static.CallGraph(prog),
			actualAlgorithm: codeanalysis.AlgorithmStatic,
		}, nil

	case codeanalysis.AlgorithmCHA:
		return &buildCallGraphResult{
			graph:           cha.CallGraph(prog),
			actualAlgorithm: codeanalysis.AlgorithmCHA,
		}, nil

	case codeanalysis.AlgorithmRTA:
		// RTA 需要 main 函数作为入口
		mains := a.findMainFunctions(pkgs, entryPoints)
		if len(mains) == 0 {
			return nil, &codeanalysis.AlgorithmFailedError{
				Algorithm:  codeanalysis.AlgorithmRTA,
				Reason:     "未找到入口函数",
				Suggestion: "请检查入口函数配置，或选择 CHA / Static 算法重试",
				Details:    "RTA 算法需要 main() 作为入口函数",
			}
		}
		// RTA 算法在处理某些特殊类型时可能 panic（如反射类型、type parameter 等）
		// 使用 recover 保护，panic 时返回错误
		return a.runRTAWithRecover(mains)

	case codeanalysis.AlgorithmVTA:
		return a.runVTAWithRecover(prog)

	default:
		// 默认使用 RTA
		mains := a.findMainFunctions(pkgs, entryPoints)
		if len(mains) == 0 {
			return nil, &codeanalysis.AlgorithmFailedError{
				Algorithm:  codeanalysis.AlgorithmRTA,
				Reason:     "未找到入口函数",
				Suggestion: "请检查入口函数配置，或选择 CHA / Static 算法重试",
				Details:    "RTA 算法需要 main() 作为入口函数",
			}
		}
		// 同样使用 recover 保护
		return a.runRTAWithRecover(mains)
	}
}

// runRTAWithRecover 运行 RTA 分析，带 panic 恢复
// RTA 算法在处理某些类型时可能 panic（已知问题：golang/go#61160）
// 当发生 panic 时返回错误
func (a *SSAAnalyzer) runRTAWithRecover(mains []*ssa.Function) (result *buildCallGraphResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicMsg := fmt.Sprintf("%v", r)
			a.logger.Warn("RTA analysis panicked",
				"panic", panicMsg,
			)
			err = &codeanalysis.AlgorithmFailedError{
				Algorithm:  codeanalysis.AlgorithmRTA,
				Reason:     "分析类型时遇到错误",
				Suggestion: "请选择 CHA / Static 算法重试",
				Details:    fmt.Sprintf("可能由于反射或泛型导致分析失败，技术详情: %s", panicMsg),
			}
		}
	}()

	res := rta.Analyze(mains, true)
	return &buildCallGraphResult{
		graph:           res.CallGraph,
		actualAlgorithm: codeanalysis.AlgorithmRTA,
	}, nil
}

// runVTAWithRecover 运行 VTA 分析，带 panic 恢复
func (a *SSAAnalyzer) runVTAWithRecover(prog *ssa.Program) (result *buildCallGraphResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicMsg := fmt.Sprintf("%v", r)
			a.logger.Warn("VTA analysis panicked",
				"panic", panicMsg,
			)
			err = &codeanalysis.AlgorithmFailedError{
				Algorithm:  codeanalysis.AlgorithmVTA,
				Reason:     "分析类型时遇到错误",
				Suggestion: "请选择 CHA / Static 算法重试",
				Details:    fmt.Sprintf("可能由于反射或泛型导致分析失败，技术详情: %s", panicMsg),
			}
		}
	}()

	return &buildCallGraphResult{
		graph:           vta.CallGraph(ssautil.AllFunctions(prog), cha.CallGraph(prog)),
		actualAlgorithm: codeanalysis.AlgorithmVTA,
	}, nil
}

// findMainFunctions 查找 main 函数
func (a *SSAAnalyzer) findMainFunctions(pkgs []*ssa.Package, entryPoints []string) []*ssa.Function {
	var mains []*ssa.Function

	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}

		// 检查是否是 main 包
		if pkg.Pkg.Name() == "main" {
			if mainFn := pkg.Func("main"); mainFn != nil {
				mains = append(mains, mainFn)
			}
		}

		// 添加 init 函数
		if initFn := pkg.Func("init"); initFn != nil {
			mains = append(mains, initFn)
		}
	}

	return mains
}

// extractCallGraphData 从调用图提取函数节点和调用边
func (a *SSAAnalyzer) extractCallGraphData(cg *callgraph.Graph, modulePath string, projectPath string) (*codeanalysis.AnalysisResult, error) {
	result := &codeanalysis.AnalysisResult{
		ModulePath: modulePath,
		FuncNodes:  make([]*codeanalysis.FuncNode, 0),
		FuncEdges:  make([]*codeanalysis.FuncEdge, 0),
	}

	// 用于去重和映射
	funcMap := make(map[string]*codeanalysis.FuncNode)
	var funcID int64 = 1

	// 遍历调用图
	callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		caller := edge.Caller.Func
		callee := edge.Callee.Func

		// 过滤：只保留项目内的调用
		if !a.shouldIncludeFunc(caller, modulePath) || !a.shouldIncludeFunc(callee, modulePath) {
			return nil
		}

		// 获取或创建调用者节点
		callerNode := a.getOrCreateFuncNode(caller, funcMap, &funcID, projectPath)
		calleeNode := a.getOrCreateFuncNode(callee, funcMap, &funcID, projectPath)

		if callerNode == nil || calleeNode == nil {
			return nil
		}

		// 获取调用位置
		callSiteFile := ""
		callSiteLine := 0
		if edge.Site != nil {
			pos := edge.Site.Pos()
			if pos.IsValid() {
				position := edge.Caller.Func.Prog.Fset.Position(pos)
				callSiteFile = a.getRelativePath(position.Filename, projectPath)
				callSiteLine = position.Line
			}
		}

		// 创建调用边
		result.FuncEdges = append(result.FuncEdges, &codeanalysis.FuncEdge{
			CallerID:     callerNode.ID,
			CalleeID:     calleeNode.ID,
			CallSiteFile: callSiteFile,
			CallSiteLine: callSiteLine,
		})

		return nil
	})

	// 收集所有函数节点
	for _, node := range funcMap {
		result.FuncNodes = append(result.FuncNodes, node)
	}

	return result, nil
}

// shouldIncludeFunc 判断函数是否应该包含在分析中
func (a *SSAAnalyzer) shouldIncludeFunc(fn *ssa.Function, modulePath string) bool {
	if fn == nil || fn.Pkg == nil {
		return false
	}

	pkgPath := fn.Pkg.Pkg.Path()

	// 必须在项目模块内
	if !strings.HasPrefix(pkgPath, modulePath) {
		return false
	}

	// 排除 vendor
	if strings.Contains(pkgPath, "/vendor/") {
		return false
	}

	// 排除测试包
	if strings.HasSuffix(pkgPath, "_test") {
		return false
	}

	return true
}

// getOrCreateFuncNode 获取或创建函数节点
func (a *SSAAnalyzer) getOrCreateFuncNode(fn *ssa.Function, funcMap map[string]*codeanalysis.FuncNode, nextID *int64, projectPath string) *codeanalysis.FuncNode {
	if fn == nil {
		return nil
	}

	fullName := fn.String()

	if node, ok := funcMap[fullName]; ok {
		return node
	}

	// 获取位置信息
	filePath := ""
	lineStart := 0
	lineEnd := 0

	if fn.Pos().IsValid() {
		position := fn.Prog.Fset.Position(fn.Pos())
		filePath = a.getRelativePath(position.Filename, projectPath)
		lineStart = position.Line
	}

	// 获取函数结束位置（如果有语法信息）
	if fn.Syntax() != nil {
		endPos := fn.Syntax().End()
		if endPos.IsValid() {
			endPosition := fn.Prog.Fset.Position(endPos)
			lineEnd = endPosition.Line
		}
	}

	// 确定包路径和函数名
	pkgPath := ""
	funcName := fn.Name()
	if fn.Pkg != nil {
		pkgPath = fn.Pkg.Pkg.Path()
	}

	// 处理方法接收者
	if fn.Signature.Recv() != nil {
		recvType := fn.Signature.Recv().Type()
		if ptr, ok := recvType.(*types.Pointer); ok {
			recvType = ptr.Elem()
		}
		if named, ok := recvType.(*types.Named); ok {
			funcName = named.Obj().Name() + "." + fn.Name()
		}
	}

	node := &codeanalysis.FuncNode{
		ID:            *nextID,
		FullName:      fullName,
		CanonicalName: canonicalizeFuncName(fullName),
		Package:       pkgPath,
		FuncName:      funcName,
		FilePath:      filePath,
		LineStart:     lineStart,
		LineEnd:       lineEnd,
		IsExported:    token.IsExported(fn.Name()),
	}

	funcMap[fullName] = node
	*nextID++

	return node
}

// getRelativePath 获取相对路径
func (a *SSAAnalyzer) getRelativePath(absPath string, basePath string) string {
	rel, err := filepath.Rel(basePath, absPath)
	if err != nil {
		return absPath
	}
	return rel
}

// canonicalizeFuncName 将 SSA 原始函数名转换为规范化名称
// (*github.com/example/pkg.Type).Method → github.com/example/pkg.Type.Method
// 普通函数和匿名闭包保持不变
func canonicalizeFuncName(ssaName string) string {
	if strings.HasPrefix(ssaName, "(*") {
		name := strings.TrimPrefix(ssaName, "(*")
		name = strings.Replace(name, ").", ".", 1)
		return name
	}
	return ssaName
}

// 确保实现接口
var _ codeanalysis.SSAAnalyzer = (*SSAAnalyzer)(nil)
