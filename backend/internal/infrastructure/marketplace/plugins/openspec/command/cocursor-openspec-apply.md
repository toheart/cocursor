---
name: /cocursor-openspec-apply
id: cocursor-openspec-apply
category: OpenSpec
description: Implement an approved OpenSpec change and keep tasks in sync.
---
<!-- OPENSPEC:START -->
**Guardrails**
- **MUST** follow `openspec/project.md` conventions and Governance 宪章 (Hard Constraints).
- Favor straightforward, minimal implementations first and add complexity only when it is requested or clearly required.
- Keep changes tightly scoped to the requested outcome.
- Refer to `openspec/AGENTS.md` (located inside the `openspec/` directory—run `ls openspec` or `openspec update` if you don't see it) if you need additional OpenSpec conventions or clarifications.

**Steps**
Track these steps as TODOs and complete them one by one.
1. **Call `mcp__cocursor__openspec_list`** to get change details and confirm the change ID.
2. Read `changes/<id>/proposal.md`, `design.md` (if present), and `tasks.md` to confirm scope and acceptance criteria.
3. **Call `mcp__cocursor__record_openspec_workflow`** to record: `stage: "apply", status: "in_progress"`.
4. Work through tasks sequentially, keeping edits minimal and focused on the requested change.
5. After each task completion:
   - Update `tasks.md` status (mark task as `- [x]`)
   - **Call `mcp__cocursor__record_openspec_workflow`** to update progress in metadata:
     - `metadata.tasks_completed`: number of completed tasks
     - `metadata.tasks_total`: total number of tasks
     - `metadata.current_task`: current task identifier
6. Confirm completion before updating statuses—make sure every item in `tasks.md` is finished.
7. When all tasks are completed:
   - **Call `mcp__cocursor__record_openspec_workflow`** will automatically detect completion and call `generate_openspec_workflow_summary`
   - The workflow status will be updated to `completed` with summary
   - Display summary to user
8. Reference **`mcp__cocursor__openspec_list`** when additional context is required.

**Reference**
- Use **`mcp__cocursor__openspec_list`** with specific change ID to get detailed context if needed.
<!-- OPENSPEC:END -->
