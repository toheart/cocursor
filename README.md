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
  <a href="https://marketplace.visualstudio.com/items?itemName=tanglyan-cocursor.cocursor">VS Code Marketplace</a>
</p>

---

> The efficiency gap between those who can use AI and those who can't is 100x. This is not an exaggeration.

## What is CoCursor?

**CoCursor** is a VS Code/Cursor extension that empowers teams to collaborate with AI more effectively. It combines work analytics, semantic search of AI conversations, skill sharing marketplace, and automated reporting — all running locally with complete data privacy.

**Tech Stack:**
- **Backend**: Go 1.24 + Gin + DDD Architecture
- **Frontend**: VS Code Extension + React + TypeScript
- **Team Collaboration**: P2P Architecture + mDNS Discovery + WebSocket Real-time Sync
- **RAG**: Qdrant Vector Database + Embedding Models (supports local deployment)
- **Workflow**: OpenSpec-driven Development

## Features

### 📊 Work Analysis Dashboard

Track every AI collaboration session automatically.

| Feature | Description |
|---------|-------------|
| **Session Tracking** | Monitor your work sessions in Cursor with detailed statistics |
| **Code Analytics** | Track lines added/removed, files changed, token usage trends |
| **Time Heatmap** | Visualize when you're most productive |
| **Top Files** | See which files you work on most with AI |
| **One-Click Reports** | Generate daily/weekly work reports instantly |

No more spending 30 minutes writing work reports. AI helps you work and helps you report.

### 🔍 AI Conversation Semantic Search (RAG)

Every question, code snippet, and solution you've discussed with AI is in your Cursor chat history.

| Feature | Description |
|---------|-------------|
| **Automatic Indexing** | Index all your Cursor conversations locally |
| **Semantic Search** | Find conversations using natural language, not keywords |
| **Knowledge Retrieval** | "How did I solve that database issue?" → Found instantly |
| **Project Filtering** | Search within specific projects |
| **Context Preview** | See relevant context before opening full conversation |

**Your AI conversations are no longer one-time use — they become searchable, reusable knowledge.**

### 🛒 Skill Marketplace

One person knowing AI isn't enough — the whole team needs to know.

| Feature | Description |
|---------|-------------|
| **Browse Skills** | Discover productivity-boosting AI skills |
| **One-Click Install** | Install skills directly to your Cursor configuration |
| **Category Filters** | Find skills by category (productivity, creative, tools, etc.) |
| **Source Filters** | View built-in skills or team-shared skills |
| **Team Publishing** | Share your custom skills with teammates |

Let the weakest member on the team use the strongest member's AI skills.

### 👥 Team Collaboration

Collaborate with your team in real-time, completely within your LAN.

| Feature | Description |
|---------|-------------|
| **P2P Discovery** | Auto-discover team members via mDNS |
| **Code Sharing** | Right-click to share selected code with team |
| **Daily Reports** | View team members' work summaries |
| **Weekly Calendar** | See team activity at a glance |
| **Member Stats** | Track team productivity metrics |

**P2P LAN direct transfer — no server involved, data stays secure.**

### ⚡ Workflow Engine

Drive AI workflows with OpenSpec specifications:

- Requirements → Design → Implementation, standardized end-to-end
- Not "what do you think we should do" but "everyone follows this process"
- AI executes according to specs, results are predictable

### 🔔 Daily Summary Reminder

Never forget to summarize your work.

| Setting | Default | Description |
|---------|---------|-------------|
| Evening Reminder | 17:50 | Get notified before leaving work |
| Morning Follow-up | 09:00 | Reminder next morning if you missed yesterday |
| Enable/Disable | Off | Toggle reminders in settings |

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
3. **RAG Search**: Search through your historical AI conversations (requires setup)
4. **Skill Marketplace**: Browse and install productivity-boosting AI skills
5. **Team Collaboration**: Create or join a team to share skills and code

## Commands

| Command | Description |
|---------|-------------|
| `CoCursor: Open Dashboard` | Open work analysis dashboard |
| `CoCursor: Open Sessions` | View recent AI conversation sessions |
| `CoCursor: Open Marketplace` | Browse and install AI skills |
| `CoCursor: Share Code to Team` | Share selected code with team members |
| `CoCursor: Toggle Status Sharing` | Enable/disable work status sharing |
| `CoCursor: Refresh Webview` | Refresh the CoCursor panel data |

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `cocursor.autoStartServer` | `true` | Auto-start the backend daemon |
| `cocursor.daemon.port` | `19960` | Backend server port |
| `cocursor.reminder.enabled` | `false` | Enable daily summary reminders |
| `cocursor.reminder.eveningTime` | `17:50` | Evening reminder time (HH:mm) |
| `cocursor.reminder.morningTime` | `09:00` | Morning follow-up time (HH:mm) |

## RAG Setup (Optional)

To enable semantic search of your AI conversations:

1. Open CoCursor sidebar → RAG Search → Settings (gear icon)
2. Configure embedding model (supports OpenAI, local models via Ollama)
3. Set up Qdrant vector database (can run locally via Docker)
4. Click "Start Indexing" to index your conversations

**Recommended Setup:**
```bash
# Run Qdrant locally
docker run -p 6333:6333 qdrant/qdrant
```

## Architecture

```
cocursor/
├── backend/                 # Go Backend Daemon (DDD Architecture)
│   ├── cmd/                 # Application entry points
│   ├── internal/
│   │   ├── domain/          # Domain models and business logic
│   │   ├── application/     # Application services
│   │   ├── infrastructure/  # External integrations (Qdrant, SQLite, etc.)
│   │   └── interfaces/      # HTTP handlers, MCP tools
│   └── pkg/                 # Shared packages
├── co-extension/            # VS Code Extension (React + TypeScript)
│   ├── src/
│   │   ├── extension.ts     # Extension entry point
│   │   ├── webview/         # React UI components
│   │   │   ├── components/  # WorkAnalysis, RAGSearch, Marketplace, Team...
│   │   │   ├── services/    # API service layer
│   │   │   └── hooks/       # React hooks
│   │   └── daemon/          # Daemon process manager
│   └── resources/           # Static assets
└── openspec/                # OpenSpec specifications
```

## Privacy & Security

- **100% Local Execution**: All data processing happens on your machine
- **No Cloud Services**: Your code and conversations never leave your computer
- **P2P Team Collaboration**: Direct peer-to-peer communication within your LAN
- **Open Source**: Fully auditable codebase
- **No Telemetry**: We don't collect any usage data

## Roadmap

| Phase | Capability | Value |
|-------|------------|-------|
| **Now** | Personal conversation search | Personal knowledge retention |
| **Next** | MCP Integration | Connect more data sources |
| **Future** | Team Brain | Aggregate all team members' AI conversations into a team knowledge base |

Imagine: A new team member doesn't need to ask veterans — just search the Team Brain: "What pitfalls did we encounter with this module?" — Everyone's experience is right there.

**When every AI conversation becomes searchable knowledge, "knowledge lost when people leave" is solved forever.**

## Contributing

Contributions are welcome! Please read our contributing guidelines before submitting PRs.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

[CoCursor Non-Commercial License](co-extension/LICENSE) - Free for non-commercial use only.

## Links

- **GitHub**: https://github.com/toheart/cocursor
- **VS Code Marketplace**: https://marketplace.visualstudio.com/items?itemName=tanglyan-cocursor.cocursor
- **Issues**: https://github.com/toheart/cocursor/issues

---

*If you're also leading a team and thinking about how to help your team use AI better — let's connect!*
