---
name: go-precision-test
description: Go 精准测试分析。基于静态调用图和 Git diff 分析代码变更影响面，推断需要运行的集成测试，生成精准测试命令。包含影响面调用链可视化和测试推荐。当用户提到「精准测试」「影响面分析」「代码变更影响」「需要跑哪些集成测试」时使用此 Skill。
---

# Skill: go-precision-test

Go 精准测试分析工具。通过静态调用图 + Git diff 分析代码变更影响面，自动推荐需要运行的集成测试。

## 使用场景

当用户询问以下问题时使用此 Skill：
- "精准测试分析" / "precision test analysis"
- "这次改动需要跑哪些集成测试？"
- "分析一下影响面"
- "代码变更影响了什么？"
- "帮我做影响面分析"
- "分析最近提交的代码变更"

## 前置条件

1. 当前项目必须是 Go 项目
2. cocursor 后端服务正在运行（默认 http://localhost:19960）
3. 需要先在 Go Impact Analysis UI 面板中生成调用图

## 完整工作流

### Step 1: 获取项目配置

从用户上下文获取当前项目路径，然后获取项目配置（含集成测试目录和 build tag）。

```bash
curl -s -X POST http://localhost:19960/api/v1/analysis/projects/config \
  -H "Content-Type: application/json" \
  -d '{"project_path": "<项目路径>"}'
```

从返回结果中提取：
- `integration_test_dir`：集成测试目录路径
- `integration_test_tag`：集成测试 build tag
- `entry_points`：入口函数列表

如果 `integration_test_dir` 为空，询问用户集成测试目录的位置（常见如 `test/integration/`、`tests/`）。

### Step 2: 检查调用图状态

```bash
curl -s -X POST http://localhost:19960/api/v1/analysis/callgraph/status \
  -H "Content-Type: application/json" \
  -d '{"project_path": "<项目路径>"}'
```

根据返回结果处理：

| 情况 | 处理 |
|------|------|
| `exists: true` | 继续 Step 3 |
| `exists: false` | **停止**，提示用户："请先在 Go Impact Analysis 面板中生成调用图后再运行精准测试分析。" |

### Step 3: 推断 Diff 范围

根据用户意图推断合适的 diff 分析范围：

| 用户意图 | commit_range |
|---------|-------------|
| "分析当前工作区改动" / 默认 | `"working"` |
| "分析最近一次提交" | `"HEAD~1..HEAD"` |
| "分析 feature 分支" | `"main..HEAD"` |
| "分析最近 N 次提交" | `"HEAD~N..HEAD"` |
| 用户指定了具体范围 | 使用用户指定的值 |

如果用户没有明确指定，默认使用 `"working"` 分析工作区未提交的改动。

### Step 4: 分析 Git Diff

```bash
curl -s -X POST http://localhost:19960/api/v1/analysis/diff \
  -H "Content-Type: application/json" \
  -d '{
    "project_path": "<项目路径>",
    "commit_range": "<Step 3 推断的范围>"
  }'
```

如果返回的 `changed_functions` 为空，向用户报告"未检测到代码变更"并结束。

### Step 5: AI 过滤非逻辑改动

拿到变更函数列表后，读取对应的 diff 内容（使用 `git diff` 命令），判断每个函数的变更是否为**有效的逻辑改动**。

过滤规则（排除以下函数）：
- 只修改了注释（`//` 或 `/* */`）
- 只修改了日志文本（`log.Info`、`log.Warn` 等的字符串参数）
- 只修改了空行
- 只修改了 import 语句
- 只修改了变量名/格式化（无逻辑变化）

**保留**的有效改动：
- 修改了条件判断、循环、函数调用、返回值
- 修改了业务逻辑
- 新增/删除了代码块

向用户报告过滤结果：
```
过滤结果：Step 4 检测到 N 个变更函数，其中 M 个为有效逻辑改动（过滤了 K 个仅含注释/日志/格式变更的函数）
```

### Step 6: 查询影响面

使用过滤后的有效变更函数查询影响面：

```bash
curl -s -X POST http://localhost:19960/api/v1/analysis/impact \
  -H "Content-Type: application/json" \
  -d '{
    "project_path": "<项目路径>",
    "functions": ["<有效变更函数的 full_name>"],
    "depth": 5
  }'
```

### Step 7: 影响面调用链可视化

整合 Step 4-6 的结果，输出可视化的影响面报告。

#### 报告格式

```markdown
## 📊 影响面分析报告

**分析范围**: <commit_range>
**变更函数**: <N> 个（有效改动 <M> 个）

### 变更函数列表

| 函数 | 文件 | 变更行 |
|------|------|--------|
| `FuncA` | path/to/file.go | +12/-5 |
| `FuncB` | path/to/file.go | +3/-1 |

### 调用链

FuncA (path/to/file.go)
  ↑ ServiceA.Method (path/to/service.go:42)
    ↑ HandlerX (path/to/handler.go:67)
      ↑ main (cmd/server/main.go:15)

FuncB (path/to/file.go)
  ↑ ServiceB.Method (path/to/service.go:88)
    ↑ HandlerY (path/to/handler.go:120)

### 受影响的入口点

- HandlerX (path/to/handler.go)
- HandlerY (path/to/handler.go)
```

### Step 8: 读取集成测试文件

读取集成测试目录下的所有测试文件：

```bash
# 列出集成测试文件
find <项目路径>/<integration_test_dir> -name "*_test.go" -type f
```

对每个测试文件，读取其内容，重点关注：
- `func Test...` 函数名
- HTTP 请求的 URL 路径（如 `/api/v1/...`）
- 请求方法（GET/POST/PUT/DELETE）

### Step 9: AI 推断测试映射

基于以下信息，推断哪些集成测试需要运行：

**输入**：
1. 受影响的函数链（特别是 Handler 层函数及其 HTTP 路由）
2. 集成测试文件的内容（测试函数名、调用的 API 路径）

**推断逻辑**：
1. 从影响面的调用链终点识别受影响的 HTTP Handler
2. 从 Handler 推断对应的 API 路由路径
3. 匹配集成测试中调用了这些 API 路径的测试函数
4. 如果无法精确匹配路由，通过测试函数名和 Handler 的语义相似性推断

**输出精准测试报告**：

```markdown
## 🎯 精准测试报告

### 推荐运行的集成测试

| 测试函数 | 测试文件 | 关联原因 |
|---------|---------|---------|
| `TestCreateOrder` | test/integration/order_test.go | 调用 POST /api/v1/orders，关联 HandlerX |
| `TestBatchOrder` | test/integration/order_test.go | 调用 POST /api/v1/orders/batch，关联 HandlerY |

### 运行命令

​```bash
# 运行推荐的集成测试
go test -tags=<integration_test_tag> -run "TestCreateOrder|TestBatchOrder" -v ./<integration_test_dir>/...
​```

### 分析置信度

- **高置信度** (3 个): TestCreateOrder, TestBatchOrder, TestOrderUpdate
  - 测试直接调用了受影响的 API 路由
- **中置信度** (1 个): TestOrderFlow
  - 测试可能间接涉及受影响的功能

### 未覆盖的影响面

以下受影响的 Handler 没有找到对应的集成测试：
- HandleDeleteOrder (path/to/handler.go:200) - 建议补充集成测试
```

## 错误处理

| 错误 | 处理方式 |
|------|---------|
| 后端服务未运行 | 提示用户启动 cocursor 后端服务 |
| 项目非 Go 项目 | 提示暂不支持，此 Skill 仅适用于 Go 项目 |
| 调用图不存在 | 提示用户先在 Go Impact Analysis 面板中生成调用图 |
| 集成测试目录未配置 | 询问用户测试目录位置 |
| Git diff 无变更 | 提示未检测到代码变更 |
| Git 仓库不存在 | 提示用户在 Git 仓库中运行 |

## API 端点汇总

| 功能 | 端点 | 方法 |
|------|------|------|
| 获取项目配置 | `/api/v1/analysis/projects/config` | POST |
| 检查调用图状态 | `/api/v1/analysis/callgraph/status` | POST |
| 分析 Git Diff | `/api/v1/analysis/diff` | POST |
| 查询影响面 | `/api/v1/analysis/impact` | POST |

## 注意事项

1. **静态分析局限**: 反射调用、动态插件加载无法被静态分析追踪
2. **AI 推断准确性**: 测试映射由 AI 推断，可能有遗漏或误判，建议人工审核
3. **调用图更新**: 当代码结构变化较大时，建议重新生成调用图
4. **工作区 Diff**: 使用 `"working"` 时分析的是未提交的改动，请确保改动已保存到文件
