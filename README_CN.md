# CoCursor

<p align="center">
  <img src="co-extension/resources/icon.png" alt="CoCursor Logo" width="128" height="128">
</p>

<p align="center">
  <strong>面向 Cursor IDE 的团队 AI 协作工具</strong>
</p>

<p align="center">
  <a href="./README.md">English</a> •
  <a href="https://github.com/toheart/cocursor/releases">发布版本</a> •
  <a href="https://marketplace.visualstudio.com/items?itemName=tanglyan-cocursor.cocursor">VS Code 市场</a>
</p>

---

> 会用 AI 的人和不会用的人，效率差距是 100 倍。这不是夸张。

## CoCursor 是什么？

**CoCursor** 是一个 VS Code/Cursor 插件，帮助团队更高效地与 AI 协作。它集成了工作分析、AI 对话语义搜索、技能共享市场和自动化报告功能——所有功能都在本地运行，确保数据完全私密。

**技术栈：**
- **后端**：Go 1.24 + Gin + DDD 架构
- **前端**：VS Code Extension + React + TypeScript
- **团队协作**：P2P 架构 + mDNS 发现 + WebSocket 实时同步
- **RAG**：Qdrant 向量数据库 + 嵌入模型（支持本地部署）
- **工作流**：OpenSpec 驱动开发

## 功能特性

### 📊 工作分析仪表板

自动追踪你和 AI 的每一次协作。

| 功能 | 说明 |
|------|------|
| **会话追踪** | 监控你在 Cursor 中的工作会话，提供详细统计 |
| **代码分析** | 追踪代码增删行数、文件变更、Token 使用趋势 |
| **时间热力图** | 可视化展示你的高效工作时段 |
| **热门文件** | 查看你最常与 AI 协作的文件 |
| **一键报告** | 即时生成日报/周报 |

再也不用花 30 分钟写工作汇报了。AI 帮你干活，也帮你汇报。

### 🔍 AI 对话语义搜索（RAG）

你和 AI 聊过的每一个问题、每一段代码、每一个解决方案——都在 Cursor 的聊天记录里。

| 功能 | 说明 |
|------|------|
| **自动索引** | 在本地自动索引所有 Cursor 对话 |
| **语义搜索** | 用自然语言搜索，而不是关键词 |
| **知识检索** | "上次那个数据库问题怎么解决的？" → 直接找到 |
| **项目过滤** | 在特定项目内搜索 |
| **上下文预览** | 打开完整对话前预览相关上下文 |

**你的 AI 对话不再是一次性的，而是可以被检索、被复用的知识。**

### 🛒 技能市场

一个人会用 AI 不够，要让整个团队都会用。

| 功能 | 说明 |
|------|------|
| **浏览技能** | 发现提升效率的 AI 技能 |
| **一键安装** | 直接安装到你的 Cursor 配置 |
| **分类筛选** | 按类别查找技能（效率、创意、工具等） |
| **来源筛选** | 查看内置技能或团队共享技能 |
| **团队发布** | 将你的自定义技能分享给队友 |

让团队里最弱的人，也能用上最强者的 AI 技能。

### 👥 团队协作

与团队实时协作，完全在局域网内完成。

| 功能 | 说明 |
|------|------|
| **P2P 发现** | 通过 mDNS 自动发现团队成员 |
| **代码分享** | 右键选中代码，一键分享给团队 |
| **每日报告** | 查看团队成员的工作总结 |
| **周历视图** | 一览团队活动情况 |
| **成员统计** | 追踪团队生产力指标 |

**P2P 局域网直传——不经过任何服务器，数据安全。**

### ⚡ 工作流引擎

用 OpenSpec 规范驱动 AI 工作流：

- 需求 → 设计方案 → 代码实现，全流程标准化
- 不是"你觉得怎么做"，而是"团队都按这个流程做"
- AI 按规范执行，结果可预期

### 🔔 每日总结提醒

再也不会忘记总结工作。

| 设置 | 默认值 | 说明 |
|------|--------|------|
| 下班提醒 | 17:50 | 下班前收到提醒 |
| 次日补充 | 09:00 | 如果昨天漏了，第二天早上提醒 |
| 开启/关闭 | 关闭 | 在设置中切换 |

## 安装

### 从 VS Code 市场安装

在 VS Code/Cursor 扩展市场搜索 "CoCursor" 并安装。

### 从 GitHub Releases 安装

1. 从 [Releases](https://github.com/toheart/cocursor/releases) 下载对应平台的 VSIX 文件：
   - `cocursor-linux-x64.vsix` - Linux x64
   - `cocursor-win32-x64.vsix` - Windows x64
   - `cocursor-darwin-x64.vsix` - macOS Intel
   - `cocursor-darwin-arm64.vsix` - macOS Apple Silicon

2. 在 VS Code/Cursor 中安装：
   ```bash
   code --install-extension cocursor-<platform>.vsix
   ```

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/toheart/cocursor.git
cd cocursor

# 构建后端（需要 Go 1.24+）
cd backend
make build-all

# 构建前端扩展（需要 Node.js 18+）
cd ../co-extension
npm install
make build

# 打包成 VSIX
npx @vscode/vsce package
```

## 快速开始

1. **打开 CoCursor 面板**：点击 VS Code/Cursor 侧边栏的 CoCursor 图标
2. **工作分析**：查看你的 AI 协作统计并生成报告
3. **RAG 搜索**：搜索历史 AI 对话（需要配置）
4. **技能市场**：浏览并安装提升效率的 AI 技能
5. **团队协作**：创建或加入团队以共享技能和代码

## 命令

| 命令 | 说明 |
|------|------|
| `CoCursor: 打开仪表板` | 打开工作分析仪表板 |
| `CoCursor: 打开会话列表` | 查看最近的 AI 对话会话 |
| `CoCursor: 打开技能市场` | 浏览并安装 AI 技能 |
| `CoCursor: 分享代码到团队` | 将选中的代码分享给团队成员 |
| `CoCursor: 切换工作状态分享` | 开启/关闭工作状态分享 |
| `CoCursor: 刷新 Webview 数据` | 刷新 CoCursor 面板数据 |

## 配置

| 设置 | 默认值 | 说明 |
|------|--------|------|
| `cocursor.autoStartServer` | `true` | 自动启动后端服务 |
| `cocursor.daemon.port` | `19960` | 后端服务端口 |
| `cocursor.reminder.enabled` | `false` | 启用每日总结提醒 |
| `cocursor.reminder.eveningTime` | `17:50` | 下班提醒时间（HH:mm） |
| `cocursor.reminder.morningTime` | `09:00` | 次日补充提醒时间（HH:mm） |

## RAG 配置（可选）

启用 AI 对话语义搜索：

1. 打开 CoCursor 侧边栏 → RAG 搜索 → 设置（齿轮图标）
2. 配置嵌入模型（支持 OpenAI、本地模型如 Ollama）
3. 设置 Qdrant 向量数据库（可通过 Docker 本地运行）
4. 点击"开始索引"索引你的对话

**推荐配置：**
```bash
# 本地运行 Qdrant
docker run -p 6333:6333 qdrant/qdrant
```

## 架构

```
cocursor/
├── backend/                 # Go 后端 Daemon（DDD 架构）
│   ├── cmd/                 # 应用入口
│   ├── internal/
│   │   ├── domain/          # 领域模型和业务逻辑
│   │   ├── application/     # 应用服务
│   │   ├── infrastructure/  # 外部集成（Qdrant、SQLite 等）
│   │   └── interfaces/      # HTTP 处理器、MCP 工具
│   └── pkg/                 # 共享包
├── co-extension/            # VS Code 扩展（React + TypeScript）
│   ├── src/
│   │   ├── extension.ts     # 扩展入口
│   │   ├── webview/         # React UI 组件
│   │   │   ├── components/  # WorkAnalysis、RAGSearch、Marketplace、Team...
│   │   │   ├── services/    # API 服务层
│   │   │   └── hooks/       # React hooks
│   │   └── daemon/          # Daemon 进程管理
│   └── resources/           # 静态资源
└── openspec/                # OpenSpec 规范
```

## 隐私与安全

- **100% 本地运行**：所有数据处理都在你的机器上完成
- **无云服务**：你的代码和对话永远不会离开你的电脑
- **P2P 团队协作**：局域网内点对点直接通信
- **开源**：完全可审计的代码库
- **无遥测**：我们不收集任何使用数据

## 路线图

| 阶段 | 能力 | 价值 |
|------|------|------|
| **现在** | 个人历史对话搜索 | 个人知识不丢失 |
| **下一步** | MCP 集成 | 打通更多数据源 |
| **终局** | 团队大脑 | 同一项目所有人的 AI 对话，汇聚成团队知识库 |

想象一下：新人入职，不用问老员工，直接搜索团队大脑——"这个模块之前踩过什么坑？"——所有人的经验都在这里。

**当团队的每一次 AI 对话都变成可检索的知识，"人走知识丢"这件事就彻底解决了。**

## 贡献

欢迎贡献！请在提交 PR 之前阅读我们的贡献指南。

1. Fork 仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 发起 Pull Request

## 许可证

[CoCursor 非商业许可证](co-extension/LICENSE) - 仅限非商业用途免费使用。

## 链接

- **GitHub**：https://github.com/toheart/cocursor
- **VS Code 市场**：https://marketplace.visualstudio.com/items?itemName=tanglyan-cocursor.cocursor
- **问题反馈**：https://github.com/toheart/cocursor/issues

---

*如果你也在带团队，也在思考怎么让团队用好 AI——欢迎交流！*

欢迎关注作者：**小唐的技术日志**
