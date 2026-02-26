# Atlas Config — Agent Instructions

## For Agents

If you are an AI agent working in this repo, read the following files for full system context:

1. **`claudecode/.claude/skills/PAI/SKILL.md`** — The complete PAI system documentation including the Algorithm, capabilities, and execution rules.
2. **`claudecode/.claude/CLAUDE.md`** — Context injection and session initialization.
3. **`claudecode/.claude/settings.json`** — Identity configuration, hooks, and permissions.

Do not modify files in `claudecode/.claude/MEMORY/` without understanding the memory system (documented in `SKILL.md`).

## Local Modifications (PAIUpdate Protection)

The following files contain **local customizations** that must NOT be overwritten during a PAI update (`/pai-upgrade`). When pulling upstream PAI changes, preserve these modifications:

| File                                    | What Was Customized                                                                                                                                                                                                                                                                                                                                   |
| --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `claudecode/.claude/skills/PAI/Tools/Banner.ts`    | Fully rewritten with Eldritch theme. WezTerm-native width detection (`wezterm cli list`). Animated asciigen.art planet frames — **high quality**, downsampled to 1/3 size (14 rows × ~39 cols, ROW_STEP=3 COL_STEP=3), single gray, 50 consecutive frames starting at frame 90 (North America). Falls back to raycast sphere when asciify dir absent. |
| `claudecode/.claude/skills/PAI/Tools/pai.ts`       | Added `--dangerously-skip-permissions` flag to `cmdLaunch()`. Removed hardcoded voice notification (handled by async hook instead).                                                                                                                                                                                                                   |
| `claudecode/.claude/hooks/StartupGreeting.hook.ts` | Removed banner display from hook (banner only shows via `pai.ts` CLI path). Kitty env persistence code is inactive (Ed uses WezTerm).                                                                                                                                                                                                                 |
| `claudecode/.claude/hooks/VoiceGreeting.hook.ts`   | **New file** — async SessionStart hook that speaks the startup catchphrase for ALL launch paths (`pai`, `atlas`, `claude`).                                                                                                                                                                                                                           |
| `claudecode/.claude/settings.json`                 | Registered `VoiceGreeting.hook.ts` as an async SessionStart hook in the hooks section. `statusLine.command` points to `atlas-statusline.sh` (not upstream `statusline-command.sh`). `plansDirectory` set to `./.atlas-plans` (project-local, Atlas-branded; avoids tilde-expansion bug).                                                              |
| `claudecode/.claude/atlas-statusline.sh`           | **User-owned statusline** — full copy of upstream `statusline-command.sh` with color palette remapped to Eldritch theme (`#04d1f9` cyan, `#37f499` green, `#a48cf2` purple). Do NOT overwrite with upstream during PAIUpdate; instead diff and merge the logic sections while keeping the COLOR PALETTE block.                                        |
| `~/.zshrc`                              | Added `atlas` alias alongside `pai` alias.                                                                                                                                                                                                                                                                                                            |

**During PAIUpdate:** If upstream changes touch any of these files, merge carefully — do not blindly replace. The voice greeting architecture depends on the hook being async and `pai.ts` NOT having its own voice call. For `atlas-statusline.sh`: only merge changes to sections OTHER than `# COLOR PALETTE` — that block is intentionally Eldritch-themed.

## For Humans

See **[README.md](README.md)** for a human-friendly overview of PAI, Atlas, all available skills, and setup instructions.
