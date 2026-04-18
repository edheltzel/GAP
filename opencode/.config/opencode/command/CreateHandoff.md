---
description: "Create a session handoff doc in .atlas-plans/handoffs/ and dump the session to Recall so /clear is safe"
---

# Create Handoff

Capture the current session so it can be resumed cleanly after `/clear`. Writes a structured handoff document to `.atlas-plans/handoffs/` **and** dumps the session to the Recall database so nothing is lost when the context window is reset.

**Optional slug/title:** $ARGUMENTS

Use this when the context window is getting tight and you want to `/clear` without losing progress.

## Behavior

1. **Derive slug.** If `$ARGUMENTS` is provided, slugify it (kebab-case). Otherwise, infer a short slug from the current task under discussion (3–6 words, kebab-case). Compute today's date and the target path: `.atlas-plans/handoffs/YYYY-MM-DD-handoff-{slug}.md`.

2. **Ensure directories exist:**
   ```bash
   mkdir -p .atlas-plans/handoffs/archive
   ```

3. **Capture git state.** Run these and include the output in the handoff doc:
   ```bash
   git rev-parse --abbrev-ref HEAD
   git status --short
   git diff --stat
   git worktree list 2>/dev/null
   ```
   Do NOT run full `git diff` — only the stat summary. If the repo is dirty, warn the user in the final summary that they may want to commit or stash before `/clear`.

4. **Write the handoff document** using this exact structure:

   ```markdown
   # Handoff: {Human-readable title}

   **Date:** {YYYY-MM-DD}
   **Project:** {basename of cwd}
   **Branch:** {current branch}
   **Worktree:** {path if in worktree, else "main tree"}

   ## Task / Context
   {1–3 sentences: what is the active goal and why. Written so a fresh session
    can pick up cold — no shorthand, no "as discussed above".}

   ## Progress

   ### Done
   - {completed step 1}
   - {completed step 2}

   ### Next
   - {remaining step 1 — be specific: file paths, function names, exact changes}
   - {remaining step 2}

   ## Key Decisions
   - **{decision}** — {why}
   - **{decision}** — {why}

   ## Files Touched
   - `{path/to/file.ts}` — {one-line description of change}
   - `{path/to/other.md}` — {one-line description}

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
   /ReviewHandoff {filename}
   ```
   Or just `/ReviewHandoff` — it defaults to the newest handoff in `.atlas-plans/handoffs/`.
   ```

5. **Dump the session to Recall:**
   ```bash
   mem dump "Handoff: {Human-readable title}" --skip-fabric
   ```
   Use `--skip-fabric` for speed. For exceptionally valuable sessions, omit the flag to capture Fabric wisdom.

6. **Report to Ed** with:
   - The handoff file path
   - Confirmation that Recall dump succeeded
   - The exact resume command (`/ReviewHandoff {filename}`)
   - A warning if the git tree is dirty: recommend committing/stashing before `/clear`

## Notes

- The handoff doc is the source of truth for the next session; Recall is the backstop so ad-hoc details (chat exchanges, tangents, micro-decisions) remain searchable.
- Do not attempt to summarize the entire conversation — focus on what's needed to continue the work. The Recall dump captures everything else.
- If `.atlas-plans/handoffs/` doesn't exist yet, create it in step 2. Do the same for `archive/`.
