# Data Format

## save_weekly_summary Parameters

```json
{
  "week_start": "2024-01-08",
  "week_end": "2024-01-14",
  "summary": "# 2024-01-08 ~ 2024-01-14 Weekly Report\n\n...",
  "language": "zh",
  "projects": [
    {
      "project_name": "cocursor",
      "project_path": "/Users/user/code/cocursor",
      "total_sessions": 15,
      "active_days": 5,
      "work_items": [
        {
          "category": "coding",
          "description": "Implemented user auth API",
          "day": "2024-01-08"
        }
      ],
      "code_changes": {
        "lines_added": 500,
        "lines_removed": 120,
        "files_changed": 15
      }
    }
  ],
  "categories": {
    "requirements_discussion": 3,
    "coding": 20,
    "problem_solving": 8,
    "refactoring": 5,
    "code_review": 2,
    "documentation": 3,
    "testing": 4,
    "other": 1
  },
  "total_sessions": 35,
  "working_days": 5,
  "code_changes": {
    "total_lines_added": 1200,
    "total_lines_removed": 280,
    "total_files_changed": 45,
    "top_changed_files": [
      {"file": "api/handler.go", "lines": 150},
      {"file": "service/user.go", "lines": 120}
    ]
  },
  "key_accomplishments": [
    {
      "title": "User Authentication Module",
      "description": "Completed login API with JWT support",
      "project": "cocursor",
      "code_lines": 350
    },
    {
      "title": "RAG Configuration Optimization",
      "description": "Refactored config flow for better UX",
      "project": "cocursor",
      "code_lines": 200
    }
  ]
}
```

## WeeklyProjectSummary Structure

```json
{
  "project_name": "Project name",
  "project_path": "Project path",
  "total_sessions": 15,
  "active_days": 5,
  "work_items": [
    {
      "category": "Work type",
      "description": "Detailed description",
      "day": "2024-01-08"
    }
  ],
  "code_changes": {
    "lines_added": 500,
    "lines_removed": 120,
    "files_changed": 15
  }
}
```

## KeyAccomplishment Structure

```json
{
  "title": "Accomplishment title",
  "description": "Accomplishment description",
  "project": "Related project",
  "code_lines": 350
}
```

## WeeklyCategories Structure

```json
{
  "requirements_discussion": 3,
  "coding": 20,
  "problem_solving": 8,
  "refactoring": 5,
  "code_review": 2,
  "documentation": 3,
  "testing": 4,
  "other": 1
}
```

## CodeChangesSummary Structure

```json
{
  "total_lines_added": 1200,
  "total_lines_removed": 280,
  "total_files_changed": 45,
  "top_changed_files": [
    {"file": "api/handler.go", "lines": 150},
    {"file": "service/user.go", "lines": 120}
  ]
}
```

## Parameter Passing Notes

⚠️ **All JSON parameters must be passed as objects, not strings:**

```python
# ✅ Correct
categories = {"coding": 20, "problem_solving": 8, ...}
save_weekly_summary(..., categories=categories)

# ❌ Wrong
categories_str = '{"coding": 20, ...}'
save_weekly_summary(..., categories=categories_str)
```
