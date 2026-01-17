# Project Context

## Purpose

cocursor 是一个 VS Code/Cursor 插件，帮助团队成员共享 AI 聊天记录，提升 Cursor 团队协作效率。通过局域网 P2P 连接，团队成员可以实时查看彼此的 AI 对话，无需中心服务器，数据安全可控。

## Tech Stack

### VS Code 插件 (extension/)

- **语言**: TypeScript
- **框架**: VS Code Extension API
- **构建工具**: esbuild
- **UI**: VS Code TreeView + Webview
- **HTTP 客户端**: Axios

### Go Daemon (daemon/)

- **语言**: Go 1.23+
- **架构**: DDD (领域驱动设计) + TDD
- **Web 框架**: Gin
- **数据库**: SQLite (读取 Cursor 数据)
- **服务发现**: mDNS
- **网络通信**: P2P over TCP

## Project Conventions

### 代码规范

- **Go 代码规范**: 参见 [specs/go-style/spec.md](specs/go-style/spec.md)
- **TypeScript 代码规范**: 参见 [specs/typescript-style/spec.md](specs/typescript-style/spec.md)
- **API 规范**: 参见 [specs/api-conventions/spec.md](specs/api-conventions/spec.md)
- **测试规范**: 参见 [specs/testing/spec.md](specs/testing/spec.md)

### Architecture Patterns

#### DDD 分层架构 (Daemon)

```
internal/
├── domain/          # 领域层 - 核心业务逻辑，不依赖任何其他层
├── application/     # 应用层 - 用例编排，依赖 domain
├── infrastructure/  # 基础设施层 - 技术实现，实现 domain 定义的接口
└── interfaces/      # 接口层 - HTTP/P2P 对外暴露，依赖 application
```

#### 依赖方向

- Domain 层不依赖任何其他层
- Application 层依赖 Domain 层
- Infrastructure 层实现 Domain 层定义的接口
- Interfaces 层依赖 Application 层

### Testing Strategy

遵循 TDD（测试驱动开发）原则，详细规范参见 [specs/testing/spec.md](specs/testing/spec.md)。

- **Go 测试**: `testify` 断言 + `mock` 模拟
- **TypeScript 测试**: VS Code Extension Test 框架
- **运行测试**: `make test`（Go）、`npm run test`（TS）
- **覆盖率**: `make test-coverage`

### Git Workflow

- 分支策略：`main` 为主分支，功能开发使用 `feature/<change-id>` 分支
- 提交信息：使用中文，格式 `<类型>: <描述>`
  - 类型：`feat`, `fix`, `docs`, `refactor`, `test`, `chore`
- PR 合并前需通过 `openspec validate --strict`

## Domain Context

### Cursor 数据结构

cocursor 从 Cursor IDE 的 SQLite 数据库读取 AI 聊天记录：

- **数据库位置**: `%APPDATA%/Cursor/User/globalStorage/state.vscdb`
- **对话元数据**: `composerData:{uuid}` - 包含对话状态、模式、消息 ID 列表
- **消息内容**: `bubbleId:{composerId}:{messageId}` - 包含消息类型、文本、Token 统计

### 消息类型

- `type=1`: 用户消息
- `type=2`: AI 回复

### 对话模式

- `chat`: 普通聊天模式
- `agent`: Agent 模式（自动执行工具）

### 团队协作模型

- **团队码**: 6 位字母数字组合，用于标识团队
- **节点发现**: 通过 mDNS 在局域网内广播和发现团队成员
- **数据获取**: 实时从远程节点 HTTP API 获取数据，不做本地同步

## Important Constraints

### 技术约束

- 仅支持局域网内的 P2P 通信（MVP 阶段）
- 数据始终存储在用户本地，不上传到云端
- 读取 Cursor 数据库为只读模式，不修改任何数据

### 安全约束

- 团队码验证：只有相同团队码的节点才能互相访问
- 离线场景：节点离线时无法查看其数据

### 性能约束

- Cursor 数据库可能很大（3GB+），需要优化查询性能
- 对话列表默认只显示最近 100 条

## External Dependencies

### Cursor IDE

- 版本：Cursor 0.44+
- 依赖其本地 SQLite 数据库结构

### Go 依赖

- `github.com/gin-gonic/gin` - HTTP 服务器
- `github.com/swaggo/swag` - Swagger 文档生成
- `modernc.org/sqlite` - SQLite 驱动（纯 Go）
- `github.com/stretchr/testify` - 测试框架

### Node.js 依赖

- `axios` - HTTP 客户端
- `esbuild` - 构建工具

## API 端点概览

HTTP API 端口：`19960`

| 模块     | 端点                             | 说明         |
| -------- | -------------------------------- | ------------ |
| 健康检查 | `GET /health`                    | 服务健康状态 |
| 对话     | `GET /api/v1/chats`              | 对话列表     |
| 对话     | `GET /api/v1/chats/:id`          | 对话详情     |
| 团队     | `GET /api/v1/teams/current`      | 当前团队     |
| 团队     | `POST /api/v1/teams/join`        | 加入团队     |
| 节点     | `GET /api/v1/peers`              | 节点列表     |
| 评论     | `GET /api/v1/chats/:id/comments` | 对话评论     |
| 归档     | `GET /api/v1/archives`           | 归档列表     |
| 统计     | `GET /api/v1/stats/usage`        | 使用统计     |

完整 API 规范参见 [specs/api-conventions/spec.md](specs/api-conventions/spec.md)。
