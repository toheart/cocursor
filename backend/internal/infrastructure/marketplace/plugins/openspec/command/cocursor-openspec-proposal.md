---
name: /cocursor-openspec-proposal
id: cocursor-openspec-proposal
category: OpenSpec
description: Scaffold a new OpenSpec change and validate strictly.
---
<!-- OPENSPEC:START -->
**Guardrails**
- Favor straightforward, minimal implementations first and add complexity only when it is requested or clearly required.
- Keep changes tightly scoped to the requested outcome.
- Refer to `openspec/AGENTS.md` (located inside the `openspec/` directory—run `ls openspec` or `openspec update` if you don't see it) if you need additional OpenSpec conventions or clarifications.
- Identify any vague or ambiguous details and ask the necessary follow-up questions before editing files.
- Do not write any code during the proposal stage. Only create design documents (proposal.md, tasks.md, design.md, and spec deltas). Implementation happens in the apply stage after approval.

**Steps**
1. Review `openspec/project.md` to understand project context.
2. **Call cocursor MCP tool `openspec_list`** with parameters:
   - `project_path`: current workspace path (get from context or ask user)
   - `type`: "all"
   This will return current changes and specs in JSON format. Note any gaps that require clarification.
3. Inspect related code or docs (e.g., via `rg`/`ls`) to ground the proposal in current behaviour.
4. Choose a unique verb-led `change-id` (check against existing changes from step 2) and scaffold `proposal.md`, `tasks.md`, and `design.md` (when needed) under `openspec/changes/<id>/`.
5. Map the change into concrete capabilities or requirements, breaking multi-scope efforts into distinct spec deltas with clear relationships and sequencing.
6. Capture architectural reasoning in `design.md` when the solution spans multiple systems, introduces new patterns, or demands trade-off discussion before committing to specs.
7. Draft spec deltas in `changes/<id>/specs/<capability>/spec.md` (one folder per capability) using `## ADDED|MODIFIED|REMOVED Requirements` with at least one `#### Scenario:` per requirement and cross-reference related capabilities when relevant.
8. Draft `tasks.md` as an ordered list of small, verifiable work items that deliver user-visible progress, include validation (tests, tooling), and highlight dependencies or parallelizable work.
9. **Call cocursor MCP tool `openspec_validate`** with parameters:
   - `project_path`: current workspace path
   - `change_id`: the change ID created
   - `strict`: true
   This will return validation results in JSON format.
10. If validation fails, fix issues based on the validation errors and re-validate until all issues are resolved.
11. **Call cocursor MCP tool `record_openspec_workflow`** to record state:
    - `project_path`: current workspace path
    - `change_id`: the change ID
    - `stage`: "proposal"
    - `status`: "completed"
12. **Ask user**: "提案已创建。是否还需要讨论以下问题？"
    - Extract "Open Questions" section from `proposal.md` if present
    - List any ambiguities or decisions needed
    - Wait for user confirmation before proceeding

**Reference**
- Use **cocursor MCP tool `openspec_list`** with `type: "specs"` to search existing requirements when validation fails.
- Search existing requirements with `rg -n "Requirement:|Scenario:" openspec/specs` before writing new ones.
- Explore the codebase with `rg <keyword>`, `ls`, or direct file reads so proposals align with current implementation realities.
<!-- OPENSPEC:END -->
