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
  <a href="#安装">安装</a> •
  <a href="#功能特性">功能特性</a>
</p>

---

> 会用 AI 的人和不会用的人，效率差距是 100 倍。这不是夸张。

## CoCursor 是什么？

**CoCursor** 是一个 VS Code/Cursor 插件，帮助团队更高效地与 AI 协作。它集成了工作分析、AI 对话语义搜索、技能共享市场和自动化报告功能——所有功能都在本地运行，确保数据完全私密。

技术栈：
- **后端**：Go 1.24 + Gin + DDD 架构
- **前端**：VS Code Extension + React + TypeScript
- **团队协作**：P2P 架构 + mDNS 发现 + WebSocket 实时同步
- **RAG**：Qdrant 向量数据库 + 嵌入模型（支持本地部署）
- **工作流**：OpenSpec 驱动开发

## 功能特性

### 📊 工作分析仪表板

自动追踪你和 AI 的每一次协作。

- 追踪你在 Cursor 中的工作会话
- 分析工作类型、技术栈、代码变更
- **一键生成日报、周报**

再也不用花 30 分钟写工作汇报了。AI 帮你干活，也帮你汇报。

### 🔍 AI 对话语义搜索（RAG）

你和 AI 聊过的每一个问题、每一段代码、每一个解决方案——都在 Cursor 的聊天记录里。

CoCursor 的 RAG 功能：
- 自动索引你在 Cursor 中的所有对话
- 语义搜索：用自然语言找到历史对话
- "上次那个数据库连接问题怎么解决的？" → 直接找到

**你的 AI 对话不再是一次性的，而是可以被检索、被复用的知识。**

### 🤝 团队技能市场

一个人会用 AI 不够，要让整个团队都会用。

- 把你写的 AI Skills 一键发布给团队
- 团队成员一键安装，立刻拥有同样能力
- **P2P 局域网直传，不经过任何服务器，数据安全**

让团队里最弱的人，也能用上最强者的 AI 技能。

### ⚡ 工作流引擎

用 OpenSpec 规范驱动 AI 工作流：

- 需求 → 设计方案 → 代码实现，全流程标准化
- 不是"你觉得怎么做"，而是"团队都按这个流程做"
- AI 按规范执行，结果可预期

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
3. **RAG 搜索**：搜索历史 AI 对话
4. **团队协作**：创建或加入团队以共享技能

## 架构

```
cocursor/
├── backend/                 # Go 后端 Daemon（DDD 架构）
│   ├── cmd/                 # 应用入口
│   ├── internal/
│   │   ├── domain/          # 领域模型和业务逻辑
│   │   ├── application/     # 应用服务
│   │   ├── infrastructure/  # 外部集成
│   │   └── interfaces/      # HTTP 处理器
│   └── pkg/                 # 共享包
├── co-extension/            # VS Code 扩展（React + TypeScript）
│   ├── src/
│   │   ├── extension.ts     # 扩展入口
│   │   ├── webview/         # React UI 组件
│   │   └── daemon/          # Daemon 进程管理
│   └── resources/           # 静态资源
└── openspec/                # OpenSpec 规范
```

## 隐私与安全

- **100% 本地运行**：所有数据处理都在你的机器上完成
- **无云服务**：你的代码和对话永远不会离开你的电脑
- **P2P 团队协作**：局域网内点对点直接通信
- **开源**：完全可审计的代码库

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

## 许可证

[CoCursor 非商业许可证](co-extension/LICENSE) - 仅限非商业用途免费使用。

## 链接

- **GitHub**：https://github.com/toheart/cocursor
- **VS Code Marketplace**：https://marketplace.visualstudio.com/items?itemName=tanglyan-cocursor.cocursor
- 欢迎关注作者：小唐的技术日志
---

*如果你也在带团队，也在思考怎么让团队用好 AI——欢迎交流！*
