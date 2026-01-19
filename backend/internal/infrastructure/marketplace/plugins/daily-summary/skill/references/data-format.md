# Data Format Reference

Complete data structure definitions for daily summary.

## ProjectSummary Structure
```json
{
  "project_name": "Project name",
  "project_path": "Project path",
  "workspace_id": "Workspace ID",
  "work_items": [
    {
      "category": "Work type",
      "description": "Detailed description",
      "session_id": "Associated session ID"
    }
  ],
  "sessions": [
    {
      "session_id": "Session ID",
      "name": "Session name",
      "project_name": "Project name",
      "created_at": timestamp,
      "updated_at": timestamp,
      "message_count": message count,
      "duration": duration in milliseconds
    }
  ],
  "session_count": 5,
  "code_changes": {
    "total_lines_added": 100,
    "total_lines_removed": 20,
    "files_changed": 5,
    "top_changed_files": ["file1.go", "file2.ts"]
  },
  "active_hours": [9, 10, 14, 15]
}
```

## CodeChangeSummary Structure
```json
{
  "total_lines_added": 100,
  "total_lines_removed": 20,
  "files_changed": 5,
  "top_changed_files": ["file1.go", "file2.ts"]
}
```

## TimeDistributionSummary Structure
```json
{
  "morning": {"sessions": 2, "hours": 1.5},
  "afternoon": {"sessions": 4, "hours": 3.2},
  "evening": {"sessions": 2, "hours": 1.8},
  "night": {"sessions": 0, "hours": 0}
}
```

## EfficiencyMetricsSummary Structure
```json
{
  "avg_session_duration": 45.5,
  "avg_messages_per_session": 28.3,
  "total_active_time": 6.5
}
```

## WorkCategories Structure
```json
{
  "requirements_discussion": 3,
  "coding": 8,
  "problem_solving": 4,
  "refactoring": 3,
  "code_review": 0,
  "documentation": 0,
  "testing": 2,
  "other": 1
}
```

## Critical: categories Parameter Format

When calling `save_daily_summary`, pass `categories` as a JSON object directly. The MCP framework handles serialization automatically.

**⚠️ DO NOT:**
- Pass it as a stringified JSON string (e.g., `'{"requirements_discussion": 3, ...}'`)
- Use single quotes instead of double quotes
- Omit commas between fields
- Use Python dictionary syntax with single quotes in the string

**✅ CORRECT Usage:**
```python
# Pass as dictionary/object
categories = {
    "requirements_discussion": 3,
    "coding": 8,
    "problem_solving": 4,
    "refactoring": 3,
    "code_review": 0,
    "documentation": 0,
    "testing": 2,
    "other": 1
}
save_daily_summary(
    date="2024-01-15",
    summary="...",
    categories=categories  # Pass object directly, NOT JSON.stringify(categories)
)
```

**❌ WRONG Usage:**
```python
# Don't stringify
categories_str = '{"requirements_discussion": 3, ...}'
save_daily_summary(..., categories=categories_str)  # ERROR: invalid params!
```

**How to pass in different contexts:**
- **Python**: Pass as a dictionary: `{"requirements_discussion": 3, "coding": 8, ...}`
- **JavaScript/TypeScript**: Pass as an object: `{requirements_discussion: 3, coding: 8, ...}`
- **JSON**: Pass the object directly, not `JSON.stringify()` result
