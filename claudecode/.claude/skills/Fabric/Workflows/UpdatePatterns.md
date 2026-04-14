# UpdatePatterns Workflow

## Trigger
"update fabric", "update patterns", "sync fabric", "pull patterns"

## Steps

### 1. Pull Latest Patterns from Upstream

Run the fabric-ai CLI to download fresh patterns from the danielmiessler/fabric GitHub repo:

```bash
fabric-ai -U
```

This clones the upstream repo and installs patterns to `~/.config/fabric/patterns/`.

### 2. Compare Config vs Skill Patterns

Identify new, updated, and removed patterns:

```bash
# New patterns (in config but not in skill)
comm -23 <(ls ~/.config/fabric/patterns/ | sort) <(ls ~/.claude/skills/Fabric/Patterns/ | sort)

# Skill-only patterns (custom or removed upstream)
comm -13 <(ls ~/.config/fabric/patterns/ | sort) <(ls ~/.claude/skills/Fabric/Patterns/ | sort)
```

### 3. Sync Patterns to Skill Directory

Use rsync to add new patterns and update existing ones without removing skill-only patterns:

```bash
# Add new patterns
rsync -av --ignore-existing ~/.config/fabric/patterns/ ~/.claude/skills/Fabric/Patterns/

# Update existing patterns with newer versions
rsync -av --update ~/.config/fabric/patterns/ ~/.claude/skills/Fabric/Patterns/
```

### 4. Report Results

Report:
- Total pattern count before and after
- Number of new patterns added
- Number of existing patterns updated
- List of skill-only patterns (custom/legacy)

## Notes

- The `fabric-ai` CLI is installed via Homebrew (`brew install fabric-ai`)
- Skill-only patterns are preserved (not deleted) during sync
- The upstream repo is: https://github.com/danielmiessler/fabric
- Patterns live in `~/.config/fabric/patterns/` (CLI managed) and `~/.claude/skills/Fabric/Patterns/` (skill managed)
