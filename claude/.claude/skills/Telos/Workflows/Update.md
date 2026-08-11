# Personal TELOS Update Compatibility Workflow

Personal TELOS mutation is disabled in the current native skill. This workflow is read-only and fail-closed.

## When a User Requests an Update

1. Resolve the requested filename only within `~/.claude/LIFEOS/USER/TELOS/`.
2. Confirm the file already exists. Do not create a missing personal-data tree or file.
3. Read the current file with the harness file tool.
4. Draft the smallest proposed patch in chat.
5. Explain exactly which file and section would change.
6. Ask for explicit approval before any separate manual edit.

Do not run `Tools/UpdateTelos.ts` to mutate data. The tool is a compatibility guard that exits nonzero for update attempts and writes nothing.

## Privacy and Integrity

- Never send personal TELOS content to an external service or model without explicit approval.
- Never append blindly, infer missing personal facts, or rewrite unrelated sections.
- Never claim a backup, changelog entry, or update occurred when no approved write was performed.
- Keep project TELOS output outside the personal TELOS directory.

## Failure Behavior

If the directory or requested file is absent, report that clearly and stop. If the desired change is ambiguous, ask one focused question. No failure path may create or modify personal data.
