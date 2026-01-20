# Cursor 数据库完整分析总结

## 概述

本文档总结了 Cursor 在 Windows 下的全局数据库和工作区数据库的完整分析结果，包括所有字段的详细分析，特别是 KV 字段的深度分析。

## 分析日期
2026-01-20

## 数据库位置

### 全局数据库
- **路径**: `C:\Users\TANG\AppData\Roaming\Cursor\User\globalStorage\state.vscdb`
- **作用**: 存储跨所有工作区共享的配置和状态

### 工作区数据库
- **路径**: `C:\Users\TANG\AppData\Roaming\Cursor\User\workspaceStorage\{workspace_id}\state.vscdb`
- **作用**: 存储每个工作区特定的数据

## 数据库结构

所有数据库都包含两个表：

### 1. ItemTable
```sql
CREATE TABLE ItemTable (
    key TEXT UNIQUE ON CONFLICT REPLACE,
    value BLOB
)
```

### 2. cursorDiskKV
```sql
CREATE TABLE cursorDiskKV (
    key TEXT UNIQUE ON CONFLICT REPLACE,
    value BLOB
)
```

---

## 全局数据库分析

### ItemTable 统计

**总记录数**: 271 条

#### 分类统计

| 分类 | 记录数 | 总大小 (字节) | 说明 |
|------|--------|---------------|------|
| workbench | 159 | 180,739 | 工作台状态 |
| other | 41 | 24,742 | 其他数据 |
| cursor_config | 17 | 86 | Cursor 配置 |
| cursor_auth | 8 | 930 | 认证信息 |
| terminal | 7 | 22,739 | 终端配置 |
| ai_code_tracking | 7 | 1,287 | AI 代码追踪 |
| cursor_ai_config | 6 | 24,564 | AI 配置 |
| memento | 6 | 4,094 | 编辑器记忆 |
| anysphere | 4 | 99 | Cursor 功能配置 |
| telemetry | 3 | 87 | 遥测数据 |
| system_marker | 2 | 15,165 | 系统标记 |
| extensions | 2 | 127 | 扩展数据 |
| composer | 2 | 8 | Composer 配置 |
| color_theme | 1 | 8,075 | 颜色主题 |
| **icon_theme** | 1 | **1,374,686** | 图标主题（最大） |
| editor_font | 1 | 982 | 编辑器字体 |
| history | 1 | 1,118 | 历史记录 |
| background_composer | 1 | 2 | 后台 Composer |
| mcp_service | 1 | 55 | MCP 服务 |
| scm | 1 | 31 | 源代码管理 |

#### 大小分布
- **最小**: 1 字节
- **最大**: 1,374,686 字节
- **中位数**: 92 字节

---

### cursorDiskKV 统计

**总记录数**: 64,043 条

#### 分类统计

| 分类 | 记录数 | 总大小 (字节) | 说明 |
|------|--------|---------------|------|
| **agent_kv_blob** | **42,358** | **429,864,504** | Agent KV Blob 数据（最大） |
| other | 18,408 | 168,145,155 | 其他数据 |
| composer | 3,278 | 40,055,472 | Composer 相关数据 |

#### 键前缀 TOP 10

| 键前缀 | 记录数 | 说明 |
|--------|--------|------|
| agentKv:blob: | 42,358 | Agent Blob 数据 |
| composer. | 3,278 | Composer 配置 |
| composerData:{UUID} | 1 | Composer 数据（每个 UUID 一条） |
| inlineDiffs-{workspace_id} | 1 | 内联差异（每个工作区一条） |

#### 值类型统计
- **text**: 26,586 条
- **binary**: 21,588 条
- **json**: 15,772 条

---

## KV 字段深度分析

### agentKv:blob:* 深度分析（采样 2000 条）

#### 类型分布
- **binary**: 1,007 条 (50.35%)
- **json**: 691 条 (34.55%)
- **text**: 302 条 (15.10%)

#### 大小分布
- **<1KB**: 946 条 (47.30%)
- **1-10KB**: 770 条 (38.50%)
- **10-100KB**: 228 条 (11.40%)
- **>100KB**: 56 条 (2.80%)

**结论**: 大部分 agentKv:blob:* 数据是小到中等大小的二进制或 JSON 数据。

---

### composer.* 深度分析（采样 5000 条）

#### 前缀分布

| 前缀 | 记录数 | 总大小 (字节) | 说明 |
|------|--------|---------------|------|
| **composer.content** | **3,279** | **40,113,120** | Composer 内容数据（主要） |
| composer.autoAccept | 2 | 53 | 自动接受配置 |

**结论**: composer.* 数据主要由 composer.content 组成，平均每条约 12.2 KB。

---

### inlineDiffs-* 深度分析

#### 工作区分布（16 个工作区）

发现 16 个不同的工作区 ID，每个工作区有一条 inlineDiffs 记录。这些记录用于存储工作区的内联代码差异。

---

### 所有 KV 数据分析（采样 10,000 条）

#### 模式分布

| 模式 | 记录数 | 总大小 (字节) | 平均大小 |
|------|--------|---------------|----------|
| agentKv:blob:* | 5,817 | 85,988,479 | 14,787 字节 |
| other | 3,605 | 50,986,082 | 14,145 字节 |
| composer.* | 467 | 9,180,307 | 19,658 字节 |
| composerData:* | 96 | 1,233,917 | 12,855 字节 |
| inlineDiffs-* | 15 | 6,458 | 430 字节 |

#### 值类型分布
- **binary**: 7,121 条 (71.21%)
- **json**: 2,067 条 (20.67%)
- **text**: 790 条 (7.90%)

#### 大小统计
- **最小**: 2 字节
- **最大**: 1,008,186 字节
- **中位数**: 2,454 字节
- **平均**: 14,772 字节

**结论**:
1. 大部分 KV 数据是二进制格式（71.21%）
2. agentKv:blob:* 是最大的数据类别，占总大小的 56.7%
3. 数据大小分布广泛，从几字节到 1MB 不等

---

## 工作区数据库分析

### 工作区概览

发现 **8 个工作区**：

| 工作区 ID | 项目名 | 路径 | ItemTable 记录数 |
|----------|--------|------|------------------|
| 861ca156f6e5c2aad73afd2854c92261 | goanalysis | c:/Users/TANG/Videos/goanalysis | 110 |
| 8c9de0af69d4b33fe3e10c7cc030549f | nsq | d:/2025/github/nsq | 144 |
| bf0871dff1e9e2d800e8ff4f97c9444b | wecode | d:/code/wecode | 77 |
| c9bf9293c2c47eef24da0f69e3884363 | github | d:/2025/github | 54 |
| ca45e3de82bddb73a4fc1291a910f3e9 | wecode | d:/code/wecode | 72 |
| **d4b798d47e9a14d74eb7965f996e8739** | **cocursor** | **d:/code/cocursor** | **284** |
| f448e6e71945186fe5a505fbaf753753 | nsq | d:/2025/github/nsq | 51 |

**注意**: 
- 同一个项目（nsq, wecode）可能有多个工作区 ID
- cocursor 项目（当前项目）的记录数最多（284 条）

### 工作区 ItemTable 分类（以 cocursor 为例）

#### 分类统计

| 分类 | 记录数 | 总大小 (字节) |
|------|--------|---------------|
| workbench | 235 | 24,276 |
| cursor_config | 13 | 47 |
| other | 10 | 2,520 |
| memento | 9 | 88,211 |
| debug | 4 | 627 |
| terminal | 3 | 1,740 |
| system_marker | 2 | 18,674 |
| **ai_service** | 2 | **694,152** |
| scm | 2 | 3,949 |
| cursor_auth | 1 | 24 |
| history | 1 | 29,186 |
| composer | 1 | 49,144 |
| anysphere | 1 | 695 |

#### 大小分布
- **最小**: 1 字节
- **最大**: 655,196 字节
- **中位数**: 55 字节

**重要发现**:
- `ai_service` 分类包含 **694,152 字节** 的数据，最大的一条记录是 655,196 字节
- 这主要是 `aiService.prompts` 和 `aiService.generations` 数据
- 这些数据包含了该工作区的所有 AI 对话历史

### 工作区 cursorDiskKV

**重要发现**: 所有工作区的 **cursorDiskKV 表都是空的**（0 条记录）。

这说明：
1. KV 存储（agentKv:blob:*, composer.* 等）只存储在全局数据库
2. 工作区只使用 ItemTable 存储工作区特定的数据
3. 这种设计可能为了减少数据重复和提高性能

---

## 数据分布特征

### 数据量对比

| 数据库 | ItemTable 记录数 | cursorDiskKV 记录数 | 主要数据 |
|--------|------------------|---------------------|----------|
| 全局 | 271 | 64,043 | agentKv:blob:*, composer.* |
| 工作区 | 平均 100+ | 0 | ai_service, workbench, memento |

### 存储策略

1. **全局数据库**:
   - 存储 Agent 相关的大量 KV 数据（agentKv:blob:*）
   - 存储 Composer 相关数据（composer.*）
   - 存储跨工作区的配置和状态

2. **工作区数据库**:
   - 存储工作区特定的 AI 对话历史（aiService.prompts, aiService.generations）
   - 存储工作区特定的编辑器状态（memento, workbench）
   - 不使用 cursorDiskKV 表

---

## 关键数据字段详解

### 1. agentKv:blob:* 字段

#### 特征
- **键格式**: `agentKv:blob:{hash}`
- **值类型**: 主要是二进制 (50.35%) 或 JSON (34.55%)
- **大小分布**: 47.30% < 1KB, 38.50% 1-10KB, 11.40% 10-100KB, 2.80% > 100KB
- **总记录数**: 42,358 条（全局）
- **总大小**: ~429 MB

#### 用途分析
这些数据可能是 Cursor Agent 的缓存数据，包括：
- Agent 上下文缓存
- LLM 响应缓存
- 文件内容缓存
- 代码片段缓存

#### 哈希机制
使用哈希值作为键，说明：
- 可能是内容寻址存储（Content-Addressable Storage）
- 相同内容会被去重
- 适合缓存和去重场景

---

### 2. composer.content 字段

#### 特征
- **键格式**: `composer.content:{composerId}:{key}`
- **记录数**: 3,279 条（全局）
- **总大小**: ~40 MB
- **平均大小**: 12.2 KB

#### 用途分析
存储 Composer 会话的内容数据，包括：
- 对话历史
- 文件引用
- 代码变更
- 上下文信息

---

### 3. aiService.prompts 字段（工作区）

#### 特征
- **位置**: 工作区 ItemTable
- **键名**: `aiService.prompts`
- **值类型**: JSON 数组
- **大小**: 可达 655 KB（cocursor 项目）

#### 数据结构
```json
[
  {
    "text": "用户输入的文本",
    "commandType": 4
  },
  ...
]
```

#### 用途
存储该工作区所有与 AI 的对话提示历史。

---

### 4. aiService.generations 字段（工作区）

#### 特征
- **位置**: 工作区 ItemTable
- **键名**: `aiService.generations`
- **值类型**: JSON 数组
- **大小**: 通常 10-50 KB

#### 数据结构
```json
[
  {
    "unixMs": 1768644518895,
    "generationUUID": "52eebe9c-d350-4d0a-99cf-7f29b29ea0fe",
    "type": "composer",
    "textDescription": "AI 的实际回复内容"
  },
  ...
]
```

#### 用途
存储 AI 的所有生成记录，包含 AI 的实际回复内容和操作描述。

---

### 5. composer.composerData 字段（工作区）

#### 特征
- **位置**: 工作区 ItemTable
- **键名**: `composer.composerData`
- **值类型**: JSON 对象
- **大小**: 通常 10-50 KB

#### 数据结构
```json
{
  "allComposers": [
    {
      "type": "head",
      "composerId": "e90362b7-ddf6-4f4a-8064-7683eeabac5b",
      "name": "会话名称",
      "createdAt": 1768655012672,
      "lastUpdatedAt": 1768656852120,
      "unifiedMode": "agent",
      "forceMode": "edit",
      "contextUsagePercent": 43.221,
      "totalLinesAdded": 605,
      "totalLinesRemoved": 0,
      "filesChangedCount": 1,
      "subtitle": "涉及文件列表",
      "createdOnBranch": "main",
      ...
    },
    ...
  ]
}
```

#### 用途
存储完整的 Composer 会话树结构，包含所有对话的层级关系、文件引用、代码变更统计等关键信息。

---

## 数据访问建议

### 读取策略

1. **全局数据**:
   - 用户认证信息 → 全局 ItemTable
   - Agent 缓存数据 → 全局 cursorDiskKV (agentKv:blob:*)
   - Composer 内容数据 → 全局 cursorDiskKV (composer.*)
   - 跨工作区配置 → 全局 ItemTable

2. **工作区数据**:
   - AI 对话历史 → 工作区 ItemTable (aiService.*)
   - Composer 会话数据 → 工作区 ItemTable (composer.composerData)
   - 编辑器状态 → 工作区 ItemTable (memento, workbench)
   - 项目特定配置 → 工作区 ItemTable

### 数据解析

1. **JSON 数据**:
   - 大部分 value 字段是 JSON 格式
   - 使用 `json.loads()` 解析
   - 注意处理可能的 JSON 语法错误

2. **二进制数据**:
   - agentKv:blob:* 数据大部分是二进制
   - 尝试 UTF-8 解码，失败则作为二进制处理
   - 可能包含 base64 编码的数据

3. **文本数据**:
   - 配置值通常是纯文本
   - 可能是数字、布尔值或字符串

### 性能优化

1. **批量读取**:
   - 使用事务批量读取
   - 避免 N+1 查询问题

2. **缓存策略**:
   - 缓存全局数据（变化较少）
   - 缓存工作区映射（workspace ID → 路径）

3. **索引使用**:
   - SQLite 自动为 UNIQUE 约束创建索引
   - key 字段已自动索引

---

## 数据安全和隐私

### 敏感数据

数据库包含以下敏感信息：
- 用户认证信息（tokens, email）
- 项目文件路径
- AI 对话历史
- 代码内容

### 使用建议

1. **访问权限**:
   - 确保数据库文件访问权限正确
   - 不要将数据库文件提交到版本控制

2. **数据脱敏**:
   - 分析时注意脱敏敏感信息
   - 不要在报告中包含完整文件路径或用户数据

3. **备份**:
   - 分析前先备份数据库
   - 使用只读模式访问

---

## 总结

### 数据库架构特点

1. **双层存储架构**:
   - 全局数据库：跨工作区共享的数据
   - 工作区数据库：工作区特定的数据

2. **KV 存储集中化**:
   - agentKv:blob:* 和 composer.* 数据只存储在全局数据库
   - 工作区数据库不使用 cursorDiskKV 表

3. **数据分类清晰**:
   - ItemTable: 配置和状态数据
   - cursorDiskKV: Agent 和 Composer 的缓存数据

### 关键发现

1. **agentKv:blob:* 是最大的数据类别**:
   - 42,358 条记录，~429 MB
   - 主要是二进制或 JSON 格式
   - 使用哈希作为键，可能用于内容寻址

2. **composer.content 是第二大数据类别**:
   - 3,279 条记录，~40 MB
   - 存储 Composer 会话内容

3. **工作区 AI 对话数据较大**:
   - cocursor 项目的 ai_service 数据达到 694 KB
   - 包含完整的对话历史

4. **所有工作区的 cursorDiskKV 表都是空的**:
   - KV 存储完全集中化在全局数据库
   - 工作区只使用 ItemTable

### 使用场景

这些数据可用于：
1. 工作流分析和可视化
2. AI 使用统计和效率分析
3. 日报/周报自动化生成
4. 代码变更追踪
5. 数据备份和迁移

### 后续建议

1. **定期清理**:
   - 清理过期的 agentKv:blob:* 数据
   - 归档旧的对话历史

2. **数据压缩**:
   - 对大型 JSON 数据进行压缩
   - 考虑使用更高效的存储格式

3. **监控增长**:
   - 监控数据库文件大小
   - 设置定期备份策略

---

## 附录：分析脚本

### 已生成的报告

1. **full_db_analysis_report.json**
   - 完整的数据库分析报告（JSON 格式）
   - 包含所有表和字段的统计信息

2. **full_db_analysis_report.txt**
   - 完整的数据库分析报告（文本格式）
   - 便于人类阅读

3. **deep_kv_analysis_report.json**
   - KV 字段深度分析报告（JSON 格式）
   - 包含 agentKv:blob:*, composer.* 等的详细分析

### 分析脚本

1. **analyze_cursor_db.py**
   - 基础数据库分析脚本
   - 快速统计和分类

2. **full_db_analysis.py**
   - 完整数据库分析脚本
   - 深度分析所有表和字段

3. **deep_kv_analysis.py**
   - KV 字段深度分析脚本
   - 专门分析 cursorDiskKV 表的数据

---

*本报告生成日期：2026-01-20*
*分析工具：Python 3.13 + SQLite3*
