---
name: daily-summary
description: Generate daily work reports from Cursor chat history. Use when users request work summaries, daily reports, or need to review daily work content. Supports querying sessions by date, reading conversations, identifying work types, extracting tech debt, and saving summaries. Can optionally integrate git commit analysis.
---

# Daily Summary Skill

> **Requires**: cocursor daemon running at `localhost:19960`

## Pre-check

Verify daemon is running:

```bash
# macOS / Linux
curl -s http://localhost:19960/health

# Windows CMD
curl.exe -s http://localhost:19960/health

# Windows PowerShell
Invoke-RestMethod -Uri "http://localhost:19960/health"
```

Expected: `{"status":"ok"}`

## Workflow

### Step 1: Get Conversations

**Recommended**: Get all conversations at once.

```bash
# macOS / Linux
curl -s "http://localhost:19960/api/v1/sessions/conversations?date=2024-01-22"

# Windows CMD
curl.exe -s "http://localhost:19960/api/v1/sessions/conversations?date=2024-01-22"

# Windows PowerShell
Invoke-RestMethod -Uri "http://localhost:19960/api/v1/sessions/conversations?date=2024-01-22"
```

**Alternative**: Get session list first, then individual content.

```bash
# Get session list
curl -s "http://localhost:19960/api/v1/sessions/daily?date=2024-01-22"

# Get single session content
curl -s "http://localhost:19960/api/v1/sessions/{session_id}/content"
```

### Step 2: Analyze and Generate Summary

Generate Markdown report including:
- Work type identification
- Code change statistics
- Time distribution
- AI coding efficiency
- Tech debt extraction

See [references/summary-examples.md](references/summary-examples.md) for report structure.

### Step 3: Save Summary

```bash
# macOS / Linux
curl -s -X POST "http://localhost:19960/api/v1/daily-summary" \
  -H "Content-Type: application/json" \
  -d '{"date":"2024-01-22","summary":"...","total_sessions":5,"categories":{"coding":8,"problem_solving":3}}'

# Windows PowerShell
$body = @{date="2024-01-22"; summary="..."; total_sessions=5; categories=@{coding=8; problem_solving=3}} | ConvertTo-Json -Depth 10
Invoke-RestMethod -Uri "http://localhost:19960/api/v1/daily-summary" -Method POST -ContentType "application/json" -Body $body
```

**Parameters:**
- `date`: YYYY-MM-DD format
- `summary`: Markdown content
- `language`: "zh" or "en"
- `total_sessions`: Session count
- `categories`: Work category statistics (**must be JSON object, not string**)

### Step 4: Query Saved Summary (Optional)

```bash
curl -s "http://localhost:19960/api/v1/daily-summary?date=2024-01-22"
```

## Git Commit Analysis (Optional)

Ask user: "Include Git commit analysis? (y/n)"

If enabled, see [references/git-analysis.md](references/git-analysis.md).

## Reference Files

- [references/work-categories.md](references/work-categories.md) - Work type identification rules
- [references/tech-debt-patterns.md](references/tech-debt-patterns.md) - Tech debt extraction
- [references/ai-efficiency-analysis.md](references/ai-efficiency-analysis.md) - AI efficiency metrics
- [references/data-format.md](references/data-format.md) - Data structure definitions
- [references/summary-examples.md](references/summary-examples.md) - Report examples

## API Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/sessions/conversations` | GET | Get all conversations for date |
| `/api/v1/sessions/daily` | GET | Get session list for date |
| `/api/v1/sessions/:id/content` | GET | Get single session content |
| `/api/v1/daily-summary` | GET | Query saved summary |
| `/api/v1/daily-summary` | POST | Save summary |
| `/api/v1/daily-summary/range` | GET | Batch query summaries |
