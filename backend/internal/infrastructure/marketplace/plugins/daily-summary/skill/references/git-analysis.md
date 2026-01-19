# Git Commit Analysis Guide

Optional feature to analyze git commits and complement session records.

## When to Use
- Need to verify actual code contributions
- Want to match commits with sessions
- Projects have good commit hygiene

## Implementation Steps

1. **Get Active Projects**: From `get_daily_sessions` result, extract projects with sessions today
2. **Quick Check**: For each project path:
   - Check if `.git` directory exists
   - Check if there are commits today using: `git log --since "YYYY-MM-DD 00:00:00" --until "YYYY-MM-DD 23:59:59" --oneline`
3. **Full Analysis**: Only for projects with commits today:
   - Execute: `git log --since "YYYY-MM-DD 00:00:00" --until "YYYY-MM-DD 23:59:59" --format="%H|%an|%ae|%at|%s" --numstat`
   - Parse output to extract commit information (hash, author, timestamp, message, file changes)
4. **Match with Sessions**: Use time window matching (±2 hours default) to associate commits with sessions

## Git Command Examples

```bash
# Quick check: Has commits today?
cd "$project_path"
commit_count=$(git log --since="2024-01-15 00:00:00" --until="2024-01-15 23:59:59" --oneline | wc -l)

# Full analysis: Get commit details
git log --since="2024-01-15 00:00:00" \
       --until="2024-01-15 23:59:59" \
       --format="%H|%an|%ae|%at|%s" \
       --numstat
```

## Performance Considerations
- Skip non-git repositories quickly (check `.git` directory)
- Skip projects without commits today (quick check before full analysis)
- Set timeout for git commands (5 seconds default)
- Handle errors gracefully (skip on failure, don't fail entire process)

## Error Handling
- Not a git repository: Skip silently
- Git command fails: Log warning, skip project
- Timeout: Skip project, continue with others

## Time Window Matching

Default time window: ±2 hours

When matching commits with sessions:
- Compare commit timestamp with session `CreatedAt` or `UpdatedAt`
- If within ±2 hours, consider them related
- Adjust window size if needed based on work patterns

## Complete Workflow Example

```bash
# 1. Get active projects from get_daily_sessions
projects=$(get_daily_sessions --date "2024-01-15")

# 2. For each project
for project in $projects; do
    project_path=$(echo $project | jq -r '.project_path')
    
    # Quick check: Is git repo?
    if [ ! -d "$project_path/.git" ]; then
        continue
    fi
    
    # Quick check: Has commits today?
    commit_count=$(cd "$project_path" && \
        git log --since="2024-01-15 00:00:00" \
               --until="2024-01-15 23:59:59" \
               --oneline | wc -l)
    
    if [ "$commit_count" -eq 0 ]; then
        continue
    fi
    
    # Full analysis: Get commit details
    cd "$project_path"
    git log --since="2024-01-15 00:00:00" \
           --until="2024-01-15 23:59:59" \
           --format="%H|%an|%ae|%at|%s" \
           --numstat
done
```
