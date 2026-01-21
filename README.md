# CoCursor

<p align="center">
  <img src="co-extension/resources/icon.png" alt="CoCursor Logo" width="128" height="128">
</p>

<p align="center">
  <strong>Team AI Collaboration Tool for Cursor IDE</strong>
</p>

<p align="center">
  <a href="./README_CN.md">中文文档</a> •
  <a href="https://github.com/toheart/cocursor/releases">Releases</a> •
  <a href="#installation">Installation</a> •
  <a href="#features">Features</a>
</p>

---

> The efficiency gap between those who can use AI and those who can't is 100x. This is not an exaggeration.

## What is CoCursor?

**CoCursor** is a VS Code/Cursor extension that empowers teams to collaborate with AI more effectively. It combines work analytics, semantic search of AI conversations, skill sharing marketplace, and automated reporting - all running locally with complete data privacy.

Built with:
- **Backend**: Go 1.24 + Gin + DDD Architecture
- **Frontend**: VS Code Extension + React + TypeScript
- **Team Collaboration**: P2P Architecture + mDNS Discovery + WebSocket Real-time Sync
- **RAG**: Qdrant Vector Database + Embedding Models (supports local deployment)
- **Workflow**: OpenSpec-driven Development

## Features

### 📊 Work Analysis Dashboard

Track every AI collaboration session automatically.

- Monitor your work sessions in Cursor
- Analyze work types, tech stacks, and code changes
- **One-click daily/weekly report generation**

No more spending 30 minutes writing work reports. AI helps you work and helps you report.

### 🔍 AI Conversation Semantic Search (RAG)

Every question, code snippet, and solution you've discussed with AI is in your Cursor chat history.

CoCursor's RAG features:
- Automatically index all your conversations in Cursor
- Semantic search: find historical conversations using natural language
- "How did I solve that database connection issue last week?" → Found instantly

**Your AI conversations are no longer one-time use - they become searchable, reusable knowledge.**

### 🤝 Team Skill Marketplace

One person knowing AI isn't enough - the whole team needs to know.

- Publish your AI Skills to the team with one click
- Team members can install instantly and gain the same capabilities
- **P2P LAN direct transfer, no server involved, data stays secure**

Let the weakest member on the team use the strongest member's AI skills.

### ⚡ Workflow Engine

Drive AI workflows with OpenSpec specifications:

- Requirements → Design → Implementation, standardized end-to-end
- Not "what do you think we should do" but "everyone follows this process"
- AI executes according to specs, results are predictable

## Installation

### From VS Code Marketplace

Search for "CoCursor" in the VS Code/Cursor Extensions marketplace and install.

### From GitHub Releases

1. Download the VSIX file for your platform from [Releases](https://github.com/toheart/cocursor/releases):
   - `cocursor-linux-x64.vsix` - Linux x64
   - `cocursor-win32-x64.vsix` - Windows x64
   - `cocursor-darwin-x64.vsix` - macOS Intel
   - `cocursor-darwin-arm64.vsix` - macOS Apple Silicon

2. Install in VS Code/Cursor:
   ```bash
   code --install-extension cocursor-<platform>.vsix
   ```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/toheart/cocursor.git
cd cocursor

# Build backend (requires Go 1.24+)
cd backend
make build-all

# Build frontend extension (requires Node.js 18+)
cd ../co-extension
npm install
make build

# Package as VSIX
npx @vscode/vsce package
```

## Quick Start

1. **Open CoCursor Panel**: Click the CoCursor icon in the VS Code/Cursor sidebar
2. **Work Analysis**: View your AI collaboration statistics and generate reports
3. **RAG Search**: Search through your historical AI conversations
4. **Team Collaboration**: Create or join a team to share skills

## Architecture

```
cocursor/
├── backend/                 # Go Backend Daemon (DDD Architecture)
│   ├── cmd/                 # Application entry points
│   ├── internal/
│   │   ├── domain/          # Domain models and business logic
│   │   ├── application/     # Application services
│   │   ├── infrastructure/  # External integrations
│   │   └── interfaces/      # HTTP handlers
│   └── pkg/                 # Shared packages
├── co-extension/            # VS Code Extension (React + TypeScript)
│   ├── src/
│   │   ├── extension.ts     # Extension entry point
│   │   ├── webview/         # React UI components
│   │   └── daemon/          # Daemon process manager
│   └── resources/           # Static assets
└── openspec/                # OpenSpec specifications
```

## Privacy & Security

- **100% Local Execution**: All data processing happens on your machine
- **No Cloud Services**: Your code and conversations never leave your computer
- **P2P Team Collaboration**: Direct peer-to-peer communication within your LAN
- **Open Source**: Fully auditable codebase

## Roadmap

| Phase | Capability | Value |
|-------|------------|-------|
| **Now** | Personal conversation search | Personal knowledge retention |
| **Next** | MCP Integration | Connect more data sources |
| **Future** | Team Brain | Aggregate all team members' AI conversations into a team knowledge base |

Imagine: A new team member doesn't need to ask veterans - just search the Team Brain: "What pitfalls did we encounter with this module?" - Everyone's experience is right there.

## Contributing

Contributions are welcome! Please read our contributing guidelines before submitting PRs.

## License

[CoCursor Non-Commercial License](co-extension/LICENSE) - Free for non-commercial use only.

## Links

- **GitHub**: https://github.com/toheart/cocursor
- **VS Code Marketplace**: https://marketplace.visualstudio.com/items?itemName=tanglyan-cocursor.cocursor

---

*If you're also leading a team and thinking about how to help your team use AI better - let's connect!*
