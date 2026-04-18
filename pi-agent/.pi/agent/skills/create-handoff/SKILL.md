---
name: create-handoff
description: Create a session handoff doc in .atlas/handoffs/ and dump the session to Recall so /clear is safe. Use when context is running low and you want to resume cleanly in a new session. Invoke directly with /skill:create-handoff [optional-slug].
allowed-tools: Read, Write, Bash(git:*), Bash(mkdir:*), Bash(mem:*), Bash(date:*)
---

# Create Handoff

Capture the current session so it can be resumed cleanly after `/clear`. Writes a structured handoff document to `.atlas/handoffs/` **and** dumps the session to the Recall database so nothing is lost when the context window is reset.

Arguments are appended as user input after this skill loads. If the user supplied text after `/skill:create-handoff`, treat it as a slug/title hint. If not, infer a short slug from the current task (3–6 words, kebab-case).

## Behavior

1. **Derive slug.** If user supplied args, slugify them (lowercase, kebab-case, strip punctuation). Otherwise infer from current task context. Compute today's date via `date +%Y-%m-%d` and the target path: `.atlas/handoffs/YYYY-MM-DD-handoff-{slug}.md`.

2. **Ensure directories exist:**
   ```bash
   mkdir -p .atlas/handoffs/archive
   ```

3. **Capture git state.** Run these and include the output in the handoff doc:
   ```bash
   git rev-parse --abbrev-ref HEAD
   git status --short
   git diff --stat
   git worktree list 2>/dev/null
   ```
   Do NOT run full `git diff` — only the stat summary. If the repo is dirty, warn the user in the final summary that they may want to commit or stash before `/clear`.

4. **Write the handoff document** using this structure:

   ```markdown
   # Handoff: {Human-readable title}

   **Date:** {YYYY-MM-DD}
   **Project:** {basename of cwd}
   **Branch:** {current branch}
   **Worktree:** {path if in worktree, else "main tree"}

   ## Task / Context
   {1–3 sentences: active goal and why. Written so a fresh session can
    pick up cold — no shorthand, no "as discussed above".}

   ## Progress

   ### Done
   - {completed step 1}
   - {completed step 2}

   ### Next
   - {remaining step 1 — specific: file paths, function names, exact changes}
   - {remaining step 2}

   ## Key Decisions
   - **{decision}** — {why}
   - **{decision}** — {why}

   ## Files Touched
   - `{path/to/file.ts}` — {one-line description of change}

   ## Git State
   ```
   Branch: {branch}
   Status:
   {git status --short output}

   Diff stat:
   {git diff --stat output}
   ```

   ## Open Questions / Blockers
   - {question needing Ed's decision}
   - {blocker that stopped progress}

   (If none: write "None.")

   ## Resume
   After `/clear`, run:
   ```
   /skill:review-handoff {filename}
   ```
   Or just `/skill:review-handoff` — it defaults to the newest handoff in `.atlas/handoffs/`.
   ```

5. **Dump the session to Recall:**
   ```bash
   mem dump "Handoff: {Human-readable title}" --skip-fabric
   ```
   Use `--skip-fabric` for speed. For exceptionally valuable sessions, omit the flag.

6. **Report to Ed** with:
   - The handoff file path
   - Confirmation that Recall dump succeeded
   - The exact resume command (`/skill:review-handoff {filename}`)
   - A warning if the git tree is dirty: recommend committing/stashing before `/clear`

## Notes

- Handoff doc = source of truth for next session. Recall dump = backstop for ad-hoc details.
- Do not summarize the entire conversation — focus on what's needed to continue. The Recall dump captures the rest.
- Create `.atlas/handoffs/` and `archive/` if they don't yet exist.
