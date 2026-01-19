# Tech Debt Extraction Patterns

Extract tech debt and follow-up plans from conversations, focusing on the following patterns:

## Deferred Markers

Look for phrases indicating deferred work:
- "Will do this later..."
- "Temporary solution..."
- "Will optimize later..."
- "TODO", "FIXME", "XXX" markers
- "Need to refactor this"
- "Should improve this"

## Temporary Solutions

Identify temporary or incomplete implementations:
- "Temporary handling"
- "Leave it as is for now"
- "Will improve later"
- "Quick fix for now"
- "Hack to make it work"
- "Workaround"

## Known Issues

Extract acknowledged problems that need attention:
- "Known issue"
- "To be fixed"
- "Needs optimization"
- "Performance issue"
- "Security concern"
- "Technical debt"

## Extraction Guidelines

When extracting tech debt, include:
- **Problem description**: Clear description of what needs to be addressed
- **Associated session ID**: Link to the conversation where it was mentioned
- **Project**: The project it belongs to
- **Priority indicators**: If mentioned (e.g., "high priority", "critical")

## Example Patterns

```
User: "I'll optimize this query later, it's working for now"
→ Tech Debt: "Optimize database query (mentioned in session XYZ)"
```

```
User: "This is a temporary solution, we should refactor it"
→ Tech Debt: "Refactor temporary implementation (mentioned in session XYZ)"
```

```
AI: "TODO: Add error handling here"
→ Tech Debt: "Add error handling (TODO marker in session XYZ)"
```
