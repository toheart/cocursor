---
name: weekly-summary
description: Aggregate daily summaries to generate comprehensive weekly work reports focused on accomplishments and progress. Supports batch querying daily summaries for a week range, analyzing productivity trends, and saving weekly reports. Use this skill when users request weekly work summaries, weekly reports, or need to review work accomplishments over a week period.
---

# Weekly Summary Skill

> **MCP Server Dependency**: This skill requires the `cocursor` MCP server.
> 
> Available tools (use full names when calling):
> - `mcp__cocursor__get_daily_summaries_range` - Batch fetch daily summaries within date range
> - `mcp__cocursor__save_weekly_summary` - Save weekly report
> - `mcp__cocursor__get_weekly_summary` - Query saved weekly report with idempotency check

Aggregate daily summaries to generate weekly work reports focused on **accomplishments and progress**.

## Workflow

### Step 1: Determine Week Range

**Parse user input:**
- "last week" → Last Monday to last Sunday
- "this week" → This Monday to today
- "2024-01-08 ~ 2024-01-14" → Specified range
- No specification → Default to current week

**Week definition**: Monday to Sunday (ISO standard week)

### Step 2: Batch Fetch Daily Summaries

Call `mcp__cocursor__get_daily_summaries_range`:

**Parameters:**
- `start_date`: Week start date (YYYY-MM-DD)
- `end_date`: Week end date (YYYY-MM-DD)

**Returns:** List of all daily summaries within the range

### Step 3: Aggregate and Analyze

Refer to [references/aggregation-rules.md](references/aggregation-rules.md) for data aggregation:

**Aggregation dimensions:**
- Overall statistics (sessions, code changes, active time)
- Project-level summary
- Work category distribution
- Time distribution

**Analysis focus:**
- **Key accomplishment extraction**: Extract highlights from work_items
- **Progress tracking**: Compare with previous progress
- **Trend insights**: Efficiency trends, problem patterns

### Step 4: Generate Weekly Report

Generate Markdown-formatted weekly report, structure in [references/report-template.md](references/report-template.md).

**Core principles:**
- **Highlight accomplishments**: Use bullet points to clearly list completed work
- **Quantify progress**: Code lines, session counts, issues resolved
- **Concise and impactful**: Suitable for upward reporting

### Step 5: Save Weekly Report

Call `mcp__cocursor__save_weekly_summary`:

**Parameters:**
- `week_start`: Week start date (YYYY-MM-DD)
- `week_end`: Week end date (YYYY-MM-DD)  
- `summary`: Report content (Markdown)
- `language`: Language (zh/en)
- `projects`: Project summary list (JSON object)
- `categories`: Work category statistics (JSON object)
- `total_sessions`: Total session count
- `working_days`: Number of working days with data
- `code_changes`: Code changes summary (JSON object)
- `key_accomplishments`: Key accomplishments list (JSON array)

**⚠️ Important:** All JSON parameters must be passed as objects, not strings.

## Data Format

See [references/data-format.md](references/data-format.md).

## Report Examples

See [references/report-examples.md](references/report-examples.md).

## Important Notes

1. **Accomplishment-oriented**: Weekly report focuses on what was completed, not process details
2. **Data aggregation**: Based on existing daily summaries, no re-analysis of raw sessions
3. **Missing data handling**: Days without daily summaries are handled gracefully, noted in report
4. **Language**: Auto-match based on dominant language in daily summaries

## Idempotency Support

Before generating a new weekly report, check if one already exists and whether source data has changed:

```
# Check existing weekly report
mcp__cocursor__get_weekly_summary(week_start: "2024-01-15")
```

**Response fields:**
- `found`: Whether a report exists for this week
- `needs_update`: Whether source daily summaries have changed since last save
- `summary`: The existing weekly report (if found)

**Recommended workflow:**
1. Call `get_weekly_summary` first to check existing report
2. If `found=false` or `needs_update=true`, generate new report
3. If `found=true` and `needs_update=false`, ask user if they want to regenerate

## MCP Tool Reference

- `mcp__cocursor__get_daily_summaries_range(start_date, end_date)`: Batch fetch daily summaries within date range
- `mcp__cocursor__save_weekly_summary(week_start, week_end, summary, ...)`: Save weekly report
- `mcp__cocursor__get_weekly_summary(week_start)`: Query saved weekly report with idempotency check

## HTTP API Fallback

If MCP tools are unavailable, use HTTP API via curl as an alternative:

### Pre-check: Verify daemon is running
```bash
curl -s http://localhost:19960/health
# Expected: {"status":"ok"}
```

### HTTP API Endpoints
```bash
# Get daily summaries for a date range (equivalent to mcp__cocursor__get_daily_summaries_range)
curl -s "http://localhost:19960/api/v1/daily-summary/range?start_date=2024-01-15&end_date=2024-01-21"

# Get weekly summary with idempotency check (equivalent to mcp__cocursor__get_weekly_summary)
curl -s "http://localhost:19960/api/v1/weekly-summary?week_start=2024-01-15"

# Save weekly summary (equivalent to mcp__cocursor__save_weekly_summary)
curl -X POST "http://localhost:19960/api/v1/weekly-summary" \
  -H "Content-Type: application/json" \
  -d '{
    "week_start": "2024-01-15",
    "week_end": "2024-01-21",
    "summary": "# Weekly Report\n...",
    "language": "zh",
    "total_sessions": 25,
    "working_days": 5
  }'
```

### Error Handling Strategy
1. **First**: Try MCP tool call
2. **If MCP fails** (tool not found, timeout): Use curl fallback
3. **If curl fails** (connection refused): Inform user that cocursor daemon is not running
