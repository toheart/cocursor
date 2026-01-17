# Cursor 数据库详细分析报告

## 数据库概览
分析版本：2.3.41

### 1. 全局存储数据库
**路径**: `C:\Users\TANG\AppData\Roaming\Cursor\User\globalStorage\state.vscdb`

#### 数据库结构
- **表结构**: 
  - `ItemTable`: `CREATE TABLE ItemTable (key TEXT UNIQUE ON CONFLICT REPLACE, value BLOB)`
  - `cursorDiskKV`: `CREATE TABLE cursorDiskKV (key TEXT UNIQUE ON CONFLICT REPLACE, value BLOB)`

#### 数据统计
- **ItemTable**: 184 条记录
- **cursorDiskKV**: 25,213 条记录（主要是 `agentKv:blob:` 格式的键）
- **总数据量**: 约 199KB（ItemTable）

#### 主要数据分类（ItemTable）

**1. 系统标记**
- `__$__isNewStorageMarker`
- `__$__targetStorageMarker`

**2. AI 相关**
- `aiCodeTracking.*` - AI 代码追踪统计（按日期）
- `aiCodeTrackingStartTime` - AI 追踪开始时间
- `aiCodeTracking.recentCommit` - 最近提交记录
- `aicontext.personalContext` - 个人上下文

**3. Cursor 特定功能**
- `anysphere.cursor-*` - Cursor 各种功能配置
- `cursor/*` - Cursor 编辑器布局、功能开关等
- `cursorAuth/*` - 认证信息（token、订阅状态等）
- `cursorai/*` - Cursor AI 配置（隐私模式、服务器配置等）

**4. 工作区管理**
- `backgroundComposer.*` - 后台编辑器数据
- `composer.*` - Composer 相关配置
- `chat.workspaceTransfer` - 工作区传输

**5. 编辑器状态**
- `colorThemeData` - 颜色主题
- `iconThemeData` - 图标主题
- `productIconThemeData` - 产品图标主题
- `editorFontInfo` - 编辑器字体信息
- `commandPalette.mru.*` - 命令面板最近使用

**6. 扩展和功能**
- `mcpService.knownServerIds` - MCP 服务已知服务器
- `extensions.*` - 扩展相关
- `vscode.*` - VS Code 功能状态

**7. 工作台状态**
- `workbench.*` - 工作台各种视图和面板状态
- `terminal.*` - 终端历史记录和配置
- `history.recentlyOpenedPathsList` - 最近打开的文件路径

**8. 遥测和统计**
- `telemetry.*` - 遥测数据（首次/最后会话日期）
- `perf/lastRunningCommit` - 性能相关

#### cursorDiskKV 表分析
- 主要存储格式: `agentKv:blob:{hash}` 
- 共 25,213 条记录
- 这些可能是 Cursor Agent 的缓存数据，使用哈希值作为键

#### 数据示例（全局存储）

**示例 1: 用户认证信息**
```json
{
  "key": "cursorAuth/cachedEmail",
  "value": "523808719@qq.com"
}
```

**示例 2: Cursor 功能配置**
```json
{
  "key": "cursor/memoriesEnabled",
  "value": "true"
}
```

**示例 3: 全局语言检测记录**
```json
{
  "key": "workbench.editor.languageDetectionOpenedLanguages.global",
  "value": "[[\"html\",true],[\"plaintext\",true],[\"css\",true],[\"jsonc\",true],[\"typescriptreact\",true],[\"typescript\",true],[\"json\",true],[\"makefile\",true],[\"go.mod\",true],[\"ignore\",true],[\"go\",true],[\"markdown\",true]]"
}
```

**示例 4: AI 代码追踪统计（按日期）**
```json
{
  "key": "aiCodeTracking.dailyStats.v1.5.2026-01-17",
  "value": "{\"date\":\"2026-01-17\",\"tabSuggestedLines\":130,\"tabAcceptedLines\":52,\"composerSuggestedLines\":0,\"composerAcceptedLines\":21463}"
}
```
说明：
- `tabSuggestedLines`: Tab 建议的代码行数 (130 行)
- `tabAcceptedLines`: Tab 接受的代码行数 (52 行，接受率 40%)
- `composerSuggestedLines`: Composer 建议的代码行数 (0 行)
- `composerAcceptedLines`: Composer 接受的代码行数 (21,463 行)

**示例 5: 终端命令历史（部分）**
```json
{
  "key": "terminal.history.entries.commands",
  "value": "{\"entries\":[{\"key\":\"touch d:\\\\code\\\\cocursor\\\\co-extension\\\\.eslintrc.json\",\"value\":{}},{\"key\":\"cd d:\\\\code\\\\cocursor\\\\co-extension && npm install\",\"value\":{}},{\"key\":\"make compile-debug\",\"value\":{\"shellType\":\"gitbash\"}},{\"key\":\"openskills sync\",\"value\":{\"shellType\":\"gitbash\"}},...]}"
}
```
说明：存储了所有工作区共享的终端命令历史，包含命令文本和 shell 类型信息。

---

### 2. 工作区存储数据库（数据最多）
**路径**: `C:\Users\TANG\AppData\Roaming\Cursor\User\workspaceStorage\d4b798d47e9a14d74eb7965f996e8739\state.vscdb`

#### 数据库结构
- **表结构**: 与全局存储相同
  - `ItemTable`: `CREATE TABLE ItemTable (key TEXT UNIQUE ON CONFLICT REPLACE, value BLOB)`
  - `cursorDiskKV`: `CREATE TABLE cursorDiskKV (key TEXT UNIQUE ON CONFLICT REPLACE, value BLOB)`

#### 数据统计
- **ItemTable**: 153 条记录
- **cursorDiskKV**: 0 条记录
- **总数据量**: 约 199KB
- **平均每条记录大小**: 约 1,301 字节

#### 主要数据分类

**1. 系统标记**
- `__$__isNewStorageMarker`
- `__$__targetStorageMarker`

**2. AI 服务数据（最大数据）**
- `aiService.prompts` - **80,076 字节** - 存储所有 AI 提示历史
  - 格式: JSON 数组，包含 `text` 和 `commandType` 字段
  - 内容: 用户与 AI 的所有对话提示记录
- `aiService.generations` - **15,298 字节** - AI 生成记录（**详细分析见下方"深度分析"章节**）
  - 包含 `unixMs`（时间戳）、`generationUUID`、`type` 和 `textDescription` 字段
  - **关键**: `textDescription` 字段包含 AI 的实际回复内容和操作描述

**3. Composer 数据**
- `composer.composerData` - **15,241 字节** - Composer 会话数据（**详细分析见下方"深度分析"章节**）
  - 包含完整的 Composer 会话树结构
  - **关键**: 包含对话层级、文件引用、代码变更统计、上下文使用率等

**4. 编辑器状态（大文件）**
- `memento/workbench.editors.files.textFileEditor` - **46,447 字节** - 文本编辑器状态
- `memento/workbench.parts.editor` - **6,818 字节** - 编辑器部件状态
- `history.entries` - **5,627 字节** - 历史记录条目

**5. Cursor 特定功能**
- `cursor/editorLayout.*` - 编辑器布局配置（侧边栏、面板等可见性和宽度）
- `cursor/needsComposerInitialOpening` - Composer 初始化标记
- `cursor/workspaceEligibleForSnippetLearning` - 代码片段学习标记
- `cursorAuth/workspaceOpenedDate` - 工作区打开日期

**6. 调试配置**
- `debug.*` - 调试器配置（断点、选中的配置等）

**7. 终端**
- `terminal.*` - 终端配置、布局、环境变量等

**8. 源代码管理**
- `scm.*` - Git 相关状态和视图
- `vscode.git` - Git 扩展状态

**9. 工作台视图状态**
- `workbench.panel.aichat.*` - AI 聊天面板状态（多个会话 ID）
- `workbench.panel.composerChatViewPane.*` - Composer 聊天视图面板
- `workbench.explorer.*` - 资源管理器状态
- `workbench.view.*` - 各种视图状态（调试、扩展、搜索等）

**10. 其他功能**
- `codelens/cache2` - 代码透镜缓存
- `anysphere.cursor-retrieval` - Cursor 检索数据
- `interactive.sessions` - 交互式会话

#### 数据示例（工作区存储）

**示例 1: AI 对话提示历史（部分）**
```json
{
  "key": "aiService.prompts",
  "value": "[{\"text\":\"安装make命令\",\"commandType\":4},{\"text\":\"查询git bash的windows路径\",\"commandType\":4},{\"text\":\"创建项目：我需要写一个cursor的个人办公的效率系统，使用vscode的插件进行实现。使用React+Go进行架构。\",\"commandType\":4},{\"text\":\"分析错误：\\nActivating extension 'cocursor.cocursor-efficiency' failed: Cannot find module 'd:\\\\code\\\\cocursor\\\\dist\\\\extension.js'\",\"commandType\":4},...]"
}
```
说明：
- 存储该工作区所有与 AI 的对话提示
- `commandType: 4` 表示这是某种特定类型的命令
- 每个条目包含用户输入的文本内容

**示例 2: 编辑器布局配置**
```json
{
  "key": "cursor/editorLayout.sidebarWidth",
  "value": "424"
}
```
说明：工作区特定的侧边栏宽度配置（424 像素）。

**示例 3: 工作区打开日期**
```json
{
  "key": "cursorAuth/workspaceOpenedDate",
  "value": "2026-01-17T04:13:05.331Z"
}
```
说明：记录该工作区首次打开或最近打开的时间。

**示例 4: 文件编辑历史（部分）**
```json
{
  "key": "history.entries",
  "value": "[{\"editor\":{\"resource\":\"file:///d%3A/code/cocursor/backend/internal/infrastructure/singleton/lock_test.go\",\"forceFile\":true,\"options\":{\"override\":\"default\"}}},{\"editor\":{\"resource\":\"file:///d%3A/code/cocursor/cursor_database_analysis.md\",\"forceFile\":true,\"options\":{\"override\":\"default\"}}},...]"
}
```
说明：
- 存储该工作区最近打开的文件列表
- 每个条目包含文件 URI（URL 编码格式）
- `forceFile: true` 表示强制作为文件打开

**示例 5: 工作区语言检测记录**
```json
{
  "key": "workbench.editor.languageDetectionOpenedLanguages.workspace",
  "value": "[[\"go\",true],[\"markdown\",true],[\"json\",true],[\"typescript\",true],...]"
}
```
说明：该工作区特定打开的语言列表，与全局存储的 `.global` 版本不同。

---

## 工作区存储与全局存储的关系

### 数据作用域设计

Cursor 采用了**双层存储架构**，将数据按照作用域分为全局和工作区两个层级：

#### 1. 全局存储（Global Storage）
**作用域**: 跨所有工作区共享，适用于用户级别的配置和状态

**存储内容**:
- **用户认证信息**: `cursorAuth/accessToken`, `cursorAuth/refreshToken`, `cursorAuth/cachedEmail` 等
- **全局功能配置**: `cursor/memoriesEnabled`, `cursor/agentLayout.*`, `cursor/composerAutocompleteHeuristicsEnabled` 等
- **跨工作区统计**: `aiCodeTracking.*` - AI 代码追踪的全局统计
- **全局主题和外观**: `colorThemeData`, `iconThemeData`, `productIconThemeData`, `editorFontInfo`
- **全局终端历史**: `terminal.history.entries.*` - 所有工作区共享的终端命令历史
- **全局编辑器记忆**: `memento/customEditors`, `memento/notebookEditors` 等
- **全局工作台配置**: `workbench.editor.languageDetectionOpenedLanguages.global`（注意 `.global` 后缀）
- **最近打开的文件**: `history.recentlyOpenedPathsList` - 跨工作区的文件历史
- **扩展和 MCP 服务**: `extensions.*`, `mcpService.knownServerIds`
- **Agent 缓存**: `cursorDiskKV` 表中的 25,213 条 `agentKv:blob:*` 记录（全局 Agent 缓存）

#### 2. 工作区存储（Workspace Storage）
**作用域**: 仅限当前工作区，每个工作区有独立的数据库文件

**存储内容**:
- **工作区特定的 AI 对话**: `aiService.prompts`, `aiService.generations` - 该工作区的所有 AI 对话历史
- **Composer 会话数据**: `composer.composerData` - 该工作区的 Composer 会话
- **工作区编辑器状态**: `memento/workbench.editors.files.textFileEditor` - 该工作区打开的文件和编辑器状态
- **工作区布局配置**: `cursor/editorLayout.*` - 该工作区特定的编辑器布局（侧边栏宽度、面板高度等）
- **工作区工作台状态**: `workbench.editor.languageDetectionOpenedLanguages.workspace`（注意 `.workspace` 后缀）
- **工作区文件历史**: `history.entries` - 该工作区内的文件编辑历史
- **工作区终端配置**: `terminal.integrated.layoutInfo`, `terminal.integrated.environmentVariableCollectionsV2` - 工作区特定的终端配置
- **工作区 Git 状态**: `scm.*`, `vscode.git` - 该工作区的 Git 仓库状态
- **工作区调试配置**: `debug.*` - 该工作区的调试器配置
- **工作区视图状态**: `workbench.panel.aichat.*`, `workbench.view.*` - 该工作区各种面板和视图的状态

#### 3. 共同键（Shared Keys）
两个数据库都包含的键（但值可能不同）:
- `__$__isNewStorageMarker` - 存储系统标记
- `__$__targetStorageMarker` - 存储目标标记
- `interactive.sessions` - 交互式会话（可能在不同作用域有不同数据）
- `vscode.git` - Git 扩展状态（全局存储可能是通用配置，工作区存储是特定仓库状态）

### 数据隔离机制

#### 命名约定
Cursor 使用**键名后缀**来区分作用域：

| 键名模式 | 存储位置 | 说明 |
|---------|---------|------|
| `*.global` | 全局存储 | 明确标记为全局作用域 |
| `*.workspace` | 工作区存储 | 明确标记为工作区作用域 |
| `cursorAuth/*` | 全局存储 | 认证信息默认全局 |
| `cursor/editorLayout.*` | 工作区存储 | 编辑器布局默认工作区特定 |
| `aiService.*` | 工作区存储 | AI 服务数据默认工作区特定 |
| `memento/workbench.*` | 工作区存储 | 编辑器记忆默认工作区特定 |

#### 数据同步与继承

1. **全局配置继承**: 工作区会继承全局配置作为默认值，但可以覆盖
   - 例如：全局主题设置会影响所有工作区，但工作区可以有自己的布局配置

2. **独立数据存储**: 工作区特定的数据完全独立，不会影响其他工作区
   - 例如：每个工作区的 `aiService.prompts` 是独立的，不会互相干扰

3. **统计聚合**: 全局存储包含跨工作区的统计信息
   - 例如：`aiCodeTracking.*` 汇总所有工作区的 AI 使用情况

### 实际应用场景

#### 场景 1: 多项目开发
- **全局存储**: 保存用户的认证信息、主题偏好、全局终端历史
- **工作区存储**: 每个项目有独立的 AI 对话历史、编辑器状态、Git 配置

#### 场景 2: 团队协作
- **全局存储**: 个人设置和偏好（不影响团队）
- **工作区存储**: 项目特定的配置可以分享（通过 `.vscode/settings.json` 等）

#### 场景 3: 数据分析
- **全局存储**: 分析用户的整体使用模式、跨项目统计
- **工作区存储**: 分析单个项目的开发活动、AI 使用情况

### 数据访问策略

当需要访问 Cursor 数据时：

1. **用户级数据** → 访问全局存储
   - 认证信息、全局配置、跨项目统计

2. **项目级数据** → 访问对应工作区的存储
   - AI 对话历史、编辑器状态、项目特定配置

3. **综合分析** → 同时访问两个存储
   - 结合全局统计和工作区细节进行完整分析

### 具体对比示例

#### 示例 1: Cursor 配置
| 配置项 | 全局存储 | 工作区存储 |
|--------|---------|-----------|
| Agent 布局 | `cursor/agentLayout.sidebarLocation` | 无 |
| 编辑器布局 | 无 | `cursor/editorLayout.sidebarWidth` (值: `"424"`) |
| 功能开关 | `cursor/memoriesEnabled` (值: `"true"`) | `cursor/workspaceEligibleForSnippetLearning` |

#### 示例 2: 工作台状态
| 状态项 | 全局存储 | 工作区存储 |
|--------|---------|-----------|
| 语言检测 | `workbench.editor.languageDetectionOpenedLanguages.global`<br/>值: `[["html",true],["go",true],...]` | `workbench.editor.languageDetectionOpenedLanguages.workspace`<br/>值: `[["go",true],["markdown",true],...]` |
| 活动栏位置 | `workbench.activityBar.location` | `workbench.activityBar.hidden` |
| 面板状态 | `workbench.panel.alignment` | `workbench.panel.aichat.{id}.numberOfVisibleViews` |

#### 示例 3: AI 相关数据
| 数据类型 | 全局存储 | 工作区存储 |
|--------|---------|-----------|
| 代码追踪统计 | `aiCodeTracking.dailyStats.v1.5.2026-01-17`<br/>值: `{"date":"2026-01-17","tabSuggestedLines":130,"tabAcceptedLines":52,...}` | 无 |
| AI 对话历史 | 无 | `aiService.prompts`<br/>值: `[{"text":"安装make命令","commandType":4},...]` (80KB) |
| AI 生成记录 | 无 | `aiService.generations`<br/>值: `[{"unixMs":1768643571672,"generationUUID":"491769d..."},...]` (15KB) |
| Composer 数据 | `backgroundComposer.*` | `composer.composerData` (15KB) |
| Agent 缓存 | `cursorDiskKV` (25,213 条 `agentKv:blob:*`) | 无 |

#### 示例 4: 编辑器状态
| 状态类型 | 全局存储 | 工作区存储 |
|--------|---------|-----------|
| 编辑器记忆 | `memento/customEditors` | `memento/workbench.editors.files.textFileEditor` (46KB) |
| 文件历史 | `history.recentlyOpenedPathsList` | `history.entries`<br/>值: `[{"editor":{"resource":"file:///d%3A/code/cocursor/..."}},...]` |
| 编辑器部件 | 无 | `memento/workbench.parts.editor` (6.8KB) |

#### 示例 5: 终端
| 终端数据 | 全局存储 | 工作区存储 |
|--------|---------|-----------|
| 命令历史 | `terminal.history.entries.commands`<br/>值: `{"entries":[{"key":"cd d:\\code\\cocursor...","value":{}},...]}` | 无 |
| 目录历史 | `terminal.history.entries.dirs` | 无 |
| 布局配置 | 无 | `terminal.integrated.layoutInfo` |
| 环境变量 | 无 | `terminal.integrated.environmentVariableCollectionsV2` |

---

## 数据使用建议

### 1. AI 使用情况分析
从 `aiService.prompts` 和 `aiService.generations` 可以提取：
- 对话总数和消息总数
- 对话时间分布
- 常用命令类型
- 对话主题分析

### 2. 代码建议接受率
从全局存储的 `aiCodeTracking.*` 可以提取：
- Tab 建议的代码行数
- Tab 接受的代码行数
- 接受率趋势
- 按日期统计

### 3. 工作区使用模式
从工作区存储可以分析：
- 编辑器布局偏好
- 常用功能（调试、终端、Git）
- 会话活跃度
- 文件编辑历史

### 4. Composer 会话分析
从 `composer.composerData` 可以提取：
- 会话上下文（附加的文件、代码片段）
- 对话模式（Chat vs Agent）
- 会话时长和效率

---

## 数据库访问注意事项

1. **文件锁定**: Cursor 运行时可能锁定数据库文件，需要确保 Cursor 关闭或使用只读模式访问
2. **数据格式**: 大部分 value 字段是 BLOB，需要根据 key 的类型进行相应解析（通常是 JSON）
3. **数据更新**: 这些数据库会频繁更新，分析时建议先备份
4. **隐私**: 包含用户对话历史、文件路径等敏感信息，注意数据安全

---

## 其他工作区统计

| 工作区 ID | ItemTable 记录数 | cursorDiskKV 记录数 |
|----------|----------------|-------------------|
| d4b798d47e9a14d74eb7965f996e8739 | 153 | 0 |
| 861ca156f6e5c2aad73afd2854c92261 | 87 | 0 |
| bf0871dff1e9e2d800e8ff4f97c9444b | 77 | 0 |
| ca45e3de82bddb73a4fc1291a910f3e9 | 72 | 0 |
| 8c9de0af69d4b33fe3e10c7cc030549f | 68 | 0 |
| ext-dev | 68 | 0 |
| c9bf9293c2c47eef24da0f69e3884363 | 54 | 0 |
| f448e6e71945186fe5a505fbaf753753 | 51 | 0 |

---

## 技术细节

### 数据库特性
- SQLite 数据库
- 使用 `UNIQUE ON CONFLICT REPLACE` 确保键唯一性
- value 字段使用 BLOB 类型存储，支持任意二进制数据
- 无显式索引，依赖 SQLite 的自动索引机制

### 数据访问示例

#### 基础查询

```sql
-- 查看所有键
SELECT key FROM ItemTable ORDER BY key;

-- 查看数据大小
SELECT key, length(value) as size FROM ItemTable ORDER BY size DESC;

-- 查看特定键的数据（需要根据类型解析）
SELECT key, value FROM ItemTable WHERE key = 'aiService.prompts';

-- 统计分类
SELECT 
  CASE 
    WHEN key LIKE 'workbench.%' THEN 'workbench'
    WHEN key LIKE 'cursor/%' THEN 'cursor'
    WHEN key LIKE 'aiService.%' THEN 'aiService'
    ELSE 'other'
  END as category,
  COUNT(*) as count
FROM ItemTable
GROUP BY category;
```

#### 实际数据查询示例

**1. 查询 AI 代码追踪统计**
```sql
-- 获取最新的 AI 代码追踪统计
SELECT key, value 
FROM ItemTable 
WHERE key LIKE 'aiCodeTracking.dailyStats.%' 
ORDER BY key DESC 
LIMIT 1;
```
结果示例：
```
key: aiCodeTracking.dailyStats.v1.5.2026-01-17
value: {"date":"2026-01-17","tabSuggestedLines":130,"tabAcceptedLines":52,"composerSuggestedLines":0,"composerAcceptedLines":21463}
```

**2. 查询工作区 AI 对话历史（前 5 条）**
```sql
-- 注意：value 是 JSON 字符串，需要解析
SELECT key, substr(value, 1, 500) as preview
FROM ItemTable 
WHERE key = 'aiService.prompts';
```
结果示例（JSON 格式）：
```json
[
  {"text":"安装make命令","commandType":4},
  {"text":"查询git bash的windows路径","commandType":4},
  {"text":"创建项目：我需要写一个cursor的个人办公的效率系统...","commandType":4},
  ...
]
```

**3. 查询编辑器布局配置**
```sql
-- 获取工作区编辑器布局
SELECT key, value 
FROM ItemTable 
WHERE key LIKE 'cursor/editorLayout.%';
```
结果示例：
```
key: cursor/editorLayout.sidebarWidth
value: 424

key: cursor/editorLayout.panelHeight
value: 300
```

**4. 对比全局和工作区的语言检测记录**
```sql
-- 全局存储
SELECT key, value 
FROM ItemTable 
WHERE key = 'workbench.editor.languageDetectionOpenedLanguages.global';

-- 工作区存储
SELECT key, value 
FROM ItemTable 
WHERE key = 'workbench.editor.languageDetectionOpenedLanguages.workspace';
```

**5. 查询终端命令历史（最近 10 条）**
```sql
-- 注意：value 是 JSON，包含 entries 数组
SELECT key, value 
FROM ItemTable 
WHERE key = 'terminal.history.entries.commands';
```
结果示例（JSON 结构）：
```json
{
  "entries": [
    {"key": "cd d:\\code\\cocursor\\co-extension", "value": {}},
    {"key": "npm install", "value": {"shellType": "gitbash"}},
    {"key": "make compile-debug", "value": {"shellType": "gitbash"}},
    ...
  ]
}
```

**6. 查询文件编辑历史**
```sql
SELECT key, substr(value, 1, 500) as preview
FROM ItemTable 
WHERE key = 'history.entries';
```
结果示例（JSON 格式）：
```json
[
  {
    "editor": {
      "resource": "file:///d%3A/code/cocursor/backend/internal/infrastructure/singleton/lock_test.go",
      "forceFile": true,
      "options": {"override": "default"}
    }
  },
  {
    "editor": {
      "resource": "file:///d%3A/code/cocursor/cursor_database_analysis.md",
      "forceFile": true,
      "options": {"override": "default"}
    }
  }
]
```

**7. 统计各类型数据的大小分布**
```sql
SELECT 
  CASE 
    WHEN key LIKE 'aiService.%' THEN 'AI服务'
    WHEN key LIKE 'composer.%' THEN 'Composer'
    WHEN key LIKE 'memento/%' THEN '编辑器记忆'
    WHEN key LIKE 'workbench.%' THEN '工作台'
    WHEN key LIKE 'cursor/%' THEN 'Cursor配置'
    ELSE '其他'
  END as category,
  COUNT(*) as count,
  SUM(length(value)) as total_size,
  ROUND(AVG(length(value)), 2) as avg_size
FROM ItemTable
WHERE key NOT LIKE '__$__%'
GROUP BY category
ORDER BY total_size DESC;
```

**8. 查找最大的数据项**
```sql
SELECT key, length(value) as size
FROM ItemTable
ORDER BY size DESC
LIMIT 10;
```
结果示例：
```
key: aiService.prompts, size: 80076
key: memento/workbench.editors.files.textFileEditor, size: 46447
key: aiService.generations, size: 15298
key: composer.composerData, size: 15165
...
```

---

## 深度分析：关键数据结构详解

### 1. AI 回复内容（Generations 详情）

#### 数据结构

`aiService.generations` 存储了 AI 的所有生成记录，包含 AI 的实际回复内容和操作描述。

**数据格式**：
```json
[
  {
    "unixMs": 1768644518895,
    "generationUUID": "52eebe9c-d350-4d0a-99cf-7f29b29ea0fe",
    "type": "composer",
    "textDescription": "继续实施"
  },
  {
    "unixMs": 1768644530445,
    "generationUUID": "f2c37877-dbce-4465-86c5-a3abadf187f8",
    "type": "composer",
    "textDescription": "/openspec-apply  继续实施"
  },
  ...
]
```

**字段说明**：
- `unixMs`: Unix 时间戳（毫秒），记录 AI 生成的时间
- `generationUUID`: 生成的唯一标识符，可用于关联其他数据
- `type`: 生成类型，常见值：
  - `"composer"`: Composer 模式的生成
  - `"chat"`: 普通聊天模式的生成
  - `"tab"`: Tab 自动补全的生成
- `textDescription`: **关键字段** - AI 的实际回复内容或操作描述
  - 包含 AI 回复的文本摘要
  - 包含执行的命令或操作
  - 可能包含错误信息或调试输出

#### 实际数据示例

**示例 1: 简单回复**
```json
{
  "unixMs": 1768645313794,
  "generationUUID": "a2e19ec3-624b-4cc6-8e36-03fc0635d5ab",
  "type": "composer",
  "textDescription": "继续"
}
```

**示例 2: 包含命令的回复**
```json
{
  "unixMs": 1768648068251,
  "generationUUID": "486f173e-4981-462d-b6a0-d39ad98a92f6",
  "type": "composer",
  "textDescription": "make compile-debug"
}
```

**示例 3: 包含详细错误信息的回复**
```json
{
  "unixMs": 1768648751347,
  "generationUUID": "b750bb05-7215-40b8-9097-f4ad1f970386",
  "type": "composer",
  "textDescription": "workbench.desktop.main.js:55  WARN No search provider registered for scheme: file, waiting\nwarn @ workbench.desktop.main.js:55\nworkbench.desktop.main.js:55  WARN [Debug] Function 'testLinearIssues' already registered, overwriting\n..."
}
```

**示例 4: 包含项目需求的回复**
```json
{
  "unixMs": 1768646262150,
  "generationUUID": "3d000ed4-d78b-4605-83fd-7510a36c0f32",
  "type": "composer",
  "textDescription": "当前项目需要做一个vscode插件，用来对cursor后台数据进行分析，并进行分析。前端目录为 @co-extension/  后端目录为： @backend/ \n现在需要创建初始化。 @.cursor/commands/openspec-proposal.md  是否还存在问题，没有就先创建目录。"
}
```

#### 数据统计

从工作区 `d4b798d47e9a14d74eb7965f996e8739` 的数据：
- **总生成记录数**: 50 条
- **数据大小**: 15,298 字节
- **平均每条记录**: 约 306 字节

#### 使用场景

1. **AI 质量分析**
   - 通过 `textDescription` 分析 AI 回复的质量
   - 统计有效回复 vs 空回复的比例
   - 识别 AI 处理的问题类型

2. **对话流程重建**
   - 结合 `aiService.prompts` 和 `aiService.generations` 重建完整对话
   - 分析用户提问 → AI 回复的对应关系

3. **代码变更追踪**
   - `generationUUID` 可能关联到具体的代码变更
   - 需要进一步探索 `cursorDiskKV` 表中的关联数据

4. **时间线分析**
   - 通过 `unixMs` 分析 AI 使用的时间分布
   - 计算对话间隔和响应时间

---

### 2. 工作区路径与 ID 的映射关系

#### 映射机制

Cursor 使用**工作区 ID（哈希值）**来标识每个工作区，而实际的项目路径存储在 `workspace.json` 文件中。

#### 映射文件位置

每个工作区目录下都有一个 `workspace.json` 文件：
```
C:\Users\TANG\AppData\Roaming\Cursor\User\workspaceStorage\{workspaceId}\workspace.json
```

**文件内容格式**：
```json
{
  "folder": "file:///d%3A/code/cocursor"
}
```

#### 实际映射数据

| 工作区 ID | 项目路径 | 说明 |
|----------|---------|------|
| `d4b798d47e9a14d74eb7965f996e8739` | `file:///d%3A/code/cocursor` | 当前分析的工作区 |
| `8c9de0af69d4b33fe3e10c7cc030549f` | `file:///d%3A/2025/github/nsq` | NSQ 项目 |
| `bf0871dff1e9e2d800e8ff4f97c9444b` | `file:///d%3A/code/wecode` | WeCode 项目 |
| `c9bf9293c2c47eef24da0f69e3884363` | `file:///d%3A/2025/github` | GitHub 项目 |
| `ca45e3de82bddb73a4fc1291a910f3e9` | `file:///d%3A/code/wecode` | WeCode 项目（另一个实例） |
| `861ca156f6e5c2aad73afd2854c92261` | `file:///c%3A/Users/TANG/Videos/goanalysis` | Go 分析项目 |
| `f448e6e71945186fe5a505fbaf753753` | `file:///d%3A/2025/github/nsq` | NSQ 项目（另一个实例） |

**注意**：
- 路径使用 URL 编码格式（`%3A` 代表 `:`）
- 同一个项目路径可能对应多个工作区 ID（不同打开方式或时间）
- Windows 路径格式：`file:///d%3A/...` 对应 `D:\...`

#### 全局存储中的映射信息

**1. `storage.json` 文件**
位置：`C:\Users\TANG\AppData\Roaming\Cursor\User\globalStorage\storage.json`

包含内容：
```json
{
  "backupWorkspaces": {
    "folders": [
      {
        "folderUri": "file:///d%3A/code/cocursor"
      }
    ]
  },
  "profileAssociations": {
    "workspaces": {
      "file:///d%3A/code/cocursor": "__default__profile__",
      "file:///d%3A/code/wecode": "__default__profile__",
      ...
    }
  },
  "windowsState": {
    "lastActiveWindow": {
      "folder": "file:///d%3A/code/cocursor",
      "backupPath": "C:\\Users\\TANG\\AppData\\Roaming\\Cursor\\Backups\\cab63a7a7177420a33b204c9b71b6aa5"
    }
  }
}
```

**2. `history.recentlyOpenedPathsList`**
位置：全局存储 `state.vscdb` 的 `ItemTable`

包含最近打开的工作区路径列表：
```json
{
  "entries": [
    {"folderUri": "file:///d%3A/code/cocursor"},
    {"folderUri": "file:///d%3A/code/wecode"},
    {"folderUri": "file:///d%3A/2025/github/nsq"},
    ...
  ]
}
```

#### 查询工作区 ID 的方法

**方法 1: 通过路径查找工作区 ID**
```bash
# 遍历所有工作区，查找匹配的路径
for dir in workspaceStorage/*/; do
  if [ -f "$dir/workspace.json" ]; then
    folder=$(cat "$dir/workspace.json" | jq -r '.folder')
    if [ "$folder" = "file:///d%3A/code/cocursor" ]; then
      echo "Found: $(basename "$dir")"
    fi
  fi
done
```

**方法 2: 通过工作区 ID 查找路径**
```bash
# 直接读取 workspace.json
cat "workspaceStorage/d4b798d47e9a14d74eb7965f996e8739/workspace.json"
```

**方法 3: 从全局存储查询**
```sql
-- 查询最近打开的工作区
SELECT value FROM ItemTable 
WHERE key = 'history.recentlyOpenedPathsList';
```

#### 实现建议

对于 Go 后端常驻服务，建议：

1. **启动时扫描映射**
   ```go
   // 扫描所有工作区，建立路径 -> ID 映射
   workspaceMap := make(map[string]string)
   workspaceDir := filepath.Join(userDataDir, "workspaceStorage")
   entries, _ := os.ReadDir(workspaceDir)
   for _, entry := range entries {
       workspaceJSON := filepath.Join(workspaceDir, entry.Name(), "workspace.json")
       if data, err := os.ReadFile(workspaceJSON); err == nil {
           var ws struct{ Folder string `json:"folder"` }
           json.Unmarshal(data, &ws)
           workspaceMap[ws.Folder] = entry.Name()
       }
   }
   ```

2. **监听工作区变化**
   - 使用文件系统监听（如 `fsnotify`）监听 `workspaceStorage` 目录
   - 当新的 `workspace.json` 创建时，更新映射

3. **缓存机制**
   - 将映射关系缓存到内存
   - 定期刷新或事件驱动更新

---

### 3. Composer 对话树结构（对话上下文）

#### 数据结构概览

`composer.composerData` 存储了完整的 Composer 会话树结构，包含所有对话的层级关系、文件引用、代码变更统计等关键信息。

**顶层结构**：
```json
{
  "allComposers": [
    {
      "type": "head",
      "composerId": "e90362b7-ddf6-4f4a-8064-7683eeabac5b",
      "name": "Sqlite3 command installation",
      ...
    },
    ...
  ]
}
```

#### 完整字段说明

**基础信息**：
- `type`: 类型，`"head"` 表示主会话
- `composerId`: 会话唯一标识符（**关键字段**，用于关联其他数据）
- `name`: 会话名称（用户输入或 AI 生成）
- `createdAt`: 创建时间戳（毫秒）
- `lastUpdatedAt`: 最后更新时间戳（毫秒）

**模式信息**：
- `unifiedMode`: 统一模式，常见值：
  - `"agent"`: Agent 模式（AI 主动执行）
  - `"edit"`: 编辑模式
  - `"chat"`: 聊天模式
- `forceMode`: 强制模式，通常为 `"edit"`

**代码变更统计**（**关键字段**）：
- `totalLinesAdded`: 总添加行数
- `totalLinesRemoved`: 总删除行数
- `filesChangedCount`: 变更的文件数量
- `subtitle`: 涉及的文件列表（逗号分隔）

**上下文使用**：
- `contextUsagePercent`: 上下文使用百分比（**用于计算 Session 熵**）
  - 值越高，说明对话引用了更多文件
  - 可用于判断对话复杂度

**状态标志**：
- `hasUnreadMessages`: 是否有未读消息
- `hasBlockingPendingActions`: 是否有阻塞的待处理操作
- `isArchived`: 是否已归档
- `isDraft`: 是否为草稿
- `isWorktree`: 是否为工作树
- `isSpec`: 是否为规范文档

**层级关系**：
- `numSubComposers`: 子会话数量
- `referencedPlans`: 引用的计划列表

**Git 信息**：
- `createdOnBranch`: 创建时的 Git 分支

#### 实际数据示例

**示例 1: 大型会话（高复杂度）**
```json
{
  "type": "head",
  "composerId": "03e41bc4-c700-491a-a01a-2048468bd7b8",
  "name": "Golang 后端服务框架",
  "lastUpdatedAt": 1768656147170,
  "createdAt": 1768650378012,
  "unifiedMode": "agent",
  "forceMode": "edit",
  "contextUsagePercent": 65.186,
  "totalLinesAdded": 1704,
  "totalLinesRemoved": 116,
  "filesChangedCount": 39,
  "subtitle": "DAEMON_MANAGER.md, extension.ts, daemonManager.ts, lock_test.go, SINGLETON_LOCK.md",
  "createdOnBranch": "main"
}
```
**分析**：
- 高上下文使用率（65.19%），说明引用了大量文件
- 大量代码变更（1704 行添加，116 行删除）
- 涉及 39 个文件，复杂度高

**示例 2: 简单会话（低复杂度）**
```json
{
  "type": "head",
  "composerId": "f2dc90df-54c2-4bc6-abb0-8ac0c92d704c",
  "name": "图标优化",
  "lastUpdatedAt": 1768651830147,
  "createdAt": 1768651820874,
  "unifiedMode": "agent",
  "forceMode": "edit",
  "contextUsagePercent": 7.094,
  "totalLinesAdded": 20,
  "totalLinesRemoved": 5,
  "filesChangedCount": 1,
  "subtitle": "icon.svg",
  "createdOnBranch": "main"
}
```
**分析**：
- 低上下文使用率（7.09%），简单任务
- 少量代码变更（20 行添加，5 行删除）
- 仅涉及 1 个文件

**示例 3: 文档生成会话**
```json
{
  "type": "head",
  "composerId": "e90362b7-ddf6-4f4a-8064-7683eeabac5b",
  "name": "Sqlite3 command installation",
  "lastUpdatedAt": 1768656852120,
  "createdAt": 1768655012672,
  "unifiedMode": "agent",
  "forceMode": "edit",
  "contextUsagePercent": 43.221,
  "totalLinesAdded": 605,
  "totalLinesRemoved": 0,
  "filesChangedCount": 1,
  "subtitle": "cursor_database_analysis.md",
  "createdOnBranch": "main"
}
```
**分析**：
- 中等上下文使用率（43.22%）
- 纯添加操作（605 行，0 行删除），典型的文档生成
- 单文件操作

#### 数据统计

从工作区 `d4b798d47e9a14d74eb7965f996e8739` 的数据：
- **总会话数**: 26 个 Composer 会话
- **数据大小**: 15,241 字节
- **平均每个会话**: 约 586 字节

#### Session 熵计算

**熵的定义**：
Session 熵用于衡量对话的复杂度和信息量，可以通过以下指标计算：

1. **上下文使用率** (`contextUsagePercent`)
   - 直接反映对话引用的文件数量
   - 值越高，熵越大

2. **文件变更密度**
   ```
   文件变更密度 = filesChangedCount / (totalLinesAdded + totalLinesRemoved)
   ```
   - 密度越高，说明变更分散在更多文件中，熵越大

3. **对话轮数**（需要结合 `aiService.prompts` 和 `aiService.generations`）
   - 通过时间戳计算对话持续时间
   - 通过生成记录数估算对话轮数

4. **代码变更规模**
   ```
   变更规模 = totalLinesAdded + totalLinesRemoved
   ```
   - 规模越大，熵越大

**熵计算公式示例**：
```python
def calculate_session_entropy(composer):
    # 基础熵 = 上下文使用率
    base_entropy = composer['contextUsagePercent']
    
    # 文件密度熵 = 文件数 / 代码行数（归一化）
    code_lines = composer['totalLinesAdded'] + composer['totalLinesRemoved']
    file_density = composer['filesChangedCount'] / max(code_lines, 1) * 100
    
    # 规模熵 = log(代码行数)（归一化）
    scale_entropy = math.log(max(code_lines, 1)) * 10
    
    # 综合熵
    total_entropy = base_entropy * 0.4 + file_density * 0.3 + scale_entropy * 0.3
    
    return total_entropy
```

#### 对话过长检测

**检测标准**：
1. **上下文使用率过高** (> 80%)
   - 说明引用了过多文件，可能超出 AI 处理能力

2. **对话持续时间过长**
   ```
   持续时间 = (lastUpdatedAt - createdAt) / 1000 / 60  # 分钟
   ```
   - 建议阈值：> 60 分钟

3. **代码变更规模过大**
   - 建议阈值：> 5000 行

4. **文件变更数量过多**
   - 建议阈值：> 50 个文件

**示例检测逻辑**：
```python
def is_conversation_too_long(composer):
    duration_minutes = (composer['lastUpdatedAt'] - composer['createdAt']) / 1000 / 60
    total_changes = composer['totalLinesAdded'] + composer['totalLinesRemoved']
    
    warnings = []
    if composer['contextUsagePercent'] > 80:
        warnings.append("上下文使用率过高")
    if duration_minutes > 60:
        warnings.append(f"对话持续时间过长 ({duration_minutes:.1f} 分钟)")
    if total_changes > 5000:
        warnings.append(f"代码变更规模过大 ({total_changes} 行)")
    if composer['filesChangedCount'] > 50:
        warnings.append(f"文件变更数量过多 ({composer['filesChangedCount']} 个文件)")
    
    return len(warnings) > 0, warnings
```

#### 文件引用分析

**提取文件列表**：
```python
def extract_referenced_files(composer):
    # 从 subtitle 字段提取
    files = [f.strip() for f in composer['subtitle'].split(',')]
    return files
```

**文件引用统计**：
- 统计每个文件被引用的次数
- 识别最常被引用的文件
- 分析文件类型分布（代码文件 vs 文档文件）

#### 使用场景

1. **工作流分析**
   - 通过 `composerId` 关联 `aiService.prompts` 和 `aiService.generations`
   - 重建完整的对话流程
   - 分析用户的工作模式

2. **效率分析**
   - 计算每个会话的代码产出效率
   - 识别高效 vs 低效的会话模式

3. **日报/周报生成**
   - 汇总所有会话的代码变更统计
   - 生成工作摘要和成果报告

4. **质量评估**
   - 通过上下文使用率和代码变更规模评估 AI 辅助质量
   - 识别需要优化的会话模式

---

## 总结

通过以上深度分析，我们获得了：

1. **AI 回复内容**：`aiService.generations` 提供了 AI 的实际回复和操作描述
2. **工作区映射**：`workspace.json` 提供了路径到 ID 的映射关系
3. **对话树结构**：`composer.composerData` 提供了完整的会话上下文和统计信息

这些数据足以支持：
- ✅ 工作流分析和可视化
- ✅ Session 熵统计和对话质量评估
- ✅ 日报/周报自动化生成
- ✅ 代码变更追踪和效率分析
