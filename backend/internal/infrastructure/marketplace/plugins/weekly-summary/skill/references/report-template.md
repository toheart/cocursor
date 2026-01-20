# Report Template

## Standard Weekly Report Structure

```markdown
# {week_start} ~ {week_end} Weekly Report

## Overview

- **Working Days**: X days
- **Total Sessions**: XX
- **Projects Involved**: X
- **Code Changes**: +XXXX / -XXX lines

## Key Accomplishments

### 1. [Accomplishment Title]
- Description of completed work
- Related Project: project-name
- Code Changes: +XX lines

### 2. [Accomplishment Title]
- Description of completed work
- Related Project: project-name

### 3. [Accomplishment Title]
...

## Project Progress

### Project A
**This Week:**
- ✅ Feature A implementation
- ✅ Bug B fix
- 🔄 Feature C in progress

**Code Stats:** +XXX / -XX lines, X files changed

### Project B
...

## Work Distribution

| Category | Count | Percentage |
|----------|-------|------------|
| Coding | XX | XX% |
| Problem Solving | XX | XX% |
| Refactoring | XX | XX% |
| Requirements Discussion | XX | XX% |
| Testing | XX | XX% |
| Documentation | XX | XX% |

## Efficiency Analysis

- **Average Daily Sessions**: X.X
- **Average Daily Active Time**: X.X hours
- **Most Productive Day**: Day X (X sessions, X hours)
- **AI Efficiency Score**: XX/100

## Follow-up Items

Tech debt and todos extracted from daily summaries:
1. [Todo Item 1] - Source: Day X
2. [Todo Item 2] - Source: Day X

## Next Week Focus

Based on this week's progress, suggested focus areas:
1. [Suggestion 1]
2. [Suggestion 2]
```

## Simplified Report (Quick Summary)

For quick reporting scenarios:

```markdown
# {week_start} ~ {week_end} Weekly Report

## Completed This Week
1. **[Project A]** Completed feature X development
2. **[Project B]** Fixed issue Y
3. **[Project C]** Refactored module Z

## Statistics
- Sessions: XX | Code: +XXXX/-XXX | Projects: X

## Follow-ups
- Item 1
- Item 2
```

## Accomplishment Extraction Principles

### Extracting from Daily work_items

**Raw Data Example:**
```
Day 1: [Coding] Implemented user login API
Day 2: [Coding] Added JWT token validation
Day 3: [Testing] Wrote login unit tests
Day 4: [Problem Solving] Fixed token expiry bug
```

**Extracted Accomplishment:**
```
### User Authentication Module
- Completed login API development with JWT support
- Covered with unit tests, fixed token expiry issue
- Code Changes: +350 lines, 8 files
```

### Merge Rules

1. **Merge Related Work**: Combine multiple work items for same feature/module into one accomplishment
2. **Highlight Results**: Emphasize "what was completed" not "what was done"
3. **Quantify**: Use numbers whenever possible (code lines, files, issues)

## Status Markers

- ✅ Completed
- 🔄 In Progress
- ⏸️ Paused
- ❌ Cancelled
