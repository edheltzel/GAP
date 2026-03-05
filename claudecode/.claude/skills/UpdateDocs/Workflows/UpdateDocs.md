# UpdateDocs Workflow

Update README.md and CLAUDE.md to reflect the current state of the project.

## Voice Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the UpdateDocs workflow to refresh project documentation"}' \
  > /dev/null 2>&1 &
```

Running the **UpdateDocs** workflow in the **UpdateDocs** skill to refresh project documentation...

## Step 1: Identify What Changed

Find changes since documentation was last updated. Run these commands **in parallel**:

```bash
# Find the last commit that touched docs
git log --oneline -1 -- README.md CLAUDE.md

# Show all commits since that docs commit (or last 20 if no docs commit exists)
git log --oneline <last-docs-commit>..HEAD

# Show files changed since last docs commit
git diff --stat <last-docs-commit>..HEAD

# Current project structure
ls -la
```

**If no prior docs commit exists:** Treat the entire project as new. Scan everything.

**If README.md or CLAUDE.md don't exist:** Skip to Step 5 (Generate from scratch).

## Step 2: Read Current Documentation

Read both files in full:

```bash
# Read existing docs
cat README.md
cat CLAUDE.md
```

Note which sections exist and their content. Identify:
- **Generated sections** (file trees, command tables, dependency lists) — safe to regenerate
- **Manual sections** (intro prose, philosophy, contributing guidelines, acknowledgments) — preserve exactly

## Step 3: Map Changes to Doc Sections

For each significant change from Step 1, determine which doc section it affects:

| Change Type | README Section | CLAUDE.md Section |
|-------------|---------------|-------------------|
| New file/directory | Project Structure | Project Structure, Key Files |
| New dependency | Installation, Prerequisites | Dependencies |
| New CLI command | Usage, Commands | Available Commands |
| Renamed/moved file | Project Structure, any path refs | All path references |
| New config option | Configuration | Configuration |
| Deleted feature | Remove from relevant sections | Remove from relevant sections |
| New API endpoint | API Reference | API Reference |
| Changed build process | Build, Development | Build Commands |

**If a change doesn't affect any doc section, skip it.**

## Step 4: Update Surgically

Edit only the sections identified in Step 3. For each section:

1. **Read the current section** to understand its format and style
2. **Match the existing style** — if the project uses bullet lists, use bullet lists; if it uses tables, use tables
3. **Edit with the Edit tool** — replace only the stale content, keep surrounding prose intact
4. **Never touch manual sections** — intro paragraphs, philosophy, contributing guides, and acknowledgments are hands-off unless they contain factual errors (wrong paths, dead commands)

### README.md Update Priorities

Update these sections (if they exist) in order of importance:

1. **Project structure / file tree** — regenerate from current state
2. **Installation / setup commands** — verify they still work
3. **Usage / CLI commands** — add new commands, remove dead ones
4. **Configuration** — reflect new options
5. **Dependencies / prerequisites** — update versions, add new ones

### CLAUDE.md Update Priorities

Update these sections (if they exist) in order of importance:

1. **Key file paths** — fix any stale paths
2. **Project structure** — regenerate from current state
3. **Build/test commands** — verify accuracy
4. **Architecture notes** — reflect structural changes
5. **Agent instructions** — update workflow guidance if project patterns changed

## Step 5: Generate Missing Docs

**If README.md doesn't exist,** generate one by scanning:

- `package.json` / `Cargo.toml` / `pyproject.toml` (name, description, scripts, dependencies)
- Project structure (key directories and their purpose)
- Config files (what's configurable)
- Existing code (entry points, exports)

**Structure for a new README.md:**
```
# Project Name
Brief description from package manifest.

## Setup
Install and configure.

## Usage
Key commands and workflows.

## Project Structure
File tree with annotations.

## Development
Build, test, lint commands.
```

**If CLAUDE.md doesn't exist,** generate one with:

```
# Project Name — Agent Instructions

## Overview
What this project does, in one paragraph.

## Key Files
Important files and what they do.

## Commands
Build, test, lint, deploy commands.

## Architecture
How the code is organized and key patterns.
```

## Step 6: Show Summary

Before committing, display a clear summary:

```
## Documentation Updated

**README.md:**
- Updated: Project Structure (added src/new-module/)
- Updated: Usage (added `bun run migrate` command)
- Preserved: Introduction, Contributing

**CLAUDE.md:**
- Updated: Key Files (added migration handler)
- Generated: (new file)

**Unchanged sections:** 4 in README, 0 in CLAUDE.md
```

## Step 7: Auto-Commit

Stage and commit the documentation changes:

```bash
git add README.md CLAUDE.md
```

Create a conventional commit with type `docs`:

```bash
git commit -m "$(cat <<'EOF'
docs: update README and CLAUDE.md

[Brief description of what changed and why]
EOF
)"
```

**Commit message rules:**
- Type is always `docs`
- Scope is optional — use it if only one file changed (e.g., `docs(readme):`)
- Description summarizes the substantive changes, not "updated docs"
- Body lists specific sections that changed

## Edge Cases

- **No changes since last docs update:** Report "Documentation is up to date" and exit
- **Only README.md exists, no CLAUDE.md:** Update README only, offer to generate CLAUDE.md
- **Only CLAUDE.md exists, no README.md:** Update CLAUDE.md only, offer to generate README
- **Monorepo:** Ask which package to update docs for
- **Uncommitted changes:** Warn the user and ask whether to proceed (docs may reference unstaged work)
