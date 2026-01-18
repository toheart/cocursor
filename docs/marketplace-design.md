# 插件市场设计文档

## 1. 概述

插件市场允许用户安装、管理和使用插件。每个插件必须包含一个 Claude Skill，可选地包含 MCP 服务器配置和 Cursor Command。

### 1.1 核心概念

- **插件（Plugin）**：包含 Skill（必须）+ MCP（可选）+ Command（可选）的组合
- **Skill**：Claude 自定义技能，存储在 `~/.claude/skills/<skill-name>/`
- **MCP**：Model Context Protocol 服务器配置，存储在 `~/.cursor/mcp.json`（全局配置）
- **Command**：Cursor 自定义命令，存储在 `~/.cursor/commands/<command-id>.md`（全局配置）

### 1.2 约束条件

- 插件必须至少包含一个 Skill
- Skill 名称在创建时不允许重复
- 版本更新直接覆盖现有文件
- MCP 仅支持 SSE 和 streamable-http 传输方式（不支持 stdio）
- MCP 仅支持全局配置（~/.cursor/mcp.json）
- 环境变量需要用户手动配置，安装时提示

## 2. 数据结构

### 2.1 插件元数据（plugin.json）

```json
{
  "id": "code-review-assistant",
  "name": "代码审查助手",
  "description": "AI 驱动的代码审查，包含 Skill、MCP 和 Command",
  "author": "CoCursor Team",
  "version": "1.0.0",
  "category": "AI",
  "skill": {
    "skill_name": "code-review"
  },
  "mcp": {
    "server_name": "code-review-mcp",
    "transport": "sse",
    "url": "http://localhost:19961/mcp/sse",
    "headers": {
      "Authorization": "Bearer ${env:CODE_REVIEW_TOKEN}"
    }
  },
  "command": {
    "command_id": "code-review",
    "scope": "global"
  }
}
```

### 2.2 Go 数据结构

```go
type Plugin struct {
    ID              string   `json:"id"`
    Name            string   `json:"name"`
    Description     string   `json:"description"`
    Author          string   `json:"author"`
    Version         string   `json:"version"`
    Icon            string   `json:"icon,omitempty"`
    Category        string   `json:"category"`
    Installed       bool     `json:"installed"`
    InstalledVersion string  `json:"installed_version,omitempty"`
    Skill           SkillComponent   `json:"skill"`
    MCP             *MCPComponent    `json:"mcp,omitempty"`
    Command         *CommandComponent `json:"command,omitempty"`
}

type SkillComponent struct {
    SkillName string `json:"skill_name"`
}

type MCPComponent struct {
    ServerName string            `json:"server_name"`
    Transport  string            `json:"transport"` // "sse" | "streamable-http"
    URL        string            `json:"url"`
    Headers    map[string]string `json:"headers,omitempty"`
}

type CommandComponent struct {
    CommandID string `json:"command_id"`
    Scope     string `json:"scope"` // "global"
}
```

## 3. 存储结构

### 3.1 内置插件（embed）

```
backend/internal/marketplace/plugins/
  ├── code-review-assistant/
  │   ├── plugin.json
  │   ├── skill/
  │   │   ├── SKILL.md
  │   │   ├── scripts/          # 可选
  │   │   ├── references/        # 可选
  │   │   └── assets/            # 可选
  │   └── command/               # 可选
  │       └── command.md
  └── simple-skill/
      ├── plugin.json
      └── skill/
          └── SKILL.md
```

**注意**：MCP 配置信息存储在 `plugin.json` 中，不需要单独的目录或文件。安装时将配置写入到 `~/.cursor/mcp.json`。

### 3.2 安装位置

- **Skill**: `~/.claude/skills/<skill_name>/`
  - 包含完整的 Skill 目录结构（SKILL.md、scripts/、references/、assets/）
  
- **MCP 配置**: `~/.cursor/mcp.json`（全局配置文件）
  - 配置格式：
  ```json
  {
    "mcpServers": {
      "server-name": {
        "type": "sse" | "streamable-http",
        "url": "http://localhost:19961/mcp/sse",
        "headers": {
          "Authorization": "Bearer ${env:TOKEN}"
        }
      }
    }
  }
  ```
  - 安装时：读取插件中的 MCP 配置，添加到 `mcpServers` 对象中
  - 卸载时：从 `mcpServers` 中删除对应配置
  
- **Command**: `~/.cursor/commands/<command_id>.md`
  - 单个 Markdown 文件，包含命令定义
  
- **状态文件**: `~/.cocursor/plugins-state.json`
  - 记录已安装插件及其版本信息

## 4. API 设计

### 4.1 获取插件列表

```
GET /api/v1/marketplace/plugins
Query params:
  - category: 分类 (可选)
  - search: 搜索关键词 (可选)
  - installed: true|false (可选)

Response: {
  "code": 200,
  "data": {
    "plugins": [...],
    "total": 10
  }
}
```

### 4.2 获取插件详情

```
GET /api/v1/marketplace/plugins/:id

Response: {
  "code": 200,
  "data": {
    "plugin": {...}
  }
}
```

### 4.3 获取已安装插件列表

```
GET /api/v1/marketplace/installed

Response: {
  "code": 200,
  "data": {
    "plugins": [...],
    "total": 5
  }
}
```

### 4.4 安装插件

```
POST /api/v1/marketplace/plugins/:id/install
Request Body: {
  "workspace_path": "/path/to/workspace" // 工作区路径（必需）
}

Response: {
  "code": 200,
  "data": {
    "success": true,
    "message": "安装成功",
    "env_vars": ["CODE_REVIEW_TOKEN"] // 如果需要配置环境变量
  }
}
```

**说明**：
- `workspace_path` 是必需参数，用于标识要安装插件的工作区
- 安装时会更新该工作区的 `AGENTS.md` 文件，添加技能说明
- Skill 文件安装到全局目录 `~/.claude/skills/`（所有工作区共享）
- MCP 配置安装到全局目录 `~/.cursor/mcp.json`（所有工作区共享）
- Command 文件安装到全局目录 `~/.cursor/commands/`（所有工作区共享）

### 4.5 卸载插件

```
POST /api/v1/marketplace/plugins/:id/uninstall
Request Body: {
  "workspace_path": "/path/to/workspace" // 工作区路径（必需）
}

Response: {
  "code": 200,
  "data": {
    "success": true,
    "message": "卸载成功"
  }
}
```

**说明**：
- `workspace_path` 是必需参数，用于标识要卸载插件的工作区
- 卸载时会从该工作区的 `AGENTS.md` 文件中移除技能说明
- 全局文件（Skill、MCP、Command）会被删除（影响所有工作区）

### 4.6 检查插件状态

```
GET /api/v1/marketplace/plugins/:id/status

Response: {
  "code": 200,
  "data": {
    "installed": true,
    "installed_version": "1.0.0",
    "latest_version": "1.0.0"
  }
}
```

## 5. 实现阶段

### 阶段 1：基础数据结构与模型（3-4 个文件）
- 创建 domain 层模型
- 定义插件数据结构
- 创建状态管理结构

### 阶段 2：插件加载与扫描（2-3 个文件）
- 实现 embed 文件系统扫描
- 实现插件元数据加载
- 实现状态文件读写

### 阶段 3：Skill 安装功能（2-3 个文件）
- 实现 Skill 文件复制
- 实现目录结构创建
- 实现 Skill 名称冲突检查

### 阶段 4：MCP 配置功能（2-3 个文件）
- 实现 MCP 配置读取/写入
- 实现环境变量检测
- 集成到安装流程

### 阶段 5：Command 安装功能（1-2 个文件）
- 实现 Command 文件复制
- 集成到安装流程

### 阶段 6：服务层与 Handler（3-4 个文件）
- 实现 PluginService
- 实现 MarketplaceHandler
- 注册路由

### 阶段 7：前端集成（2-3 个文件）
- 更新 Marketplace 组件
- 实现 API 调用
- 添加环境变量提示

## 6. 关键实现细节

### 6.1 插件扫描

启动时扫描 `internal/marketplace/plugins/` 目录，读取每个插件目录下的 `plugin.json`，加载插件元数据。

### 6.2 状态管理

使用 `~/.cocursor/plugins-state.json` 存储已安装插件信息：

```json
{
  "installed_plugins": {
    "code-review-assistant": {
      "version": "1.0.0",
      "installed_at": "2026-01-20T10:00:00Z"
    }
  }
}
```

### 6.3 MCP 安装流程

MCP 配置的安装流程：

1. **读取现有配置**
   - 读取 `~/.cursor/mcp.json` 文件
   - 如果文件不存在，创建空配置对象 `{"mcpServers": {}}`
   - 支持 JSONC 格式（需要移除注释）

2. **构建 MCP 配置对象**
   ```go
   mcpConfig := map[string]interface{}{
       "type": plugin.MCP.Transport,  // "sse" 或 "streamable-http"
       "url":  plugin.MCP.URL,
   }
   if len(plugin.MCP.Headers) > 0 {
       mcpConfig["headers"] = plugin.MCP.Headers  // 原样写入，不处理变量插值
   }
   ```

3. **添加到配置**
   - 将配置添加到 `mcpServers[server_name]` 中
   - 如果 `server_name` 已存在，覆盖现有配置

4. **写入配置文件**
   - 格式化 JSON（2 空格缩进）
   - 写入到 `~/.cursor/mcp.json`

5. **卸载流程**
   - 从 `mcpServers` 中删除对应的 `server_name`
   - 如果 `mcpServers` 为空，删除整个 `mcpServers` 键
   - 写入更新后的配置

**注意事项**：
- MCP 配置仅支持全局配置（`~/.cursor/mcp.json`），不支持项目级配置
- 环境变量（`${env:VAR}`）原样写入，不进行插值处理
- 配置格式必须符合 Cursor 的 MCP 配置规范

### 6.4 环境变量检测

安装时检查 MCP headers 中是否包含 `${env:VAR}` 格式的变量，提取变量名并返回给前端提示用户配置。

**检测逻辑**：
- 遍历 MCP headers 的所有值
- 使用正则表达式匹配 `${env:VAR_NAME}` 格式
- 提取所有环境变量名称
- 返回给前端，提示用户配置

### 6.5 冲突检查

安装前检查 Skill 名称是否已存在，如果存在且不是同一插件，拒绝安装。

**检查逻辑**：
- 检查 `~/.claude/skills/<skill_name>/` 目录是否存在
- 如果存在，检查状态文件中是否记录为同一插件
- 如果是不同插件，拒绝安装并返回错误
- 如果是同一插件（版本更新），允许覆盖

### 6.6 版本更新

更新时直接覆盖现有文件，不进行备份。

## 7. 错误处理

- 文件不存在：返回 404
- 插件已安装：返回 400，提示已安装
- Skill 名称冲突：返回 400，提示冲突
- 文件操作失败：返回 500，记录错误日志
- 配置写入失败：返回 500，记录错误日志

## 8. 测试策略

- 单元测试：每个服务方法
- 集成测试：完整安装/卸载流程
- 文件系统测试：使用临时目录模拟
- 配置测试：验证 MCP 配置正确性
