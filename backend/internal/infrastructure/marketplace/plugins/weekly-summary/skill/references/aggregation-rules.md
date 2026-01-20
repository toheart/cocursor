# Aggregation Rules

Rules for aggregating daily summaries into weekly reports.

## Code Changes Aggregation

### Sum-Based Metrics
- `total_lines_added`: Direct sum
- `total_lines_removed`: Direct sum
- `files_changed`: Direct sum (no deduplication)

### Top Files Merge
```python
# Merge file changes from all dates
all_files = {}
for daily in daily_summaries:
    for file, lines in daily.top_changed_files:
        all_files[file] = all_files.get(file, 0) + lines

# Take Top 10
top_files = sorted(all_files.items(), key=lambda x: x[1], reverse=True)[:10]
```

## Work Categories Aggregation

Direct sum of each category count:

```python
weekly_categories = {
    "requirements_discussion": sum(d.categories.requirements_discussion for d in days),
    "coding": sum(d.categories.coding for d in days),
    "problem_solving": sum(d.categories.problem_solving for d in days),
    "refactoring": sum(d.categories.refactoring for d in days),
    "code_review": sum(d.categories.code_review for d in days),
    "documentation": sum(d.categories.documentation for d in days),
    "testing": sum(d.categories.testing for d in days),
    "other": sum(d.categories.other for d in days)
}
```

## Project Aggregation

Aggregate all daily data by project:

```python
projects = {}
for daily in daily_summaries:
    for project in daily.projects:
        name = project.project_name
        if name not in projects:
            projects[name] = {
                "project_name": name,
                "project_path": project.project_path,
                "sessions": [],
                "work_items": [],
                "code_changes": {"added": 0, "removed": 0, "files": 0},
                "active_days": []
            }
        # Aggregate data
        projects[name]["sessions"].extend(project.sessions)
        projects[name]["work_items"].extend(project.work_items)
        projects[name]["code_changes"]["added"] += project.code_changes.lines_added
        projects[name]["code_changes"]["removed"] += project.code_changes.lines_removed
        projects[name]["active_days"].append(daily.date)
```

## Accomplishment Extraction

### Extract Key Accomplishments from work_items

1. **Group by project**
2. **Cluster by category** related work
3. **Merge descriptions** to form accomplishment statements

```python
def extract_accomplishments(projects):
    accomplishments = []
    for project in projects:
        # Group related work_items by feature/module
        grouped = group_related_items(project.work_items)
        for group in grouped:
            accomplishment = {
                "title": synthesize_title(group),
                "description": merge_descriptions(group),
                "project": project.project_name,
                "code_changes": sum_code_changes(group)
            }
            accomplishments.append(accomplishment)
    return accomplishments
```

### Related Work Item Detection

Criteria for relatedness:
- Involves same files or modules
- Belongs to same feature development
- Related work on consecutive dates

## Efficiency Metrics Aggregation

### Weighted Average Calculation
```python
total_sessions = sum(d.total_sessions for d in days)
avg_duration = sum(
    d.efficiency_metrics.avg_session_duration * d.total_sessions
    for d in days
) / total_sessions
```

### Most Productive Day Identification
```python
best_day = max(days, key=lambda d: d.total_sessions * d.efficiency_metrics.efficiency_score)
```

## Tech Debt Aggregation

Collect tech_debt from all daily summaries:
```python
all_todos = []
for daily in daily_summaries:
    for item in daily.tech_debt:
        all_todos.append({
            "item": item,
            "source_date": daily.date
        })
# Deduplicate similar items
todos = deduplicate_todos(all_todos)
```

## Missing Data Handling

### No Daily Summary for a Day
- Not counted in working_days
- Does not affect sum-based metrics
- Can be noted in weekly report

### Partial Data Missing
- Use available data
- Default to 0 for missing fields
