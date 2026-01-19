---
name: /daily-summary
id: daily-summary
category: Daily Summary
description: Generate and save daily chat summary. Auto-analyze work content, identify work types, extract tech debt, analyze AI coding efficiency, and identify issues/optimizations.
---

Use the **cocursor** `daily-summary` skill to generate and save daily work summaries.

**Prerequisites:**
- cocursor daemon must be running
- cocursor MCP server must be configured

**Usage:**
```
/daily-summary [date] [--with-git] [--no-git]
```

**Parameters:**
- `date` (optional): Date in YYYY-MM-DD format, defaults to today
- `--with-git` (optional): Enable Git commit analysis (analyzes git commits for active projects)
- `--no-git` (optional): Disable Git commit analysis (default: ask user)

**Instructions:**
1. Load and use the `daily-summary` skill (check available skills in AGENTS.md)
2. **Decision Point**: If `--with-git` or `--no-git` is not specified, ask the user:
   - "是否需要包含 Git commit 分析？这将分析当天有提交的项目，补充会话记录数据。(y/n)"
   - Based on user response, enable or disable Git analysis
3. The skill contains complete instructions on:
   - How to query daily sessions (via cocursor MCP tools: `get_daily_conversations` - recommended)
   - How to analyze conversation content
   - How to analyze AI coding efficiency from conversations
   - How to identify issues and optimizations
   - How to generate and save summaries
   - Work type identification
   - Tech debt extraction
   - Deduplication strategies

**Default behavior:**
- Summarizes today's chat records by default
- Git analysis: Ask user if not specified
- Includes AI coding efficiency analysis
- Includes issues and optimizations analysis
