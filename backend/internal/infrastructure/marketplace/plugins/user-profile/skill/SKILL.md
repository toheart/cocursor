---
name: user-profile
description: Generate personalized user profile from Cursor chat history to help AI understand coding style, technical preferences, and communication habits. Use when users want to create or update their profile, or say "let Cursor know me better", "analyze my habits", "generate my profile".
---

# User Profile Skill

> **Requires**: cocursor daemon running at `localhost:19960`

## Pre-check

```bash
curl -s http://localhost:19960/health
# Expected: {"status":"ok"}
```

## Workflow

**MUST complete all steps including SAVE.**

### Step 1: Get User Messages

```bash
# macOS / Linux
curl -s -X POST "http://localhost:19960/api/v1/profile/messages" \
  -H "Content-Type: application/json" \
  -d '{"scope":"project","project_path":"/path/to/project","days_back":30}'

# Windows PowerShell
$body = @{scope="project"; project_path="C:/path/to/project"; days_back=30} | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:19960/api/v1/profile/messages" -Method POST -ContentType "application/json" -Body $body
```

**Parameters:**
- `scope`: "global" or "project"
- `project_path`: Required if scope is "project"
- `days_back`: Days to analyze (default 30)

**Returns:**
- `messages`: User messages (recent + historical)
- `stats`: Statistics including `primary_language` ('zh' or 'en')
- `existing_profile`: Current profile if exists
- `meta`: Includes `needs_update` flag

### Step 2: Check if Update Needed

If `meta.needs_update` is false: Ask user if they want to force regeneration.

### Step 3: Analyze Messages

Analyze across dimensions:
- **Coding Style**: Naming conventions, architecture preferences, comment style
- **Technical Profile**: Expert/Proficient/Learning categorization
- **Communication Style**: Question patterns, feedback patterns, language preference
- **Work Habits**: Active time periods, session depth, problem-solving approach

### Step 4: Generate Profile

Generate Markdown in user's `stats.primary_language`:

```markdown
# User Profile

## Coding Style
- [Habits, naming, architecture preferences]

## Technical Profile
- **Expert**: [Languages/frameworks]
- **Proficient**: [Languages/frameworks]
- **Learning**: [Languages/frameworks]

## Communication Style
- [Question style, feedback patterns]

## Work Habits
- Active hours: [Time periods]
- Session style: [Deep/Quick]
```

### Step 5: SAVE Profile (REQUIRED)

```bash
# macOS / Linux
curl -s -X POST "http://localhost:19960/api/v1/profile" \
  -H "Content-Type: application/json" \
  -d '{"scope":"project","project_path":"/path/to/project","content":"# User Profile\n...","language":"zh"}'

# Windows PowerShell
$body = @{scope="project"; project_path="C:/path/to/project"; content="# User Profile..."; language="zh"} | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:19960/api/v1/profile" -Method POST -ContentType "application/json" -Body $body
```

**Parameters:**
- `scope`: Same as Step 1
- `project_path`: Same as Step 1
- `content`: Generated Markdown (no frontmatter)
- `language`: Use `stats.primary_language` value

### Step 6: Confirm to User (REQUIRED)

After save, inform user:
1. File path where saved
2. Auto-load explanation
3. Git-ignored status
4. Brief profile summary

## Output Locations

| Scope | Location | Auto-loaded |
|-------|----------|-------------|
| Global | `~/.cocursor/profiles/global.md` | Via merge |
| Project | `{project}/.cursor/rules/user-profile.mdc` | Yes |

## Important Notes

1. **Privacy**: Stored locally only
2. **Git Safety**: Project profiles auto-added to `.gitignore`
3. **Incremental**: Merge with existing profile, don't replace entirely
4. **User Messages Only**: Analyze only user messages, not AI responses

## API Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/profile/messages` | POST | Get user messages for analysis |
| `/api/v1/profile` | POST | Save user profile |
