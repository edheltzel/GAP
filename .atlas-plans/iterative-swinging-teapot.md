# Stow Restructure: claudecode + opencode packages

## Context

The repo treats `atlas-config` as a single stow package (`stow -d ~/Developer -t ~ atlas-config`). This means `opencode/.config/opencode/` stows to `~/opencode/.config/opencode/` (wrong) instead of `~/.config/opencode/`. We need multi-package stow: each subdirectory becomes its own package with the repo root as STOW_DIR.

## Critical Research Findings

### Stow `.stow-local-ignore` REPLACES defaults — does NOT extend them

When a `.stow-local-ignore` exists in a package directory, GNU Stow's **entire built-in default ignore list is discarded**. No merging. This means every per-package ignore file must re-include stow defaults we still want (`.git`, `.gitignore`, emacs backups, etc.).

**Stow's built-in defaults (must be re-included):**
```
\.git
\.gitignore
\.gitmodules
.+~
\#.*\#
^/README.*
^/LICENSE.*
^/COPYING
```

### Pattern matching rules
- Patterns **without `/`** → match basename at ANY depth (e.g., `node_modules` catches all `node_modules/` everywhere)
- Patterns **with `/`** → anchored path match (e.g., `^/README.*` only matches README at package root)
- `.stow-local-ignore` goes **inside each package directory**, not the stow dir root

### OpenCode runtime files
All contents of `~/.config/opencode/` are needed at runtime: agents, commands, skills, plugins, themes, tools. `node_modules/`, `package.json`, `bun.lock` are dev artifacts — opencode installs plugins from `opencode.jsonc` declarations directly.

### Superpowers: managed independently, NOT stowed
Superpowers is a plugin for both opencode AND Claude Code. Rather than stowing it from atlas-config, each coding agent manages its own superpowers installation. The existing symlinks in `plugins/superpowers.js` and `skills/superpowers` already use **absolute paths** to `~/.config/opencode/superpowers/`, so they work independently of stow. Plan: remove superpowers from atlas-config git tracking, add to `.gitignore`, and add to opencode's `.stow-local-ignore`.

## Target Structure

```
atlas-config/              (STOW_DIR — repo root, not a package)
├── claudecode/            (stow package)
│   ├── .claude/           → stows to ~/.claude
│   └── .stow-local-ignore
├── opencode/              (stow package — already has correct internal structure)
│   ├── .config/opencode/  → stows to ~/.config/opencode
│   └── .stow-local-ignore
├── makefile               (updated for multi-package)
├── CLAUDE.md
├── .gitignore             (paths updated for claudecode/)
└── .stow-local-ignore     (inert — repo root is STOW_DIR, not a package)
```

## Steps

### 1. Unlink current stow
```bash
make unlink    # remove existing symlinks before restructuring
```

### 2. Create `claudecode/` and move `.claude` into it
```bash
mkdir claudecode
git mv .claude claudecode/.claude
```

### 3. Create per-package `.stow-local-ignore` files

**`claudecode/.stow-local-ignore`:**
```
# Stow defaults (REQUIRED — this file replaces built-in defaults)
\.git
\.gitignore
\.gitmodules
\.gitattributes
.+~
\#.*\#

# macOS
\.DS_Store

# Dependencies
node_modules

# Templates (not needed at runtime)
\.env\.example
```

**`opencode/.stow-local-ignore`:**
```
# Stow defaults (REQUIRED — this file replaces built-in defaults)
\.git
\.gitignore
\.gitmodules
\.gitattributes
.+~
\#.*\#

# macOS
\.DS_Store

# Dependencies (opencode installs plugins from opencode.jsonc directly)
node_modules
bun\.lock
package\.json

# Superpowers — managed independently, not stowed
# Each coding agent (opencode, Claude Code) manages its own superpowers installation.
# The plugins/superpowers.js and skills/superpowers symlinks use absolute paths
# to ~/.config/opencode/superpowers/ and work independently of stow.
superpowers
```

### 4. Remove superpowers from git tracking
```bash
# Stop tracking superpowers in atlas-config (it has its own .git)
git rm -r --cached opencode/.config/opencode/superpowers/
```
The directory stays on disk — just no longer tracked by atlas-config.

After stow creates `~/.config/opencode/`, superpowers needs to be cloned/present at `~/.config/opencode/superpowers/`. The existing symlinks (`plugins/superpowers.js`, `skills/superpowers`) already point there via absolute paths.

### 5. Update root `.stow-local-ignore`
The root file becomes **inert** in the new setup (repo root = STOW_DIR, not a package). Keep it as documentation or remove it. It has no effect on stow operations.

### 6. Update Makefile

```makefile
STOW_DIR := $(CURDIR)
STOW_TARGET := $(HOME)
STOW_CMD := stow -d $(STOW_DIR) -t $(STOW_TARGET)
PKGS := claudecode opencode

YELLOW := \033[33m
GREEN := \033[32m
WHITE := \033[37m
CYAN := \033[36m
CLR := \033[0m

.PHONY: default
default: help

.PHONY: help
help: ## Show this help message (default)
	@awk 'BEGIN {FS = ":.*?## "}; \
		/^[^\t][a-zA-Z0-9_-]+:.*?##/ \
		{ printf "\033[36m%-24s$(CLR) %s\n", $$1, $$2 } \
		/^##/ { printf "$(YELLOW)%s$(CLR)\n", substr($$0, 4) }' $(MAKEFILE_LIST)

.PHONY: up update
up: update
update: ## Restow all packages (use after adding/removing files)
	@for pkg in $(PKGS); do $(STOW_CMD) --restow $$pkg; done
	@echo "$(GREEN)All packages restowed$(CLR)"

.PHONY: link
link: ## Create symlinks (first time setup)
	@for pkg in $(PKGS); do $(STOW_CMD) $$pkg; done
	@echo "$(GREEN)Packages linked: $(PKGS)$(CLR)"

.PHONY: unlink
unlink: ## Remove all symlinks
	@for pkg in $(PKGS); do $(STOW_CMD) --delete $$pkg; done
	@echo "$(WHITE)All package symlinks removed$(CLR)"

.PHONY: status
status: ## Show what stow would do (dry run)
	@for pkg in $(PKGS); do \
		echo "$(CYAN)Package: $$pkg$(CLR)"; \
		$(STOW_CMD) --restow -n -v $$pkg 2>&1 || true; \
		echo ""; \
	done

.PHONY: push
push: ## Commit and push config changes
	@git add -A
	@git diff --cached --quiet && echo "$(YELLOW)Nothing to commit$(CLR)" || \
		(git commit -m "Update config" && git push origin master && echo "$(GREEN)Pushed to origin$(CLR)")
```

### 7. Update `.gitignore`
Path changes for items that moved into `claudecode/`:
```
.claude/MEMORY/**     →  claudecode/.claude/MEMORY/**
!.claude/MEMORY/...   →  !claudecode/.claude/MEMORY/...
.claude/pai_updates/  →  claudecode/.claude/pai_updates/
.claude/pai_backups/  →  claudecode/.claude/pai_backups/
.claude/.pai-sync-history → claudecode/.claude/.pai-sync-history
.claude/.sidecar/     →  claudecode/.claude/.sidecar/
.claude/Observability/MenuBarApp/Observability.app/ → claudecode/.claude/...
```
Add superpowers to gitignore (managed independently):
```
# Superpowers plugin (managed independently by each coding agent)
opencode/.config/opencode/superpowers/
```

### 8. Update `CLAUDE.md`
Update references from `.claude/` to `claudecode/.claude/` where they refer to repo-relative paths (e.g., instructions telling agents to read `.claude/skills/PAI/SKILL.md`).

**Note:** `settings.json` `contextFiles` paths are relative to `.claude/` itself (e.g., `skills/PAI/SKILL.md`), NOT to the repo root. These do NOT need updating since they resolve from wherever `.claude/` is found.

### 9. Relink with new structure
```bash
make link      # create new symlinks
make status    # dry run to verify
```

## What Does NOT Break

| Concern | Why It's Safe |
|---------|---------------|
| `~/.claude` discovery | Claude Code walks up from CWD; finds `~/.claude` (stow symlink) at home level |
| `settings.json` contextFiles | Paths like `skills/PAI/SKILL.md` are relative to `.claude/`, not repo root |
| Hook `${PAI_DIR}` references | `PAI_DIR=/Users/ed/.claude` — the stow target, unchanged |
| `.atlas-plans/` | Stays at repo root; `plansDirectory: "./.atlas-plans"` is CWD-relative |
| `opencode.jsonc` | Already inside `.config/opencode/` — correct stow target path |

## Verification

```bash
# 1. Dry run — confirm stow would create correct symlinks
make status

# 2. Verify ~/.claude exists and points to correct location
ls -la ~/.claude

# 3. Verify ~/.config/opencode exists
ls -la ~/.config/opencode

# 4. Verify Claude Code finds settings
cat ~/.claude/settings.json | head -5

# 5. Verify opencode config
ls ~/.config/opencode/opencode.jsonc
```

## Critical Files Modified
- `makefile` — rewrite for multi-package stow
- `claudecode/.stow-local-ignore` — new file
- `opencode/.stow-local-ignore` — new file
- `.gitignore` — path updates for `claudecode/` prefix
- `CLAUDE.md` — path updates for `claudecode/` prefix
