# Zellij Keymaps & Status Bar Plan

**Goal:** Bring Zellij in line with Wezterm/TMUX — `Ctrl+K` leader key, rich status bar showing session, CWD, git branch, mode, and keybinding hints.

---

## Part 1: Leader Key (`Ctrl+K` → tmux mode)

### Mechanism

Zellij doesn't have a "leader key" concept like TMUX/Wezterm. Instead, it uses **mode switching**. The `tmux` mode is the closest equivalent — press a key to enter it, then press an action key, and it returns to `normal`.

**Change:** Rebind `Ctrl+K` to enter `tmux` mode (currently `Ctrl+B`).

```kdl
// In config.kdl keybinds section:
shared_except "locked" "scroll" "search" "tmux" {
    unbind "Ctrl b"
    bind "Ctrl k" { SwitchToMode "tmux"; }
}
```

### Ctrl+K Conflict Mitigation

`Ctrl+K` is used by:
- **Fish shell:** Kill from cursor to end of line (readline `kill-line`)
- **Neovim/Vim:** Various plugin bindings (LSP signature help, etc.)

**Mitigation:** Since tmux mode is a **mode switch** (not a timeout-based prefix), pressing `Ctrl+K` in Zellij will always enter tmux mode. To pass `Ctrl+K` through to the terminal app:

1. **Option A — Passthrough binding in tmux mode:** Add `bind "Ctrl k" { Write 11; SwitchToMode "normal"; }` to the `tmux` block. Double-tap `Ctrl+K` sends the literal keystroke. This matches TMUX's `send-prefix` pattern.
2. **Option B — Use locked mode:** `Ctrl+G` enters locked mode where all keys pass through. Use this when you need heavy `Ctrl+K` usage in an app.

**Recommendation:** Option A (double-tap passthrough). It's the same UX as TMUX's `C-k C-k` to send literal.

### Keybinding Mapping (Wezterm/TMUX → Zellij)

| Action | Wezterm (`Cmd+K` then) | TMUX (`C-k` then) | Zellij (`C-k` then) |
|--------|------------------------|---------------------|----------------------|
| **Clear screen** | `k` | `k` | `k` → `Write 12; SwitchToMode "normal"` (sends Ctrl+L) |
| **Split right** | `\` or `l` | `\` or `l` | `\` or `l` → `NewPane "right"; SwitchToMode "normal"` |
| **Split down** | `-` or `j` | `-` or `j` | `-` or `j` → `NewPane "down"; SwitchToMode "normal"` |
| **Split left** | `h` | `h` | `h` → *(no native split-left in Zellij — use NewPane + MovePane)* |
| **Split up** | `u` | `u` | `u` → *(no native split-up — use NewPane + MovePane)* |
| **Close pane** | `x` | `x` | `x` → `CloseFocus; SwitchToMode "normal"` |
| **Toggle zoom** | `=` or `z` | `=` or `z` | `=` or `z` → `ToggleFocusFullscreen; SwitchToMode "normal"` |
| **New tab** | `t` | `t` | `t` → `NewTab; SwitchToMode "normal"` |
| **New window** | `n` | `n` | `n` → `NewPane; SwitchToMode "normal"` (Zellij has no "windows") |
| **Tab navigator** | `T` | `T` | `T` → `GoToTab; SwitchToMode "normal"` |
| **Tab 1-9** | `1`-`9` | `1`-`9` | `1`-`9` → `GoToTab N; SwitchToMode "normal"` |
| **Switch session** | `s` | `s` | `s` → Launch session-manager plugin |
| **Create session** | `S` | `S` | `S` → *(handled by session-manager plugin)* |
| **Rename tab** | `e` | `e` | `e` → `SwitchToMode "renametab"; TabNameInput 0` |
| **Rename session** | `E` | `E` | `E` → *(no native session rename in Zellij)* |
| **Resize mode** | `r` → `hjkl` | `r` → `hjkl` | `r` → `SwitchToMode "resize"` (then hjkl natively) |
| **Move tab** | `m` → `hl` | `m` → `hl` | `m` → `SwitchToMode "move"` (then hjkl natively) |
| **Copy mode** | `c` | `[` | `[` → `SwitchToMode "scroll"` (then `s` for search) |
| **Detach** | N/A | `d` | `d` → `Detach` |
| **Pane nav** | `←↑↓→` | `←↑↓→` | `←↑↓→` → `MoveFocus "dir"; SwitchToMode "normal"` |
| **Prev/Next tab** | `Ctrl+H/L` | `Ctrl+H/L` | `p`/`n` → `GoToPreviousTab`/`GoToNextTab` |
| **Send Ctrl+K** | N/A | `C-k` | `Ctrl k` → `Write 11; SwitchToMode "normal"` |
| **Command palette** | `p` or `Space` | N/A | `p` → Launch plugin-manager |
| **Debug** | `B` | N/A | `a` → Launch about plugin |

### Notes on Zellij Limitations

- **No split-left/split-up:** Zellij's `NewPane` only supports `"right"` and `"down"` directions. Workaround: create pane then use move mode to reposition. Could bind `h` to `NewPane "right"; MovePaneBackwards` but this is fragile.
- **No session rename:** Zellij sessions are named at creation. The session-manager plugin handles switching/creating.
- **Tab vs Window:** Zellij has tabs and panes, no separate window concept like TMUX. Map `n` to NewPane (new pane in current tab).

---

## Part 2: Status Bar (zjstatus)

### Installation

1. Download zjstatus WASM binary (manual method, recommended):
   ```bash
   mkdir -p ~/.config/zellij/plugins
   curl -L -o ~/.config/zellij/plugins/zjstatus.wasm \
     https://github.com/dj95/zjstatus/releases/latest/download/zjstatus.wasm
   ```

2. Create a custom layout file that uses zjstatus instead of `compact-bar`.

### Layout File Structure

Create `~/.config/zellij/layouts/atlas.kdl`:

```kdl
layout {
    default_tab_template {
        // zjstatus at the top (like Wezterm's tab bar position)
        pane size=1 borderless=true {
            plugin location="file:~/.config/zellij/plugins/zjstatus.wasm" {
                // --- FORMAT ---
                format_left   "{mode} #[fg=#89B4FA,bold]{session} #[fg=#6C7086]⋮ {command_cwd}"
                format_center "{tabs}"
                format_right  "{command_git_branch} #[fg=#6C7086]⋮ {datetime}"
                format_space  ""

                hide_frame_for_single_pane "true"

                // --- MODE INDICATOR (matches Wezterm's leader indicator) ---
                mode_normal        "#[bg=#313244,fg=#CDD6F4]  NORMAL "
                mode_tmux          "#[bg=#89DCEB,fg=#1E1E2E,bold]  LEADER "
                mode_locked        "#[bg=#F38BA8,fg=#1E1E2E]  LOCKED "
                mode_pane          "#[bg=#A6E3A1,fg=#1E1E2E]  PANE "
                mode_tab           "#[bg=#F9E2AF,fg=#1E1E2E]  TAB "
                mode_resize        "#[bg=#CBA6F7,fg=#1E1E2E]  RESIZE "
                mode_move          "#[bg=#FAB387,fg=#1E1E2E]  MOVE "
                mode_scroll        "#[bg=#74C7EC,fg=#1E1E2E]  SCROLL "
                mode_search        "#[bg=#74C7EC,fg=#1E1E2E]  SEARCH "
                mode_enter_search  "#[bg=#74C7EC,fg=#1E1E2E]  SEARCH "
                mode_rename_tab    "#[bg=#F9E2AF,fg=#1E1E2E]  RENAME "
                mode_rename_pane   "#[bg=#A6E3A1,fg=#1E1E2E]  RENAME "
                mode_session       "#[bg=#F5C2E7,fg=#1E1E2E]  SESSION "
                mode_default_to_mode "normal"

                // --- SESSION ---
                session "#[fg=#89B4FA,bold] {name} "

                // --- TABS (matching Wezterm pill-tab style) ---
                tab_normal              "#[fg=#6C7086] {index}:{name} "
                tab_active              "#[fg=#CDD6F4,bold,italic] {index}:{name} "
                tab_normal_fullscreen   "#[fg=#6C7086] {index}:{name}□ "
                tab_active_fullscreen   "#[fg=#CDD6F4,bold] {index}:{name}□ "
                tab_normal_sync         "#[fg=#6C7086,italic] {index}:{name} "
                tab_active_sync         "#[fg=#CDD6F4,bold,italic] {index}:{name} "
                tab_separator           " "
                tab_rename              "#[fg=#F9E2AF,italic] {index}:{name} "

                // --- CWD (command widget — shows plugin CWD) ---
                command_cwd_command     "bash -c 'pwd'"
                command_cwd_format      "#[fg=#F5C2E7] {stdout}"
                command_cwd_interval    "2"
                command_cwd_rendermode  "static"

                // --- GIT BRANCH ---
                command_git_branch_command     "bash -c 'git rev-parse --abbrev-ref HEAD 2>/dev/null'"
                command_git_branch_format      "#[fg=#CBA6F7] {stdout}"
                command_git_branch_interval    "5"
                command_git_branch_rendermode  "static"
                command_git_branch_hide_on_empty_stdout "true"

                // --- DATETIME ---
                datetime          "#[fg=#6C7086] {format}"
                datetime_format   "%H:%M"
                datetime_timezone "America/New_York"
            }
        }
        children
    }
}
```

### Update config.kdl

Change the default layout from `compact` to `atlas`:

```kdl
default_layout "atlas"
```

### Color Scheme (Catppuccin Mocha — matching Wezterm)

| Element | Color | Hex | Wezterm Equivalent |
|---------|-------|-----|-------------------|
| Mode: Normal | Surface0 bg, Text fg | `#313244` / `#CDD6F4` | Workspace (red) |
| Mode: Leader/Tmux | Teal bg | `#89DCEB` | Leader active (cyan) |
| Mode: Locked | Red bg | `#F38BA8` | — |
| Session name | Blue | `#89B4FA` | — |
| Active tab | Text | `#CDD6F4` | Active tab (bold) |
| Inactive tab | Overlay0 | `#6C7086` | Inactive tab (dim) |
| CWD icon/text | Pink | `#F5C2E7` | Pink folder icon |
| Git branch | Mauve | `#CBA6F7` | Purple git icon |
| Separator | Overlay0 | `#6C7086` | Purple alt separator |
| DateTime | Overlay0 | `#6C7086` | — |

### Keybinding Hints in Status Bar

zjstatus doesn't have a native keybinding hint widget. Three options:

1. **Mode-based hints:** Encode key hints into the mode format strings:
   ```kdl
   mode_tmux "#[bg=#89DCEB,fg=#1E1E2E,bold]  LDR #[fg=#45475A]t:tab \\:split -:split x:close s:session "
   ```
   This shows available keys when you're in tmux (leader) mode. Wezterm doesn't do this, but it's better for Zellij since modes are explicit.

2. **Compact-bar tooltip:** Zellij's built-in compact-bar has a `tooltip "F1"` option that shows keybinding hints when you press a key. But this can't be combined with zjstatus on the same bar.

3. **Second status line:** Add a second `pane size=1` with `zellij:status-bar` for keybinding hints at the bottom, keeping zjstatus at the top. This costs one extra terminal line.

**Recommendation:** Option 1 (mode-based hints in tmux mode). This is the most useful — hints appear exactly when you need them (after pressing the leader key), and disappear when you don't. Aligns with how Wezterm shows "LDR" during leader-active state.

---

## Part 3: CWD Tracking (Active Pane)

### The Problem

zjstatus's `command_cwd` widget runs `pwd` in the **plugin's** pane, not the active terminal pane. It will always show `~/.config/zellij/` or similar, not where you're actually working.

### Solution: Shell Hook + zjstatus Pipe

zjstatus supports a `pipe` widget that can receive data from external commands via `zellij pipe`.

**Fish shell hook** (add to `~/.config/fish/conf.d/zellij-cwd.fish`):

```fish
function __zellij_cwd --on-variable PWD
    if set -q ZELLIJ
        zellij pipe -n zjstatus "cwd:$PWD"
    end
end
```

**zjstatus config** (replace the command_cwd widget):

```kdl
// Remove command_cwd_* lines and use pipe instead:
format_left   "{mode} #[fg=#89B4FA,bold]{session} #[fg=#6C7086]⋮ #[fg=#F5C2E7] {pipe_cwd}"

// The pipe widget receives data from `zellij pipe -n zjstatus "cwd:VALUE"`
```

**Note:** This requires zjstatus to support the pipe widget for CWD. Check zjstatus docs/wiki for `pipe` widget support. If not available, fall back to the `command` widget approach (which shows static CWD) or use tab renaming:

**Fallback — Tab rename approach:**

```fish
function __zellij_tab_cwd --on-variable PWD
    if set -q ZELLIJ
        command zellij action rename-tab (basename $PWD)
    end
end
```

This renames tabs to the current directory basename, giving CWD visibility through the tab names themselves. Simpler and works without zjstatus pipe support.

---

## Part 4: Implementation Order

1. **Phase 1 — Keymaps** (~10 min)
   - Update `~/.config/zellij/config.kdl` keybinds
   - Rebind `Ctrl+K` → tmux mode
   - Map all Wezterm/TMUX bindings into the `tmux` block
   - Add double-tap `Ctrl+K` passthrough
   - Test with `zellij` restart

2. **Phase 2 — zjstatus** (~15 min)
   - Download zjstatus.wasm to `~/.config/zellij/plugins/`
   - Create `~/.config/zellij/layouts/atlas.kdl`
   - Configure status bar sections (mode, session, tabs, CWD, git, time)
   - Update `config.kdl` to use `default_layout "atlas"`
   - Test with fresh Zellij session

3. **Phase 3 — CWD Tracking** (~10 min)
   - Add fish shell hook for `zellij pipe` or tab rename
   - Verify CWD updates when changing directories
   - Fine-tune refresh interval

4. **Phase 4 — Keybinding Hints** (~5 min)
   - Add mode-specific hint text to tmux mode format
   - Optionally add bottom status bar for persistent hints

---

## File Inventory

| File | Action | Purpose |
|------|--------|---------|
| `~/.config/zellij/config.kdl` | Modify | Keybinds + default_layout |
| `~/.config/zellij/layouts/atlas.kdl` | Create | zjstatus layout |
| `~/.config/zellij/plugins/zjstatus.wasm` | Download | Status bar plugin |
| `~/.config/fish/conf.d/zellij-cwd.fish` | Create | CWD tracking hook |

---

## Open Questions for Ed

1. **Status bar position:** Top (like Wezterm) or bottom (like TMUX)? Plan assumes top.
2. **Second status line for hints:** Worth the extra terminal line, or mode-based hints sufficient?
3. **CWD approach:** Pipe widget (if supported), tab rename, or static command widget?
4. **Stow management:** Should Zellij config be added to atlas-config as a new stow package (e.g., `zellij/`)?
