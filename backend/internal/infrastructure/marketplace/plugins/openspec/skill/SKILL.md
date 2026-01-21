---
name: openspec
description: Specification-driven development workflow tool that helps teams and AI assistants reach consensus on requirements before coding. Provides complete OpenSpec workflow support through cocursor MCP tools. Use this skill when working with OpenSpec workflows, creating change proposals, implementing approved changes, or archiving completed work.
---

# OpenSpec Skill

OpenSpec is a specification-driven development workflow tool that helps teams and AI assistants reach consensus on requirements before coding.

## Workflow

### 1. Initialize (Init)
Use the `/cocursor-openspec-init` command to initialize the OpenSpec directory structure.

### 2. Create Proposal (Proposal)
Use the `/cocursor-openspec-proposal` command to create a change proposal. This command will:
- Call `mcp__cocursor__openspec_list` to get existing changes and specs
- Create proposal.md, tasks.md, design.md (if needed)
- Create spec delta files
- Call `mcp__cocursor__openspec_validate` to validate format
- Call `mcp__cocursor__record_openspec_workflow` to record state

### 3. Apply Changes (Apply)
Use the `/cocursor-openspec-apply` command to implement approved changes. This command will:
- Call `mcp__cocursor__openspec_list` to get change details
- Complete tasks in tasks.md in order
- Call `mcp__cocursor__record_openspec_workflow` to update progress after each task completion
- Automatically call `mcp__cocursor__generate_openspec_workflow_summary` to generate work summary when all tasks are completed

### 4. Archive Changes (Archive)
Use the `/cocursor-openspec-archive` command to archive deployed changes. This command will:
- Move change directory to archive/
- Merge spec deltas into main spec files
- Call `mcp__cocursor__record_openspec_workflow` to record archive state

## Cocursor MCP Tools

> **MCP Server Dependency**: This skill requires the `cocursor` MCP server.

All OpenSpec operations are performed through cocursor MCP tools (use full names when calling):

- `mcp__cocursor__openspec_list` - List changes and specs (returns JSON format)
- `mcp__cocursor__openspec_validate` - Validate change format
- `mcp__cocursor__record_openspec_workflow` - Record workflow state (automatically detects tasks.md completion)
- `mcp__cocursor__generate_openspec_workflow_summary` - Generate work summary
- `mcp__cocursor__get_openspec_workflow_status` - Get workflow status

## Workflow State Tracking

Workflow state for each stage is automatically recorded to the `~/.cocursor/cocursor.db` database, including:
- Current stage (init/proposal/apply/archive)
- Status (in_progress/completed/paused)
- Progress information (task completion status)
- Work summary (automatically generated when tasks.md is completed)

## Usage Instructions

1. After installing the OpenSpec plugin, Command files are automatically installed to the workspace's `.cursor/commands/` directory
2. Use `/cocursor-openspec-*` commands to perform OpenSpec workflow operations
3. All operations call cocursor MCP tools and return structured JSON data
4. Workflow state is automatically tracked and can be viewed in the UI (to be implemented)
