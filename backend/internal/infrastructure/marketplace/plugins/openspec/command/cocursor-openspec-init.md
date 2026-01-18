---
name: /cocursor-openspec-init
id: cocursor-openspec-init
category: OpenSpec
description: Initialize OpenSpec in the current workspace.
---

<!-- OPENSPEC:START -->
**Steps**
1. Get the current workspace path (from context or ask user)
2. Check if `openspec/` directory exists
3. If not exists, create directory structure:
   - `openspec/specs/`
   - `openspec/changes/`
   - `openspec/changes/archive/`
4. Copy template files from `~/.claude/skills/openspec/templates/`:
   - `AGENTS.md` → `openspec/AGENTS.md` (if not exists)
   - `project.md` → `openspec/project.md` (if not exists)
5. **If `project.md` was just created or contains placeholder text** (e.g., `[Describe the purpose of your project`, `[Describe your technology stack]`, etc.):
   - Analyze the current project structure (read README.md, package.json, go.mod, Cargo.toml, requirements.txt, etc.)
   - Read existing code to understand architecture patterns and conventions
   - Generate a filled-out `project.md` based on the project context, including:
     - **Purpose**: Project mission and goals
     - **Tech Stack**: All technologies, frameworks, and tools used
     - **Project Conventions**:
       - Code Style (naming, formatting, linting rules)
       - Architecture Patterns (folder structure, design patterns, layer separation)
       - Testing Strategy (test types, frameworks, coverage requirements)
       - Git Workflow (branching strategy, commit conventions, PR process)
     - **Domain Context** (if applicable): Domain-specific knowledge and terminology
     - **Important Constraints**: Technical, business, or performance constraints
     - **External Dependencies**: Third-party services, APIs, libraries, runtime requirements
   - Ask user to review and confirm the generated content before finalizing
6. Update root `AGENTS.md`:
   - Read `AGENTS.md` (if exists)
   - Add OPENSPEC:START/END block if not present
   - Create `AGENTS.md` if not exists
7. **Call cocursor MCP tool `record_openspec_workflow`** to record: `stage: "init", status: "completed"`
8. Display success message with next steps
<!-- OPENSPEC:END -->
