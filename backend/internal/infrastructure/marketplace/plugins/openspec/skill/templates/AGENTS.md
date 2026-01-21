# OpenSpec Instructions

Instructions for AI coding assistants using OpenSpec for spec-driven development.

## TL;DR Quick Checklist

- Search existing work: Call `mcp__cocursor__openspec_list` with `type: "all"` to get current changes and specs
- Decide scope: new capability vs modify existing capability
- Pick a unique `change-id`: kebab-case, verb-led (`add-`, `update-`, `remove-`, `refactor-`)
- Scaffold: `proposal.md`, `tasks.md`, `design.md` (only if needed), and delta specs per affected capability
- Write deltas: use `## ADDED|MODIFIED|REMOVED|RENAMED Requirements`; include at least one `#### Scenario:` per requirement
- Validate: Call `mcp__cocursor__openspec_validate` with `strict: true` and fix issues
- Request approval: Do not start implementation until proposal is approved

## Three-Stage Workflow

### Stage 1: Creating Changes
Create proposal when you need to:
- Add features or functionality
- Make breaking changes (API, schema)
- Change architecture or patterns  
- Optimize performance (changes behavior)
- Update security patterns

**Workflow**
1. Review `openspec/project.md` to understand project context.
2. Call `mcp__cocursor__openspec_list` with `type: "all"` to get current changes and specs.
3. Choose a unique verb-led `change-id` and scaffold `proposal.md`, `tasks.md`, optional `design.md`, and spec deltas under `openspec/changes/<id>/`.
4. Draft spec deltas using `## ADDED|MODIFIED|REMOVED Requirements` with at least one `#### Scenario:` per requirement.
5. Call `mcp__cocursor__openspec_validate` with `change_id` and `strict: true` to validate.
6. Call `mcp__cocursor__record_openspec_workflow` to record state: `stage: "proposal", status: "completed"`.
7. Ask user if there are any open questions or discussion points.

### Stage 2: Implementing Changes
Track these steps as TODOs and complete them one by one.
1. **Read proposal.md** - Understand what's being built
2. **Read design.md** (if exists) - Review technical decisions
3. **Read tasks.md** - Get implementation checklist
4. **Call `mcp__cocursor__record_openspec_workflow`** to record: `stage: "apply", status: "in_progress"`
5. **Implement tasks sequentially** - Complete in order
6. After each task completion, update `tasks.md` and call `mcp__cocursor__record_openspec_workflow` to update progress
7. When all tasks are completed, `mcp__cocursor__record_openspec_workflow` will automatically detect completion and generate summary
8. **Update checklist** - After all work is done, set every task to `- [x]` so the list reflects reality

### Stage 3: Archiving Changes
After deployment:
- Move `changes/[name]/` → `changes/archive/YYYY-MM-DD-[name]/`
- Update `specs/` if capabilities changed
- Call `mcp__cocursor__record_openspec_workflow` to record: `stage: "archive", status: "completed"`

## Cocursor MCP Tools

> **MCP Server Dependency**: This skill requires the `cocursor` MCP server.

All OpenSpec operations use cocursor MCP tools (use full names when calling):

- `mcp__cocursor__openspec_list` - List changes and specs (returns JSON)
- `mcp__cocursor__openspec_validate` - Validate change format
- `mcp__cocursor__record_openspec_workflow` - Record workflow state (auto-detects task completion)
- `mcp__cocursor__generate_openspec_workflow_summary` - Generate workflow summary
- `mcp__cocursor__get_openspec_workflow_status` - Get workflow status

## Spec File Format

### Critical: Scenario Formatting

**CORRECT** (use #### headers):
```markdown
#### Scenario: User login success
- **WHEN** valid credentials provided
- **THEN** return JWT token
```

Every requirement MUST have at least one scenario.

### Delta Operations

- `## ADDED Requirements` - New capabilities
- `## MODIFIED Requirements` - Changed behavior
- `## REMOVED Requirements` - Deprecated features
- `## RENAMED Requirements` - Name changes
