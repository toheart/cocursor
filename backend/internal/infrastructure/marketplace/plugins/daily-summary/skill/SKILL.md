---
name: daily-summary
description: Automatically summarize Cursor chat records to generate daily work reports. Supports querying sessions by date, reading text-only dialogues, identifying work types, extracting tech debt, and saving summaries to database. Can optionally integrate git commit analysis for comprehensive work tracking. Use this skill when users request work summaries, daily reports, or need to review daily work content.
---

# Daily Summary Skill

> **MCP Server Dependency**: This skill requires the `cocursor` MCP server.
> 
> Available tools (use full names when calling):
> - `mcp__cocursor__get_daily_conversations` - Get all conversations for a date (recommended)
> - `mcp__cocursor__get_daily_sessions` - Get session list for a date
> - `mcp__cocursor__get_session_content` - Get content of a specific session
> - `mcp__cocursor__save_daily_summary` - Save generated summary
> - `mcp__cocursor__get_daily_summary` - Query saved summary

Automatically summarize Cursor chat records to generate daily work reports.

## Initial Decision: Git Analysis

**Before starting analysis, decide whether to include Git commit analysis:**

1. **Check user input**: If user specified `--with-git` or `--no-git` in command, use that decision
2. **Ask user if not specified**: "是否需要包含 Git commit 分析？这将分析当天有提交的项目，补充会话记录数据。注意：这需要遍历项目目录执行 git 命令，可能需要一些时间。(y/n)"
3. **Based on response**:
   - `y` or `yes`: Enable Git analysis (see [references/git-analysis.md](references/git-analysis.md))
   - `n` or `no`: Skip Git analysis, use only session records

**Note**: Git analysis is optional and can be skipped if:
- User explicitly says no
- Performance is a concern
- Projects don't have git repositories

## Workflow

### Option 1: Get All Conversations at Once (Recommended)
Call `mcp__cocursor__get_daily_conversations` to get all session conversations for a specified date in one call:
- Parameter: `date` (optional, format: YYYY-MM-DD, defaults to today)
- Returns: All conversations grouped by project, including full message content
- **Advantage**: Single MCP call, more efficient than calling `get_session_content` multiple times

### Option 2: Get Session List Then Read Content (Alternative)
If you only need session metadata first:

1. Call `mcp__cocursor__get_daily_sessions` to get the session list:
   - Parameter: `date` (optional, format: YYYY-MM-DD, defaults to today)
   - Returns: Session list grouped by project (metadata only, no message content)

2. Call `mcp__cocursor__get_session_content` for each session (only if needed):
   - Parameter: `session_id` (required)
   - Returns: Plain text message list (filtered to exclude tool calls and code blocks)
   - **Note**: This requires multiple MCP calls, less efficient than `mcp__cocursor__get_daily_conversations`

### 3. Analyze and Generate Summary
Analyze all conversation content and generate a Markdown-formatted summary.

**Analysis includes:**
- Work type identification
- Code change statistics (from session metadata)
- Time distribution
- **AI coding efficiency** (from conversation analysis)
- **Issues and optimizations** (from conversation patterns)
- Tech debt extraction

**Summary Structure**:
```markdown
# {date} Work Summary

## Overview
[Overall statistics: total sessions, number of projects involved, main work types, etc.]

## Code Changes Summary
- Total Lines Added: XXX
- Total Lines Removed: XXX
- Files Changed: XXX
- Top Changed Files:
  1. file1.go (50 lines)
  2. file2.ts (30 lines)

## Time Distribution
- Morning (9-12): X sessions, X hours
- Afternoon (14-18): X sessions, X hours
- Evening (19-22): X sessions, X hours

## Efficiency Metrics
- Average Session Duration: X minutes
- Average Messages per Session: X
- Total Active Time: X hours

## Work Category Statistics
- Requirements Discussion: X times
- Coding: X times
- Problem Solving: X times
- Refactoring: X times
- Code Review: X times
- Documentation: X times
- Testing: X times
- Other: X times

## Project Details

### Project 1: {project_name}
**Path:** {project_path}

**Code Changes:**
- Lines Added: XXX
- Lines Removed: XXX
- Files Changed: XXX

**Active Hours:** [9, 10, 14, 15]

**Work Items:**
1. [Work Type] {Detailed description}
2. [Work Type] {Detailed description}

**Related Sessions:**
- {session_name} ({duration} minutes, {message_count} messages)

### Project 2: {project_name}
...

## AI Coding Efficiency
- Estimated Acceptance Rate: XX%
- High Acceptance Sessions: X
- Low Acceptance Sessions: X
- Context Issues: X
- Quick Resolved Issues: X
- Long Stuck Issues: X
- Repeated Issues: X
- Efficiency Score: XX/100

## Issues & Optimizations

### Issues Found
1. [Severity] Category: Description
   - Occurrences: X
   - Sessions: session-1, session-2
   - Example: "..."

### Optimization Suggestions
1. [Priority] Category: Suggestion
   - Impact: efficiency/quality/experience
   - Related Sessions: session-3

## Tech Debt / Follow-up Plans
[Automatically extracted todo items]
```

### 4. Save Summary
Call `mcp__cocursor__save_daily_summary` to save the summary:

**Parameters:**
- `date`: Date (YYYY-MM-DD)
- `summary`: Summary content (Markdown)
- `language`: Language (zh/en, determined based on chat content)
- `projects`: Project list (includes work items and session information)
- `categories`: **CRITICAL** - Work category statistics **MUST be a JSON object, NOT a string**
- `total_sessions`: Total number of sessions
- `code_changes`: Code change statistics (optional, JSON object)
- `time_distribution`: Time distribution statistics (optional, JSON object)
- `efficiency_metrics`: Efficiency metrics (optional, JSON object)

**⚠️ CRITICAL: `categories` Parameter Format**

The `categories` parameter **MUST** be passed as a JSON object directly, NOT as a stringified JSON string.

**Quick Reference:**
- ✅ CORRECT: Pass as object/dictionary: `{"requirements_discussion": 3, "coding": 8, ...}`
- ❌ WRONG: Pass as string: `'{"requirements_discussion": 3, ...}'`

See [references/data-format.md](references/data-format.md) for complete format details and examples.

## Work Category Identification

See [references/work-categories.md](references/work-categories.md) for detailed work category identification rules and examples.

## Tech Debt Extraction

See [references/tech-debt-patterns.md](references/tech-debt-patterns.md) for tech debt extraction patterns and guidelines.

## AI Coding Efficiency Analysis

Analyze AI coding efficiency from conversation content (non-structured data).

**Key Metrics:**
- Acceptance rate estimation (from conversation feedback)
- Context management issues (context loss, resets)
- Problem resolution efficiency (quick resolved vs long stuck)
- Code quality feedback (concerns vs praise)
- Iteration count per task
- Overall efficiency score (0-100)

See [references/ai-efficiency-analysis.md](references/ai-efficiency-analysis.md) for detailed analysis patterns and calculation methods.

## Issues and Optimizations Analysis

Extract issues and optimization opportunities from conversations.

**Issue Categories:**
- Context loss (上下文丢失)
- Repeated problems (重复问题)
- Stuck/blocked (卡住/阻塞)
- Quality issues (质量问题)
- Tool usage problems (工具使用问题)

**Optimization Categories:**
- Workflow improvements (工作流程)
- Prompt quality (提示词质量)
- Tool usage optimization (工具使用)
- Context management (上下文管理)

See [references/issues-optimizations-analysis.md](references/issues-optimizations-analysis.md) for detailed patterns and extraction guidelines.

## Optional: Git Commit Analysis

When enabled, the skill can analyze git commits to complement session records.

**Decision Point**: Ask user or check command parameters (`--with-git` / `--no-git`)

**Quick Overview:**
- Extract active projects from `mcp__cocursor__get_daily_sessions` result
- Quick check: Verify git repository and commits today
- Full analysis: Get commit details only for projects with commits
- Match commits with sessions using time window (±2 hours default)

See [references/git-analysis.md](references/git-analysis.md) for complete implementation guide, command examples, and best practices.

## Deduplication and Merging Strategy

When users discuss the same work across multiple sessions, merge into a single work item:

1. **Identify Duplicates**:
   - Same work type
   - Similar work descriptions (semantically similar)
   - Involving the same files or features

2. **Merging Rules**:
   - Merge descriptions, keeping the most detailed information
   - Merge associated session IDs
   - Count work type only once

3. **Example**:
   - Session 1: Fix login bug
   - Session 2: Fix login issue
   - Session 3: Resolve login error
   - → Merged as: Problem Solving - Fix login feature issues

## Data Format

See [references/data-format.md](references/data-format.md) for complete data structure definitions including:
- `ProjectSummary` structure
- `CodeChangeSummary` structure
- `TimeDistributionSummary` structure
- `EfficiencyMetricsSummary` structure
- `WorkCategories` structure
- Critical `categories` parameter format requirements and examples

## Important Notes

1. **Date Handling**: Use local timezone, format: YYYY-MM-DD
2. **Content Filtering**: Automatically filter tool calls and code blocks
3. **Project Names**: Prefer Git remote repository name, fallback to directory name
4. **Session Filtering**: Include sessions created or updated on the target date
5. **Summary Quality**: Work items must include detailed descriptions, automatically deduplicate
6. **⚠️ CRITICAL: `categories` Parameter Format**: 
   - **MUST** be a JSON object, NOT a string
   - Pass the object directly, do NOT use `JSON.stringify()` or similar
   - Use double quotes for keys, not single quotes
   - Include all commas between fields
   - The MCP framework handles serialization automatically

## Examples

See [references/summary-examples.md](references/summary-examples.md) for complete summary examples and best practices.

## MCP Tool Reference

- `mcp__cocursor__get_daily_conversations(date?)`: **Recommended** - Get all session conversations for specified date in one call (includes full message content)
- `mcp__cocursor__get_daily_sessions(date?)`: Query session list for specified date (metadata only, no message content)
- `mcp__cocursor__get_session_content(session_id)`: Read plain text content of a single session (use only if you need individual session content after getting session list)
- `mcp__cocursor__save_daily_summary(date, summary, language, projects, categories, total_sessions, code_changes?, time_distribution?, efficiency_metrics?)`: Save summary
- `mcp__cocursor__get_daily_summary(date)`: Query historical summary (optional)

**Performance Tip**: Always prefer `mcp__cocursor__get_daily_conversations` over multiple `mcp__cocursor__get_session_content` calls. It's more efficient and reduces MCP round trips.
