---
name: openspec
description: Specification-driven development workflow tool for reaching consensus on requirements before coding. Use when working with OpenSpec workflows, creating change proposals, implementing approved changes, or archiving completed work.
---

# OpenSpec Skill

> **Requires**: cocursor daemon running at `localhost:19960`

OpenSpec helps teams reach consensus on requirements before coding through structured proposals and specs.

## Pre-check

```bash
curl -s http://localhost:19960/health
# Expected: {"status":"ok"}
```

## Workflow

### 1. Initialize

Create OpenSpec directory structure in project:

```
openspec/
├── project.md      # Project context
├── changes/        # Active change proposals
└── specs/          # Specification files
```

### 2. Create Proposal

List existing changes and specs first:

```bash
# macOS / Linux
curl -s "http://localhost:19960/api/v1/openspec/list?project_path=/path/to/project&type=all"

# Windows PowerShell
Invoke-RestMethod -Uri "http://localhost:19960/api/v1/openspec/list?project_path=C:/path/to/project&type=all"
```

Then create:
- `proposal.md` - Problem and solution
- `tasks.md` - Implementation tasks
- `design.md` - Technical design (if needed)
- `specs/` - Spec delta files

Validate format:

```bash
# macOS / Linux
curl -s -X POST "http://localhost:19960/api/v1/openspec/validate" \
  -H "Content-Type: application/json" \
  -d '{"project_path":"/path/to/project","change_id":"my-change","strict":true}'

# Windows PowerShell
$body = @{project_path="C:/path/to/project"; change_id="my-change"; strict=$true} | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:19960/api/v1/openspec/validate" -Method POST -ContentType "application/json" -Body $body
```

### 3. Apply Changes

After proposal approval:
1. Read change details via `/api/v1/openspec/list`
2. Complete tasks in `tasks.md` order
3. Mark tasks complete as you progress

### 4. Archive

After deployment:
1. Move change directory to `archive/`
2. Merge spec deltas into main spec files

## API Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/openspec/list` | GET | List changes and specs |
| `/api/v1/openspec/validate` | POST | Validate change format |

**List Parameters:**
- `project_path`: Project root path
- `type`: "changes", "specs", or "all" (default)

**Validate Parameters:**
- `project_path`: Project root path
- `change_id`: Change directory name
- `strict`: Enable strict validation (boolean)

## Change Structure

```
openspec/changes/{change-id}/
├── proposal.md     # Problem background, solution
├── tasks.md        # Implementation checklist
├── design.md       # Technical design (optional)
└── specs/          # Spec deltas
    └── {capability}/
        └── delta.md
```

## Validation Rules

Required files:
- `proposal.md` - Must contain "## Why" or "## Problem" section
- `tasks.md` - Implementation tasks

Spec delta files must contain:
- `## ADDED Requirements`, `## MODIFIED Requirements`, or `## REMOVED Requirements`
- At least one `#### Scenario:` per requirement
