# 项目发现机制

## 概述

项目发现机制用于自动识别和分组 Cursor 工作区，使同一项目的多个工作区能够统一管理和查询。

## 核心组件

### 1. ProjectDiscovery（项目发现器）

位置：`backend/internal/application/cursor/project_discovery.go`

功能：
- 扫描所有 Cursor 工作区存储目录
- 读取每个工作区的 `workspace.json` 文件
- 提取项目路径和 Git 信息
- 生成 `DiscoveredWorkspace` 列表

### 2. ProjectManager（项目管理器）

位置：`backend/internal/application/cursor/project_manager.go`

功能：
- 接收 `DiscoveredWorkspace` 列表
- 按"同一项目"规则分组
- 维护内存缓存（projects map, pathMap）
- 提供项目查询接口

### 3. 同一项目判断规则

优先级从高到低：

**P0: Git 远程 URL 相同**
- 如果两个工作区的 Git 远程 URL 相同，则认为是同一项目
- 支持 URL 规范化（统一协议、大小写、移除 .git 后缀）

**P1: 物理路径完全相同**
- 解析符号链接后，如果路径完全相同，则认为是同一项目
- 路径规范化：统一分隔符、大小写（Windows）

**P2: 项目名相同 + 路径相似度 > 90%**
- 如果项目名（路径最后一个目录名）相同
- 且路径相似度（最长公共子序列算法）> 90%
- 则认为是同一项目

### 4. 路径相似度计算

位置：`backend/internal/infrastructure/cursor/path_matcher.go`

算法：最长公共子序列（LCS）

公式：`相似度 = LCS长度 / max(路径1长度, 路径2长度)`

示例：
- `/path/to/project` 和 `/path/to/project-backup` → 相似度 ≈ 0.92
- `/path/to/project1` 和 `/path/to/project2` → 相似度 ≈ 0.85

## API 接口

### 项目列表

```
GET /api/v1/project/list
```

返回所有项目列表，每个项目包含：
- 项目名
- 工作区列表
- Git 信息（如果有）
- 活跃状态

### 项目查询

```
GET /api/v1/project/{project_name}/prompts
GET /api/v1/project/{project_name}/generations
GET /api/v1/project/{project_name}/sessions
GET /api/v1/project/{project_name}/stats/acceptance
```

支持通过项目名查询数据，自动合并多个工作区的数据。

### 项目激活

```
POST /api/v1/project/activate
```

前端上报当前项目，更新活跃状态。

## 数据合并策略

### 统计数据（接受率）

位置：`backend/internal/application/cursor/data_merger.go`

- **合并方式**：累加所有工作区的数据
- **计算方式**：重新计算接受率（接受行数 / 建议行数）
- **返回格式**：合并后的汇总数据 + 原始每日数据数组

### 原始数据（Prompts/Generations/Sessions）

- **合并方式**：按时间排序，保留 source（工作区 ID）
- **排序规则**：时间戳降序（最新的在前）
- **返回格式**：带来源的数据数组

## 前端集成

### 工作区检测

位置：`co-extension/src/utils/workspaceDetector.ts`

- 监听工作区变化事件
- 自动检测当前工作区路径

### 项目上报

位置：`co-extension/src/utils/projectReporter.ts`

- 检测当前项目
- 调用后端 API 上报
- 更新活跃状态

### 项目列表视图

位置：`co-extension/src/sidebar/projectProvider.ts`

- 显示所有项目列表
- 标记活跃项目
- 显示工作区列表
- 支持点击查看详情

## 使用示例

### 后端启动

```go
projectManager := cursor.NewProjectManager()
err := projectManager.Start()
if err != nil {
    log.Fatal(err)
}
```

### 查询项目

```go
project := projectManager.GetProject("cocursor")
if project != nil {
    fmt.Printf("项目: %s, 工作区数: %d\n", project.ProjectName, len(project.Workspaces))
}
```

### 根据路径查找

```go
projectName, workspace := projectManager.FindByPath("/path/to/project")
if workspace != nil {
    fmt.Printf("项目: %s, 工作区 ID: %s\n", projectName, workspace.WorkspaceID)
}
```

## 注意事项

1. **性能考虑**：启动时扫描所有工作区，建议在后台异步执行
2. **内存占用**：所有项目信息缓存在内存中，大量工作区可能占用较多内存
3. **实时性**：项目列表在启动时生成，后续变化需要手动刷新或重新启动
4. **跨平台**：路径处理已考虑 Windows/Unix 差异，但符号链接解析可能因平台而异

## 测试

单元测试文件：
- `backend/internal/infrastructure/cursor/git_reader_test.go`
- `backend/internal/infrastructure/cursor/path_matcher_test.go`
- `backend/internal/application/cursor/project_discovery_test.go`
- `backend/internal/application/cursor/project_manager_test.go`

运行测试：
```bash
cd backend
go test ./internal/infrastructure/cursor/...
go test ./internal/application/cursor/...
```
