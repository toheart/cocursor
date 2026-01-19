# Summary Examples and Best Practices

## Complete Summary Example

```markdown
# 2024-01-15 Work Summary

## Overview
- Total Sessions: 8
- Projects Involved: 2
- Main Work Types: Coding (5), Problem Solving (2), Refactoring (1)

## Code Changes Summary
- Total Lines Added: 450
- Total Lines Removed: 120
- Files Changed: 15
- Top Changed Files:
  1. backend/api/handler.go (120 lines)
  2. frontend/components/UserList.tsx (80 lines)
  3. backend/internal/service/user.go (65 lines)

## Time Distribution
- Morning (9-12): 2 sessions, 1.5 hours
- Afternoon (14-18): 4 sessions, 3.2 hours
- Evening (19-22): 2 sessions, 1.8 hours

## Efficiency Metrics
- Average Session Duration: 45 minutes
- Average Messages per Session: 28
- Total Active Time: 6.5 hours

## Work Category Statistics
- Requirements Discussion: 1
- Coding: 5
- Problem Solving: 2
- Refactoring: 1
- Code Review: 0
- Documentation: 0
- Testing: 1
- Other: 0

## Project Details

### Project 1: cocursor
**Path:** /Users/user/code/cocursor

**Code Changes:**
- Lines Added: 300
- Lines Removed: 80
- Files Changed: 10

**Active Hours:** [9, 10, 14, 15, 16, 20]

**Work Items:**
1. [Coding] Implemented user authentication API endpoint
2. [Problem Solving] Fixed database connection timeout issue
3. [Refactoring] Refactored session management code

**Related Sessions:**
- "Add user auth" (45 minutes, 25 messages)
- "Fix DB timeout" (30 minutes, 18 messages)
- "Refactor sessions" (60 minutes, 35 messages)

### Project 2: my-app
**Path:** /Users/user/code/my-app

**Code Changes:**
- Lines Added: 150
- Lines Removed: 40
- Files Changed: 5

**Active Hours:** [14, 15, 21]

**Work Items:**
1. [Coding] Added user profile page
2. [Testing] Wrote unit tests for user service

**Related Sessions:**
- "User profile page" (50 minutes, 30 messages)
- "User service tests" (40 minutes, 22 messages)

## Tech Debt / Follow-up Plans
- Optimize database query in user service (mentioned in session "Fix DB timeout")
- Refactor authentication middleware (TODO marker in session "Add user auth")
- Add error handling for API endpoints (mentioned in session "User profile page")
```

## Best Practices

### Work Item Descriptions
- **Good**: "Implemented user authentication API endpoint with JWT token support"
- **Bad**: "Coding"

### Tech Debt Extraction
- Include context: What needs to be done and why
- Link to session: Help trace back to original discussion
- Be specific: "Optimize query" not "Fix performance"

### Code Changes Summary
- Include both global and project-level statistics
- List top changed files (default: top 5)
- Show net change (added - removed)

### Time Distribution
- Use clear time slots (morning, afternoon, evening, night)
- Show both session count and total hours
- Help identify work patterns
