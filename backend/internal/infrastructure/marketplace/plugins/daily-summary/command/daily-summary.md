---
name: /daily-summary
id: daily-summary
category: Daily Summary
description: Generate and save daily chat summary. Auto-analyze work content, identify work types, extract tech debt.
---

Use the **cocursor** `daily-summary` skill to generate and save daily work summaries.

**Prerequisites:**
- cocursor daemon must be running
- cocursor MCP server must be configured

**Instructions:**
1. Load and use the `daily-summary` skill (check available skills in AGENTS.md)
2. The skill contains complete instructions on:
   - How to query daily sessions (via cocursor MCP tools: `get_daily_sessions`)
   - How to read session content (`get_session_content`)
   - How to analyze and generate summaries
   - How to save summaries to database (`save_daily_summary`)
   - Work type identification
   - Tech debt extraction
   - Deduplication strategies

**Default behavior:**
- Summarizes today's chat records by default
- Specify a different date if needed (format: YYYY-MM-DD)
