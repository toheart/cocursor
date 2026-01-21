---
name: user-profile
description: Generate personalized user profile from Cursor chat history to help AI understand user's coding style, technical preferences, and communication habits. Use this skill when users want to create or update their profile, or when they say "let Cursor know me better", "analyze my habits", "generate my profile", "分析我的画像", "更新用户画像". This skill requires the cocursor MCP server.
---

# User Profile Skill

> **MCP Server Dependency**: This skill requires the `cocursor` MCP server.
> 
> Available tools:
> - `mcp__cocursor__get_user_messages_for_profile` - Get user messages for analysis
> - `mcp__cocursor__save_user_profile` - Save generated profile

## Critical: Complete Workflow Required

**MUST complete all steps including SAVE.** The workflow is NOT complete until the profile is saved and confirmed to user.

## Workflow Overview

1. Get user messages → 2. Check update needed → 3. Analyze → 4. Generate profile → **5. SAVE profile** → **6. Confirm to user**

### Step 1: Get User Messages

Call `mcp__cocursor__get_user_messages_for_profile` with:
- `scope`: "global" or "project"
- `project_path`: (required if scope is "project")
- `days_back`: Number of days to analyze (default: 30)

The tool returns:
- User messages (only user's input, not AI responses)
- Statistics (time distribution, project distribution, **primary_language**)
- Existing profile (if any, for incremental update)
- Metadata (data hash for idempotency check)

**Important**: Note the `stats.primary_language` value ('zh' or 'en') - use this for Step 5 and Step 6.

### Step 2: Check if Update Needed

If `meta.needs_update` is `false` and user didn't force refresh:
- Inform user: "Your profile is up to date. No new conversations to analyze. 您的画像已是最新，没有新的对话需要分析。"
- Ask if they want to force regeneration

### Step 3: Analyze and Generate Profile

Analyze user messages across these dimensions:

#### 编码风格 (Coding Style)
From user messages, identify:
- Naming conventions mentioned (camelCase, snake_case, etc.)
- Architecture preferences (DDD, MVC, Clean Architecture, etc.)
- Code organization habits
- Comment style preferences (language, verbosity)

Indicators:
- Variable/function names the user mentions
- Architecture terms in discussions
- Style preferences expressed explicitly

#### 技术画像 (Technical Profile)
Categorize by proficiency level:
- **Expert (精通)**: Discusses internals, edge cases, performance optimization
- **Proficient (熟练)**: Knows best practices, asks about alternatives
- **Learning (学习中)**: Basic questions, conceptual confusion

From messages, identify:
- Primary languages and frameworks
- Tools and platforms
- Areas of active learning

#### 沟通风格 (Communication Style)
Analyze communication patterns:
- Question style: direct ("帮我实现...") vs exploratory ("你觉得...有什么建议")
- Expected response detail level
- Feedback patterns ("不对", "换个方案", "可以", "很好")
- Language preference (Chinese/English/mixed)

#### 工作习惯 (Work Habits)
From statistics and message patterns:
- Active time periods (use `stats.time_distribution`)
- Session depth preference (quick fixes vs deep exploration)
- Problem-solving approach
- Iteration style (fast prototype vs careful planning)

#### 项目上下文 (Project Context) - For project-level profiles only
- Familiar modules and code areas
- Common task types in this project
- Project-specific conventions (comment language, log format, etc.)

### Step 4: Incremental Update

If `existing_profile` exists:
1. Compare new analysis with existing profile
2. Keep stable characteristics that remain consistent
3. Update characteristics that have changed
4. Add new discoveries
5. Note: Don't completely overwrite - merge intelligently

### Step 5: Generate Markdown Profile

Generate profile **in the user's primary language** (use `stats.primary_language`):
- If 'zh': Generate in Chinese
- If 'en': Generate in English

Format:

```markdown
# User Profile

## 编码风格
- [具体的编码习惯描述]
- [命名偏好]
- [架构偏好]
- [注释风格偏好]

## 技术画像
- **精通**: [语言/框架列表]
- **熟练**: [语言/框架列表]
- **学习中**: [语言/框架列表]

## 沟通风格
- [提问方式描述]
- [期望的回答风格]
- [反馈模式]
- [语言偏好]

## 工作习惯
- 活跃时段: [时间段描述]
- 会话风格: [深度/快速]
- 问题解决: [方式描述]

## 项目约定 (项目级)
- [项目特定的约定]
```

### Step 6: SAVE Profile (REQUIRED)

**This step is MANDATORY. Do NOT skip.** After generating the profile, MUST call save tool.

Call `mcp__cocursor__save_user_profile` with:
- `scope`: "global" or "project" (use same scope as Step 1)
- `project_path`: (required if scope is "project")
- `content`: Generated Markdown content (without frontmatter - the tool adds it automatically)
- `language`: Use `stats.primary_language` value ('zh' or 'en') to set correct frontmatter description language

### Step 7: Confirm to User (REQUIRED)

After save tool returns success, MUST inform user:

1. **保存位置**: Display the `file_path` from save result
2. **生效说明**: Explain the profile will auto-load in future conversations
3. **Git 状态**: If `git_ignored` is true, mention the file is git-ignored for privacy
4. **画像摘要**: Briefly summarize key traits identified

Example response:
```
✅ 用户画像已保存

**保存位置**: /path/to/project/.cursor/rules/user-profile.mdc
**生效方式**: 每次在此项目打开 Cursor 时自动加载

**画像摘要**:
- 编码风格: [简要描述]
- 技术栈: [主要技术]
- 沟通偏好: [偏好描述]

该文件已添加到 .gitignore，不会被提交到代码仓库。
```

## Output Locations

| Scope | File Location | Auto-loaded by Cursor |
|-------|---------------|----------------------|
| Global | `~/.cocursor/profiles/global.md` | Via project profile merge |
| Project | `{project}/.cursor/rules/user-profile.mdc` | Yes (alwaysApply: true) |

## Important Notes

1. **Privacy**: Profile is stored locally only, never uploaded to cloud
2. **Git Safety**: Project profiles are automatically added to `.gitignore`
3. **Idempotency**: If no new conversations since last analysis, skip regeneration (unless user forces)
4. **Incremental**: Merge with existing profile intelligently, don't replace entirely
5. **User Messages Only**: Only analyze user's messages, ignore AI responses

## Example Usage

User: "帮我分析一下我的用户画像"

**Complete response flow** (all steps required):
1. Call `mcp__cocursor__get_user_messages_for_profile` with `scope: "project"`, `project_path: "/current/project/path"`
2. Check `meta.needs_update` - if false and no force, ask user
3. Analyze messages across all dimensions
4. Generate structured Markdown profile
5. **MUST** call `mcp__cocursor__save_user_profile` with the generated content
6. **MUST** confirm to user with file location and summary

**Common mistake to avoid**: Do NOT stop after generating the profile text. MUST call save tool and confirm.

## MCP Tool Reference

```
mcp__cocursor__get_user_messages_for_profile
  scope (required): "global" | "project"
  project_path: string (required if scope is "project")
  days_back: int (default 30)
  
  Returns: { messages, stats: { ..., primary_language: "zh"|"en" }, existing_profile, meta }

mcp__cocursor__save_user_profile  
  scope (required): "global" | "project"
  project_path: string (required if scope is "project")
  content (required): string (Markdown, no frontmatter)
  language: "zh" | "en" (use stats.primary_language)
```
