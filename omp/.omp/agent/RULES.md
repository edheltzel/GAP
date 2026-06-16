# Global OMP Rules

## Handoff Storage

When using the `/handoff` skill or creating a session handoff, always save the handoff markdown under the current project's `.agents/atlas/handoffs/` directory.

- Resolve the project root with `PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"`.
- Create the directory before writing: `mkdir -p "$PROJECT_ROOT/.agents/atlas/handoffs"`.
- Use a timestamped, descriptive Markdown filename, e.g. `YYYY-MM-DD-HHMMSS-handoff-<slug>.md`.
- Never write handoff docs to `/tmp`, OS temp folders, home-level scratch dirs, or the OMP/user config directory.
- Report the final handoff path.

## Roughdraft

Use Roughdraft when the user wants to review or comment on a Markdown file.

The user may refer to Roughdraft as `rd` in natural language. Treat `rd` as shorthand for Roughdraft in user requests, but do not create or modify any shell alias, executable, symlink, or command named `rd`.

When the user asks for a plan, write the plan as a Markdown file on disk before asking them to review it.

When you write or modify a Markdown file and want the user to review or comment on it, open it with:

```bash
roughdraft open "/absolute/path/to/file.md"
```

Roughdraft is currently a single-file Markdown viewer/editor. Open one `.md` file at a time.

If Roughdraft is not running, `roughdraft open` will start it automatically.

After `roughdraft open` opens the document, leave the command running. Do not interrupt, kill, background, detach, or treat the waiting process as cleanup. The wait is intentional: Roughdraft will exit the command after the user clicks Done Reviewing, and that exit is your signal to resume.

After the user finishes reviewing in Roughdraft, read the Markdown file from disk and respond to any CriticMarkup comments or suggested changes.

Use Roughdraft-flavored CriticMarkup when reading or writing inline review feedback in Markdown. The base markers are:

Comment: `{>>comment<<}`
Insertion: `{++new text++}`
Deletion: `{--old text--}`
Substitution: `{~~old~>new~~}`
Highlight: `{==text==}`

When you add a new comment or suggested change, use the extended Roughdraft format with a compact inline reference such as `{#c1}` or `{#s1}`, then add metadata in final YAML endmatter. Generate a stable document-local id (`c1`, `c2`, etc. for comments; `s1`, `s2`, etc. for suggestions), set `by` to your agent or author label, set `at` to the current ISO timestamp, and set `re` when replying to an existing comment or suggestion.

Roughdraft may already have inline attribute blocks after comments and suggestions from older documents. Preserve those attributes unless you are intentionally removing the associated comment or suggestion. For new feedback, prefer compact references plus YAML endmatter.

Anchored comments usually look like `{==selected text==}{>>Comment text<<}{#c1}`. Suggested changes usually look like `{++new text++}{#s1}` or `{~~old text~>new text~~}{#s2}`. Replies live in final YAML endmatter with a `body` and `re` pointer.

Example:

```markdown
{==selected text==}{>>Comment text<<}{#c1}
{++new text++}{#s1}

---
comments:
  c1:
    by: AI
    at: "2026-04-28T12:00:00.000Z"
  c2:
    body: I can make that edit.
    by: AI
    at: "2026-04-28T12:05:00.000Z"
    re: c1
suggestions:
  s1:
    by: AI
    at: "2026-04-28T12:10:00.000Z"
```

Use `roughdraft help` and `roughdraft help criticmarkup` for local command and syntax details.
