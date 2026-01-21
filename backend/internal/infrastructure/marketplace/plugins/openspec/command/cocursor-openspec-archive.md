---
name: /cocursor-openspec-archive
id: cocursor-openspec-archive
category: OpenSpec
description: Archive a deployed OpenSpec change and update specs.
---
<!-- OPENSPEC:START -->
**Guardrails**
- Favor straightforward, minimal implementations first and add complexity only when it is requested or clearly required.
- Keep changes tightly scoped to the requested outcome.
- Refer to `openspec/AGENTS.md` (located inside the `openspec/` directory—run `ls openspec` or `openspec update` if you don't see it) if you need additional OpenSpec conventions or clarifications.

**Steps**
1. Determine the change ID to archive:
   - If this prompt already includes a specific change ID (for example inside a `<ChangeId>` block populated by slash-command arguments), use that value after trimming whitespace.
   - If the conversation references a change loosely (for example by title or summary), **call `mcp__cocursor__openspec_list`** to surface likely IDs, share the relevant candidates, and confirm which one the user intends.
   - Otherwise, review the conversation, **call `mcp__cocursor__openspec_list`**, and ask the user which change to archive; wait for a confirmed change ID before proceeding.
   - If you still cannot identify a single change ID, stop and tell the user you cannot archive anything yet.
2. Validate the change ID by **calling `mcp__cocursor__openspec_list`** and stop if the change is missing, already archived, or otherwise not ready to archive.
3. Move the change directory from `openspec/changes/<id>/` to `openspec/changes/archive/YYYY-MM-DD-<id>/`.
4. Merge spec deltas into the main spec files in `openspec/specs/` (unless `--skip-specs` flag is used for tooling-only work).
5. **Call `mcp__cocursor__openspec_validate`** with `strict: true` to validate the archived change.
6. **Call `mcp__cocursor__record_openspec_workflow`** to record: `stage: "archive", status: "completed"`.
7. Review the changes to confirm the target specs were updated and the change landed in `changes/archive/`.

**Reference**
- Use **`mcp__cocursor__openspec_list`** to confirm change IDs before archiving.
- Use **`mcp__cocursor__openspec_list`** with `type: "specs"` to inspect refreshed specs and address any validation issues.
<!-- OPENSPEC:END -->
