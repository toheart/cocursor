# CoCursor

<p align="center">
  <img src="https://raw.githubusercontent.com/toheart/cocursor/main/co-extension/resources/icon.png" alt="CoCursor Logo" width="128" height="128">
</p>

<p align="center">
  <strong>Team AI Collaboration Tool for Cursor IDE</strong>
</p>

<p align="center">
  <a href="https://github.com/toheart/cocursor">GitHub</a> •
  <a href="https://github.com/toheart/cocursor/blob/main/README_CN.md">中文文档</a>
</p>

---

> The efficiency gap between those who can use AI and those who can't is 100x.

CoCursor helps teams collaborate with AI more effectively. Track your work, search past conversations, share skills with teammates, and generate reports automatically — all running locally with complete data privacy.

## Features

### 📊 Work Analysis Dashboard

Track every AI collaboration session automatically.

- **Session Tracking**: Monitor your work sessions in Cursor with detailed statistics
- **Code Change Analytics**: Track lines added/removed, files changed, and token usage
- **Time Distribution**: Visualize your productivity patterns with heatmaps
- **One-Click Reports**: Generate daily/weekly work reports instantly

No more spending 30 minutes writing status updates. AI helps you work and helps you report.

### 🔍 AI Conversation Semantic Search (RAG)

Every question, code snippet, and solution you've discussed with AI is in your Cursor chat history.

- **Automatic Indexing**: Index all your Cursor conversations locally
- **Semantic Search**: Find historical conversations using natural language
- **Knowledge Retrieval**: "How did I solve that database connection issue?" → Found instantly
- **Project Filtering**: Search within specific projects

Your AI conversations are no longer one-time use — they become searchable, reusable knowledge.

### 🛒 Skill Marketplace

One person knowing AI isn't enough — the whole team needs to know.

- **Browse Skills**: Discover productivity-boosting AI skills
- **One-Click Install**: Install skills directly to your Cursor configuration
- **Team Sharing**: Share custom skills with your team via P2P
- **Built-in Collection**: Curated skills for common development tasks

Let the newest member use the most experienced member's AI skills.

### 👥 Team Collaboration

Collaborate with your team in real-time.

- **P2P Architecture**: Direct peer-to-peer communication within your LAN
- **Code Sharing**: Share code snippets with team members instantly
- **Daily Reports**: View team members' work summaries
- **Skill Publishing**: Publish your AI skills to the team marketplace

No server involved — data stays secure within your network.

### ⚡ Daily Summary Reminder

Never forget to summarize your work.

- **Smart Reminders**: Get notified before leaving work
- **Morning Follow-up**: Reminder next morning if you missed yesterday
- **Configurable Times**: Set your preferred reminder schedule

## Commands

| Command | Description |
|---------|-------------|
| `CoCursor: Open Dashboard` | Open work analysis dashboard |
| `CoCursor: Open Sessions` | View recent AI conversation sessions |
| `CoCursor: Open Marketplace` | Browse and install AI skills |
| `CoCursor: Share Code to Team` | Share selected code with team members |
| `CoCursor: Toggle Status Sharing` | Enable/disable work status sharing |

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

1. Open CoCursor sidebar → RAG Search → Settings
2. Configure embedding model (supports local models)
3. Set up Qdrant vector database (can run locally)
4. Start indexing your conversations

## Privacy & Security

- **100% Local Execution**: All data processing happens on your machine
- **No Cloud Services**: Your code and conversations never leave your computer
- **P2P Team Collaboration**: Direct peer-to-peer communication within your LAN
- **Open Source**: Fully auditable codebase

## Requirements

- VS Code 1.80.0 or higher / Cursor IDE
- macOS, Windows, or Linux

## Links

- **GitHub**: https://github.com/toheart/cocursor
- **Issues**: https://github.com/toheart/cocursor/issues
- **Releases**: https://github.com/toheart/cocursor/releases

## License

[CoCursor Non-Commercial License](https://github.com/toheart/cocursor/blob/main/co-extension/LICENSE) - Free for non-commercial use only.
