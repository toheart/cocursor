---
name: weekly-summary
description: Generate weekly work reports by aggregating daily summaries. Use when users request weekly reports, weekly summaries, or need to review accomplishments over a week period. Focuses on key accomplishments and progress suitable for upward reporting.
---

# Weekly Summary Skill

> **Requires**: cocursor daemon running at `localhost:19960`

## Pre-check

```bash
curl -s http://localhost:19960/health
# Expected: {"status":"ok"}
```

## Workflow

### Step 1: Determine Week Range

Parse user input:
- "last week" → Last Monday to last Sunday
- "this week" → This Monday to today
- "2024-01-08 ~ 2024-01-14" → Specified range
- No specification → Current week

Week definition: Monday to Sunday (ISO standard)

### Step 2: Fetch Daily Summaries

```bash
# macOS / Linux
curl -s "http://localhost:19960/api/v1/daily-summary/range?start_date=2024-01-15&end_date=2024-01-21"

# Windows PowerShell
Invoke-RestMethod -Uri "http://localhost:19960/api/v1/daily-summary/range?start_date=2024-01-15&end_date=2024-01-21"
```

### Step 3: Aggregate and Analyze

Aggregation dimensions:
- Overall statistics (sessions, code changes, active time)
- Project-level summary
- Work category distribution
- Key accomplishments extraction

See [references/aggregation-rules.md](references/aggregation-rules.md) for details.

### Step 4: Generate Weekly Report

Core principles:
- **Highlight accomplishments**: Use bullet points
- **Quantify progress**: Code lines, session counts
- **Concise and impactful**: Suitable for upward reporting

See [references/report-template.md](references/report-template.md) for structure.

### Step 5: Save Weekly Report

```bash
# macOS / Linux
curl -s -X POST "http://localhost:19960/api/v1/weekly-summary" \
  -H "Content-Type: application/json" \
  -d '{"week_start":"2024-01-15","week_end":"2024-01-21","summary":"...","total_sessions":25,"working_days":5}'

# Windows PowerShell
$body = @{week_start="2024-01-15"; week_end="2024-01-21"; summary="..."; total_sessions=25; working_days=5} | ConvertTo-Json -Depth 10
Invoke-RestMethod -Uri "http://localhost:19960/api/v1/weekly-summary" -Method POST -ContentType "application/json" -Body $body
```

**Parameters:**
- `week_start`: Monday date (YYYY-MM-DD)
- `week_end`: Sunday date (YYYY-MM-DD)
- `summary`: Markdown content
- `language`: "zh" or "en"
- `total_sessions`: Total session count
- `working_days`: Days with data
- `categories`: Work category statistics (JSON object)
- `key_accomplishments`: Key accomplishments list (JSON array)

### Step 6: Check Existing Report (Optional)

```bash
curl -s "http://localhost:19960/api/v1/weekly-summary?week_start=2024-01-15"
```

Response includes `needs_update` flag indicating if source data changed.

## Reference Files

- [references/aggregation-rules.md](references/aggregation-rules.md) - Data aggregation rules
- [references/report-template.md](references/report-template.md) - Report structure
- [references/report-examples.md](references/report-examples.md) - Example reports
- [references/data-format.md](references/data-format.md) - Data structures

## API Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/daily-summary/range` | GET | Batch fetch daily summaries |
| `/api/v1/weekly-summary` | GET | Query weekly report |
| `/api/v1/weekly-summary` | POST | Save weekly report |
