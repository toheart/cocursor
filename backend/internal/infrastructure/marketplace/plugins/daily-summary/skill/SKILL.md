---
name: daily-summary
description: Automatically summarize Cursor chat records to generate daily work reports. Supports querying sessions by date, reading text-only dialogues, identifying work types, extracting tech debt, and saving summaries to database. Use this skill when users request work summaries, daily reports, or need to review daily work content.
---

# Daily Summary Skill

> **Important Note**: This skill is provided by the **cocursor project** and requires calling related tools through the cocursor MCP server.
> 
> - This skill depends on tools provided by the cocursor MCP server: `get_daily_sessions`, `get_session_content`, `save_daily_summary`, `get_daily_summary`
> - Ensure the cocursor daemon is running and the MCP server is properly configured
> - The skill is installed via the cocursor plugin system to `~/.claude/skills/daily-summary/`
> - Invocation: `Bash("openskills read daily-summary")`

Automatically summarize Cursor chat records to generate daily work reports.

## Workflow

### 1. Get Session List
Use the `get_daily_sessions` MCP tool to get the session list for a specified date:
- Parameter: `date` (optional, format: YYYY-MM-DD, defaults to today)
- Returns: Session list grouped by project

### 2. Read Session Content
Call the `get_session_content` MCP tool for each session:
- Parameter: `session_id` (required)
- Returns: Plain text message list (filtered to exclude tool calls and code blocks)

### 3. Analyze and Generate Summary
Analyze all conversation content and generate a Markdown-formatted summary:

**Summary Structure**:
```markdown
# {date} Work Summary

## Overview
[Overall statistics: total sessions, number of projects involved, main work types, etc.]

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

**Work Items:**
1. [Work Type] {Detailed description}
2. [Work Type] {Detailed description}
...

**Related Sessions:**
- {session_name} ({duration} minutes, {message_count} messages)

### Project 2: {project_name}
...

## Tech Debt / Follow-up Plans
[Automatically extracted todo items, including:
- "Will optimize this later"
- "Temporary handling for this"
- "TODO" markers
- Other deferred work items]
```

### 4. Save Summary
Use the `save_daily_summary` MCP tool to save the summary:
- Parameters:
  - `date`: Date (YYYY-MM-DD)
  - `summary`: Summary content (Markdown)
  - `language`: Language (zh/en, determined based on chat content)
  - `projects`: Project list (includes work items and session information)
  - `categories`: Work category statistics
  - `total_sessions`: Total number of sessions

## Work Category Identification

Identify work types based on conversation content:

- **Requirements Discussion** (requirements_discussion): 
  - Keywords: requirements, discussion, proposal, design, planning, confirmation
  - Scenarios: Discussing feature requirements, design proposals, technology selection, etc.

- **Coding** (coding):
  - Keywords: implement, write, develop, add, create, modify
  - Scenarios: Implementing new features, writing code, modifying files, etc.

- **Problem Solving** (problem_solving):
  - Keywords: bug, error, issue, fix, troubleshoot, debug, exception
  - Scenarios: Fixing bugs, troubleshooting issues, debugging errors, etc.

- **Refactoring** (refactoring):
  - Keywords: refactor, optimize, improve, clean up, organize, rewrite
  - Scenarios: Code refactoring, performance optimization, code cleanup, etc.

- **Code Review** (code_review):
  - Keywords: review, check, evaluate, suggest
  - Scenarios: Code review, code inspection, etc.

- **Documentation** (documentation):
  - Keywords: document, documentation, comment, README, note
  - Scenarios: Writing documentation, adding comments, etc.

- **Testing** (testing):
  - Keywords: test, unit test, integration test, verify, validation
  - Scenarios: Writing tests, running tests, etc.

- **Other** (other):
  - Work that cannot be clearly categorized

## Tech Debt Extraction

Extract tech debt and follow-up plans from conversations, focusing on the following patterns:

- **Deferred Markers**:
  - "Will do this later..."
  - "Temporary solution..."
  - "Will optimize later..."
  - "TODO", "FIXME", "XXX"

- **Temporary Solutions**:
  - "Temporary handling"
  - "Leave it as is for now"
  - "Will improve later"

- **Known Issues**:
  - "Known issue"
  - "To be fixed"
  - "Needs optimization"

When extracting, include:
- Problem description
- Associated session ID
- Project it belongs to

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

### ProjectSummary Structure
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
  "session_count": session count
}
```

### WorkCategories Structure
```json
{
  "requirements_discussion": count,
  "coding": count,
  "problem_solving": count,
  "refactoring": count,
  "code_review": count,
  "documentation": count,
  "testing": count,
  "other": count
}
```

## Important Notes

1. **Date Handling**:
   - Use local timezone
   - Date format: YYYY-MM-DD
   - Default query is today, can specify other dates via parameter

2. **Content Filtering**:
   - Automatically filter tool calls
   - Automatically filter code blocks
   - Keep only user and AI text conversations

3. **Project Names**:
   - Prefer Git remote repository name
   - Fallback to directory name
   - Obtained via ProjectManager

4. **Session Filtering**:
   - Include sessions created or updated on the target date
   - Ensure all active sessions are covered

5. **Summary Quality**:
   - Work items must include detailed descriptions, not just categories
   - Automatically deduplicate and merge duplicate work
   - Extract tech debt and follow-up plans
   - Include session duration information

## MCP Tool Reference

- `get_daily_sessions(date?)`: Query session list for specified date
- `get_session_content(session_id)`: Read plain text content of session
- `save_daily_summary(date, summary, language, projects, categories, total_sessions)`: Save summary
- `get_daily_summary(date)`: Query historical summary (optional)
