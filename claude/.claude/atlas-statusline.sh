#!/bin/bash
# ═══════════════════════════════════════════════════════════════════════════════
# Atlas Status Line
# ═══════════════════════════════════════════════════════════════════════════════
#
# Responsive status line with 4 display modes based on terminal width:
#   - nano   (<35 cols): Minimal single-line displays
#   - micro  (35-54):    Compact with key metrics
#   - mini   (55-79):    Balanced information density
#   - normal (80+):      Full display with all info
#
# Output order: Header → Usage+Context
#
# Rate limits and context data come from Claude Code's native stdin JSON.
# Extra usage (Max plan overage) fetched from API with 5-minute cache.
#
# Context percentage scales to compaction threshold if configured in settings.json.
# When contextDisplay.compactionThreshold is set (e.g., 83), the bar shows 83% as 100%.
# Set threshold to 100 or remove the setting to show raw 0-100% from Claude Code.
# ═══════════════════════════════════════════════════════════════════════════════

set -o pipefail

# ─────────────────────────────────────────────────────────────────────────────
# CONFIGURATION
# ─────────────────────────────────────────────────────────────────────────────

ATLAS_DIR="${ATLAS_DIR:-$HOME/.claude}"
SETTINGS_FILE="$ATLAS_DIR/settings.json"
MODEL_CACHE="$ATLAS_DIR/MEMORY/STATE/model-cache.txt"
EXTRA_USAGE_CACHE="$ATLAS_DIR/MEMORY/STATE/extra-usage-cache.json"

# Extra usage cache TTL — only for Max plan overage (API-only field)
EXTRA_USAGE_CACHE_TTL=300

# ─────────────────────────────────────────────────────────────────────────────
# PARSE INPUT (single jq call extracts everything from stdin)
# ─────────────────────────────────────────────────────────────────────────────

# --print-theme resolves the active theme and exits (used by atlas-statusline.test.sh).
# Skip the blocking stdin read in that mode; theme detection needs no stdin.
if [ "${1:-}" = "--print-theme" ]; then
  input="{}"
else
  input=$(cat)
fi

# Get PAI version from settings
ATLAS_VERSION=$(jq -r '.pai.version // "—"' "$SETTINGS_FILE" 2>/dev/null)
ATLAS_VERSION="${ATLAS_VERSION:-—}"

# Extract all data from JSON in single jq call
eval "$(echo "$input" | jq -r '
  "current_dir=" + (.workspace.current_dir // .cwd // "." | @sh) + "\n" +
  "session_id=" + (.session_id // "" | @sh) + "\n" +
  "model_id=" + (.model.id // "" | @sh) + "\n" +
  "model_name=" + (.model.display_name // "unknown" | @sh) + "\n" +
  "cc_version_json=" + (.version // "" | @sh) + "\n" +
  "context_max=" + (.context_window.context_window_size // 200000 | tostring) + "\n" +
  "context_pct=" + (.context_window.used_percentage // 0 | tostring) + "\n" +
  "cache_read=" + (.context_window.current_usage.cache_read_input_tokens // 0 | tostring) + "\n" +
  "cache_create=" + (.context_window.current_usage.cache_creation_input_tokens // 0 | tostring) + "\n" +
  "exceeds_200k=" + (.exceeds_200k_tokens // false | tostring) + "\n" +
  "rl_5h_pct=" + (.rate_limits.five_hour.used_percentage // -1 | tostring) + "\n" +
  "rl_5h_resets=" + (.rate_limits.five_hour.resets_at // 0 | tostring) + "\n" +
  "rl_7d_pct=" + (.rate_limits.seven_day.used_percentage // -1 | tostring) + "\n" +
  "rl_7d_resets=" + (.rate_limits.seven_day.resets_at // 0 | tostring) + "\n" +
  "worktree_name=" + (.worktree.name // "" | @sh) + "\n" +
  "agent_name=" + (.agent.name // "" | @sh) + "\n" +
  "output_style=" + (.output_style.name // "" | @sh)
' 2>/dev/null)"

# Ensure defaults for critical numeric values
context_pct=${context_pct:-0}
context_max=${context_max:-200000}
cache_read=${cache_read:-0}
cache_create=${cache_create:-0}
rl_5h_pct=${rl_5h_pct:--1}
rl_7d_pct=${rl_7d_pct:--1}

# Cache model name for other tools
mkdir -p "$(dirname "$MODEL_CACHE")" 2>/dev/null
echo "$model_name" >"$MODEL_CACHE" 2>/dev/null

# Parse model version from ID (e.g. "claude-sonnet-4-6" → "4.6")
model_version=$(echo "${model_id:-}" | sed -E 's/claude-[a-z]+-([0-9]+)-([0-9]+).*/\1.\2/' 2>/dev/null)
[ -z "$model_version" ] || [ "$model_version" = "${model_id:-}" ] && model_version=""
# Don't show version if display_name already contains it (e.g. "Sonnet 4.6")
[ -n "$model_version" ] && echo "$model_name" | grep -qF "$model_version" 2>/dev/null && model_version=""
# Split model display: name prefix in PAI_A, version suffix in SLATE_300 white
model_name_short="$model_name"
model_ver_str="${model_version:-}"
if [ -z "$model_ver_str" ]; then
  extracted_ver=$(echo "$model_name" | grep -oE '[0-9]+\.[0-9]+' | head -1)
  if [ -n "$extracted_ver" ]; then
    model_name_short=$(echo "$model_name" | sed "s/ *${extracted_ver}//")
    model_ver_str="$extracted_ver"
  fi
fi
# Compact context window suffix: "(1M context)" → "-1M"
model_name_short=$(echo "$model_name_short" | sed -E 's/ *\(([0-9]+[KMG]) context\)/-\1/')

cc_version="${cc_version_json:-}"

# Get session label — authoritative source: Claude Code's sessions-index.json customTitle
SESSION_LABEL=""
SESSION_NAMES_FILE="$ATLAS_DIR/MEMORY/STATE/session-names.json"
SESSION_CACHE="$ATLAS_DIR/MEMORY/STATE/session-name-cache.sh"
if [ -n "$session_id" ]; then
  project_slug=$(echo "$current_dir" | tr '/.' '-')
  SESSIONS_INDEX="$ATLAS_DIR/projects/${project_slug}/sessions-index.json"

  if [ -f "$SESSION_CACHE" ]; then
    source "$SESSION_CACHE" 2>/dev/null
    if [ "${cached_session_id:-}" = "$session_id" ] && [ -n "${cached_session_label:-}" ]; then
      cache_mtime=$(stat -f %m "$SESSION_CACHE" 2>/dev/null || echo 0)
      idx_mtime=$(stat -f %m "$SESSIONS_INDEX" 2>/dev/null || echo 0)
      names_mtime=$(stat -f %m "$SESSION_NAMES_FILE" 2>/dev/null || echo 0)
      max_source_mtime=$idx_mtime
      [ "$names_mtime" -gt "$max_source_mtime" ] && max_source_mtime=$names_mtime
      [ "$cache_mtime" -ge "$max_source_mtime" ] && SESSION_LABEL="${cached_session_label}"
    fi
  fi

  if [ -z "$SESSION_LABEL" ] && [ -f "$SESSIONS_INDEX" ]; then
    custom_title_line=$(grep -A10 "\"sessionId\": \"$session_id\"" "$SESSIONS_INDEX" 2>/dev/null | grep '"customTitle"' | head -1)
    if [ -n "$custom_title_line" ]; then
      SESSION_LABEL=$(echo "$custom_title_line" | sed 's/.*"customTitle": "//; s/".*//')
    fi
  fi

  if [ -z "$SESSION_LABEL" ] && [ -f "$SESSION_NAMES_FILE" ]; then
    SESSION_LABEL=$(jq -r --arg sid "$session_id" '.[$sid] // empty' "$SESSION_NAMES_FILE" 2>/dev/null)
  fi

  if [ -n "$SESSION_LABEL" ]; then
    mkdir -p "$(dirname "$SESSION_CACHE")" 2>/dev/null
    printf "cached_session_id='%s'\ncached_session_label='%s'\n" "$session_id" "$SESSION_LABEL" >"$SESSION_CACHE"
  fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# RATE LIMITS (native from Claude Code JSON, no API call needed)
# ─────────────────────────────────────────────────────────────────────────────

# Convert native used_percentage (0-100 float) to integer
usage_5h_int=${rl_5h_pct%%.*}
usage_7d_int=${rl_7d_pct%%.*}
# -1 means field was absent (not a Claude.ai subscriber or pre-first-response)
[ "$usage_5h_int" -lt 0 ] 2>/dev/null && usage_5h_int=0
[ "$usage_7d_int" -lt 0 ] 2>/dev/null && usage_7d_int=0
has_rate_limits=false
[ "$rl_5h_pct" != "-1" ] && has_rate_limits=true

# Convert Unix epoch resets_at to local clock time (macOS date -r)
reset_5h_time=""
reset_7d_time=""
if [ "$has_rate_limits" = "true" ]; then
  [ "${rl_5h_resets:-0}" -gt 0 ] 2>/dev/null && reset_5h_time=$(date -r "$rl_5h_resets" "+%H:%M" 2>/dev/null)
  [ "${rl_7d_resets:-0}" -gt 0 ] 2>/dev/null && reset_7d_time=$(date -r "$rl_7d_resets" "+%a %H:%M" 2>/dev/null)
fi

# ─────────────────────────────────────────────────────────────────────────────
# EXTRA USAGE (API-only — Max plan overage, fetched with long cache)
# ─────────────────────────────────────────────────────────────────────────────

extra_display=""
if [ -f "$EXTRA_USAGE_CACHE" ]; then
  cache_age=$(($(date +%s) - $(stat -f %m "$EXTRA_USAGE_CACHE" 2>/dev/null || echo 0)))
  if [ "$cache_age" -gt "$EXTRA_USAGE_CACHE_TTL" ]; then
    # Refresh in background — non-blocking
    {
      keychain_data=$(security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null)
      token=$(echo "$keychain_data" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('claudeAiOauth',{}).get('accessToken',''))" 2>/dev/null)
      if [ -n "$token" ]; then
        usage_json=$(curl -s --max-time 3 \
          -H "Authorization: Bearer $token" \
          -H "Content-Type: application/json" \
          -H "anthropic-beta: oauth-2025-04-20" \
          "https://api.anthropic.com/api/oauth/usage" 2>/dev/null)
        if [ -n "$usage_json" ] && echo "$usage_json" | jq -e '.extra_usage' >/dev/null 2>&1; then
          echo "$usage_json" | jq '.extra_usage' >"$EXTRA_USAGE_CACHE" 2>/dev/null
        fi
      fi
    } &
  fi

  # Read cached extra usage
  eval "$(jq -r '
    "extra_enabled=" + (.is_enabled // false | tostring) + "\n" +
    "extra_limit=" + (.monthly_limit // 0 | tostring) + "\n" +
    "extra_used=" + (.used_credits // 0 | tostring)
  ' "$EXTRA_USAGE_CACHE" 2>/dev/null)"

  if [ "${extra_enabled:-false}" = "true" ]; then
    extra_limit_dollars=$((${extra_limit:-0} / 100))
    extra_used_dollars=$((${extra_used%%.*} / 100))
    if [ "${extra_limit_dollars:-0}" -ge 1000 ]; then
      extra_limit_fmt="\$$((extra_limit_dollars / 1000))K"
    else
      extra_limit_fmt="\$${extra_limit_dollars:-0}"
    fi
    extra_display="\$${extra_used_dollars:-0}/${extra_limit_fmt}"
  fi
fi

# Calculate cache hit rate
cache_total=$((cache_read + cache_create))
if [ "$cache_total" -gt 0 ]; then
  cache_hit_rate=$((cache_read * 100 / cache_total))
else
  cache_hit_rate=0
fi

# ─────────────────────────────────────────────────────────────────────────────
# TERMINAL WIDTH DETECTION
# ─────────────────────────────────────────────────────────────────────────────

detect_terminal_width() {
  local width=""

  # Tier 1: Direct TTY query (works in most terminals)
  width=$(stty size </dev/tty 2>/dev/null | awk '{print $2}')

  # Tier 2: tput fallback
  [ -z "$width" ] || [ "$width" = "0" ] && width=$(tput cols 2>/dev/null)

  # Tier 3: Environment variable
  [ -z "$width" ] || [ "$width" = "0" ] && width=${COLUMNS:-80}

  echo "$width"
}

term_width=$(detect_terminal_width)

if [ "$term_width" -lt 35 ]; then
  MODE="nano"
elif [ "$term_width" -lt 55 ]; then
  MODE="micro"
elif [ "$term_width" -lt 80 ]; then
  MODE="mini"
else
  MODE="normal"
fi

# ─────────────────────────────────────────────────────────────────────────────
# COLOR PALETTE
# ─────────────────────────────────────────────────────────────────────────────
# Multi-theme: Eldritch (dark), Rosé Pine Dawn (light),
# Catppuccin Latte (light) and Catppuccin Mocha (dark — covers frappe/macchiato/mocha).
#
# Resolution priority:
#   1. settings.json override → "statusline": { "theme": "auto"|"light"|"dark" }
#   2. dotfiles theme-switcher state file (~/.config/theme-switcher/current)
#   3. $COLORFGBG
#   4. macOS system appearance
#   5. "dark" default
#

RESET='\033[0m'

# Nerd font icons (theme-independent)
NF_CACHE=$(printf '\xef\x92\x9b')     # nf-oct-cache           U+F49B
NF_GAUGE=$(printf '\xf3\xb0\x8a\x9a') # nf-md-gauge            U+F029A
NF_BRANCH=$(printf '\xef\x90\x98')    # nf-oct-git_branch      U+F418

# Resolve active theme: settings override > theme-switcher state > COLORFGBG > macOS > "dark"
detect_appearance() {
  local override
  override=$(jq -r '.statusline.theme // "auto"' "$SETTINGS_FILE" 2>/dev/null)
  case "$override" in
  light | dark)
    echo "$override"
    return
    ;;
  esac

  # Dotfiles theme-switcher state file — single source of truth when present
  local theme_state="$HOME/.config/theme-switcher/current"
  if [ -f "$theme_state" ]; then
    case "$(cat "$theme_state" 2>/dev/null)" in
    rose-pine-dawn)
      echo "light"
      return
      ;;
    eldritch | tokyonight | rose-pine | rose-pine-moon)
      echo "dark"
      return
      ;;
    catppuccin-latte)
      echo "catppuccin-light"
      return
      ;;
    catppuccin-frappe | catppuccin-macchiato | catppuccin-mocha)
      echo "catppuccin-dark"
      return
      ;;
    esac
  fi

  # COLORFGBG format is "fg;bg" with 0-15 palette indices; bg < 8 = dark, >= 8 = light
  if [ -n "${COLORFGBG:-}" ]; then
    local bg="${COLORFGBG##*;}"
    case "$bg" in
    0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8)
      echo "dark"
      return
      ;;
    9 | 10 | 11 | 12 | 13 | 14 | 15 | default)
      echo "light"
      return
      ;;
    esac
  fi

  # macOS: `defaults` prints "Dark" in dark mode, exits non-zero in light mode
  if defaults read -g AppleInterfaceStyle 2>/dev/null | grep -qi dark; then
    echo "dark"
  elif [ "$(uname)" = "Darwin" ]; then
    echo "light"
  else
    echo "dark"
  fi
}

STATUSLINE_THEME=$(detect_appearance)

# --print-theme: emit the resolved theme name and exit before rendering (unit-test hook).
if [ "${1:-}" = "--print-theme" ]; then
  printf '%s\n' "$STATUSLINE_THEME"
  exit 0
fi

if [ "$STATUSLINE_THEME" = "light" ]; then
  # ── Rosé Pine Dawn ──────────────────────────────────────────────────────
  # base:#faf4ed  text:#575279  subtle:#797593  muted:#9893a5
  # love:#b4637a  gold:#ea9d34  rose:#d7827e  pine:#286983  foam:#56949f  iris:#907aa9

  SLATE_300='\033[38;2;87;82;121m'   # text (primary fg)
  SLATE_500='\033[38;2;121;117;147m' # subtle (muted labels)
  SLATE_600='\033[38;2;223;218;217m' # highlight med (separators)

  EMERALD='\033[38;2;40;105;131m' # pine (positive/success)
  ROSE='\033[38;2;180;99;122m'    # love (error/negative)

  GREET_PRIMARY='\033[38;2;144;122;169m' # iris (header/brand)
  PAI_SESSION='\033[38;2;86;148;159m'    # foam (session name)

  CACHE_ICON='\033[38;2;144;122;169m'       # iris
  CC_ICON='\033[38;2;234;157;52m'           # gold (✱ CC icon)
  DIR_DISPLAY='\033[38;2;86;148;159m'       # foam (PWD)
  GIT_BRANCH_COLOR='\033[38;2;215;130;126m' # rose (git branch)
  WORKTREE_COLOR='\033[38;2;86;148;159m'    # foam (worktree)
  WARN_COLOR='\033[38;2;180;99;122m'        # love (warnings)

  CTX_BUCKET_EMPTY='\033[38;2;223;218;217m' # highlight med (empty bucket)

  USAGE_PRIMARY='\033[38;2;234;157;52m' # gold (usage icon)
  USAGE_RESET='\033[38;2;121;117;147m'  # subtle (reset time)
  USAGE_EXTRA='\033[38;2;215;130;126m'  # rose (overage)

  PAI_A='\033[38;2;144;122;169m' # iris (branding)

  # Gradient stops for get_bucket_color (pine → gold → rose → love)
  GRAD_LOW_R=40
  GRAD_LOW_G=105
  GRAD_LOW_B=131
  GRAD_MID_R=234
  GRAD_MID_G=157
  GRAD_MID_B=52
  GRAD_WRN_R=215
  GRAD_WRN_G=130
  GRAD_WRN_B=126
  GRAD_HI_R=180
  GRAD_HI_G=99
  GRAD_HI_B=122

  # Thresholded accents for get_cache_color (high→low)
  CACHE_TIER_HI="$EMERALD"               # pine
  CACHE_TIER_MH='\033[38;2;86;148;159m'  # foam
  CACHE_TIER_MD='\033[38;2;234;157;52m'  # gold
  CACHE_TIER_ML='\033[38;2;215;130;126m' # rose
  CACHE_TIER_LO="$ROSE"                  # love

  # Thresholded accents for get_usage_color (low→high severity)
  USAGE_TIER_OK="$EMERALD"                 # pine
  USAGE_TIER_MD='\033[38;2;234;157;52m'    # gold
  USAGE_TIER_WARN='\033[38;2;215;130;126m' # rose
  USAGE_TIER_CRIT="$ROSE"                  # love

elif [ "$STATUSLINE_THEME" = "catppuccin-light" ]; then
  # ── Catppuccin Latte (light) ──────────────────────────────────────────────
  # text:#4c4f69  subtext0:#6c6f85  surface1:#bcc0cc
  # green:#40a02b  red:#d20f39  mauve:#8839ef  sky:#04a5e5  peach:#fe640b  pink:#ea76cb  yellow:#df8e1d

  SLATE_300='\033[1;38;2;76;79;105m'   # text (primary fg)
  SLATE_500='\033[1;38;2;108;111;133m' # subtext0 (muted labels)
  SLATE_600='\033[1;38;2;188;192;204m' # surface1 (separators)

  EMERALD='\033[1;38;2;64;160;43m' # green (positive/success)
  ROSE='\033[1;38;2;210;15;57m'    # red (error/negative)

  GREET_PRIMARY='\033[1;38;2;136;57;239m' # mauve (header/brand)
  PAI_SESSION='\033[1;38;2;4;165;229m'    # sky (session name)

  CACHE_ICON='\033[1;38;2;136;57;239m'        # mauve
  CC_ICON='\033[1;38;2;254;100;11m'           # peach (✱ CC icon)
  DIR_DISPLAY='\033[1;38;2;4;165;229m'        # sky (PWD)
  GIT_BRANCH_COLOR='\033[1;38;2;234;118;203m' # pink (git branch)
  WORKTREE_COLOR='\033[1;38;2;4;165;229m'     # sky (worktree)
  WARN_COLOR='\033[1;38;2;210;15;57m'         # red (warnings)

  CTX_BUCKET_EMPTY='\033[1;38;2;188;192;204m' # surface1 (empty bucket)

  USAGE_PRIMARY='\033[1;38;2;254;100;11m' # peach (usage icon)
  USAGE_RESET='\033[1;38;2;140;143;161m'  # overlay1 (reset time)
  USAGE_EXTRA='\033[1;38;2;234;118;203m'  # pink (overage)

  PAI_A='\033[1;38;2;136;57;239m' # mauve (branding)

  # Gradient stops for get_bucket_color (green → yellow → peach → red)
  GRAD_LOW_R=64
  GRAD_LOW_G=160
  GRAD_LOW_B=43
  GRAD_MID_R=223
  GRAD_MID_G=142
  GRAD_MID_B=29
  GRAD_WRN_R=254
  GRAD_WRN_G=100
  GRAD_WRN_B=11
  GRAD_HI_R=210
  GRAD_HI_G=15
  GRAD_HI_B=57

  CACHE_TIER_HI="$EMERALD"                # green
  CACHE_TIER_MH='\033[1;38;2;4;165;229m'  # sky
  CACHE_TIER_MD='\033[1;38;2;223;142;29m' # yellow
  CACHE_TIER_ML='\033[1;38;2;254;100;11m' # peach
  CACHE_TIER_LO="$ROSE"                   # red

  USAGE_TIER_OK="$EMERALD"
  USAGE_TIER_MD='\033[1;38;2;223;142;29m'   # yellow
  USAGE_TIER_WARN='\033[1;38;2;254;100;11m' # peach
  USAGE_TIER_CRIT="$ROSE"

elif [ "$STATUSLINE_THEME" = "catppuccin-dark" ]; then
  # ── Catppuccin Mocha (dark — frappe/macchiato/mocha) ──────────────────────────────────────────────
  # text:#cdd6f4  subtext0:#a6adc8  surface1:#45475a
  # green:#a6e3a1  red:#f38ba8  mauve:#cba6f7  sky:#89dceb  peach:#fab387  pink:#f5c2e7  yellow:#f9e2af

  SLATE_300='\033[38;2;205;214;244m' # text (primary fg)
  SLATE_500='\033[38;2;166;173;200m' # subtext0 (muted labels)
  SLATE_600='\033[38;2;69;71;90m'    # surface1 (separators)

  EMERALD='\033[38;2;166;227;161m' # green (positive/success)
  ROSE='\033[38;2;243;139;168m'    # red (error/negative)

  GREET_PRIMARY='\033[38;2;203;166;247m' # mauve (header/brand)
  PAI_SESSION='\033[38;2;137;220;235m'   # sky (session name)

  CACHE_ICON='\033[38;2;203;166;247m'       # mauve
  CC_ICON='\033[38;2;250;179;135m'          # peach (✱ CC icon)
  DIR_DISPLAY='\033[38;2;137;220;235m'      # sky (PWD)
  GIT_BRANCH_COLOR='\033[38;2;245;194;231m' # pink (git branch)
  WORKTREE_COLOR='\033[38;2;137;220;235m'   # sky (worktree)
  WARN_COLOR='\033[38;2;243;139;168m'       # red (warnings)

  CTX_BUCKET_EMPTY='\033[38;2;69;71;90m' # surface1 (empty bucket)

  USAGE_PRIMARY='\033[38;2;250;179;135m' # peach (usage icon)
  USAGE_RESET='\033[38;2;127;132;156m'   # overlay1 (reset time)
  USAGE_EXTRA='\033[38;2;245;194;231m'   # pink (overage)

  PAI_A='\033[38;2;203;166;247m' # mauve (branding)

  # Gradient stops for get_bucket_color (green → yellow → peach → red)
  GRAD_LOW_R=166
  GRAD_LOW_G=227
  GRAD_LOW_B=161
  GRAD_MID_R=249
  GRAD_MID_G=226
  GRAD_MID_B=175
  GRAD_WRN_R=250
  GRAD_WRN_G=179
  GRAD_WRN_B=135
  GRAD_HI_R=243
  GRAD_HI_G=139
  GRAD_HI_B=168

  CACHE_TIER_HI="$EMERALD"               # green
  CACHE_TIER_MH='\033[38;2;137;220;235m' # sky
  CACHE_TIER_MD='\033[38;2;249;226;175m' # yellow
  CACHE_TIER_ML='\033[38;2;250;179;135m' # peach
  CACHE_TIER_LO="$ROSE"                  # red

  USAGE_TIER_OK="$EMERALD"
  USAGE_TIER_MD='\033[38;2;249;226;175m'   # yellow
  USAGE_TIER_WARN='\033[38;2;250;179;135m' # peach
  USAGE_TIER_CRIT="$ROSE"

else
  # ── Eldritch (dark, default) ────────────────────────────────────────────
  # bg:#212337  line:#323449  fg:#ebfafa
  # cyan:#04d1f9  green:#37f499  purple:#a48cf2  muted-purple:#7081d0
  # orange:#f7c67f  pink:#f265b5  red:#f16c75  yellow:#f1fc79

  SLATE_300='\033[38;2;235;250;250m' # Lighthouse White (fg)
  SLATE_500='\033[38;2;112;129;208m' # The Old One Purple (muted)
  SLATE_600='\033[38;2;50;52;73m'    # Shallow Depths Grey (separators)

  EMERALD='\033[38;2;55;244;153m' # Great Old One Green (positive/success)
  ROSE='\033[38;2;241;108;117m'   # R'lyeh Red (error/negative)

  GREET_PRIMARY='\033[38;2;164;140;242m' # Lovecraft Purple
  PAI_SESSION='\033[38;2;4;209;249m'     # Watery Tomb Cyan for session name

  CACHE_ICON='\033[38;2;164;140;242m'       # Lovecraft Purple for cache icon
  CC_ICON='\033[38;2;232;97;32m'            # Anthropic Orange for ✱ CC icon
  DIR_DISPLAY='\033[38;2;4;209;249m'        # Watery Tomb Cyan for PWD
  GIT_BRANCH_COLOR='\033[38;2;242;101;181m' # Pustule Pink for git branch
  WORKTREE_COLOR='\033[38;2;4;209;249m'     # Watery Tomb Cyan for worktree
  WARN_COLOR='\033[38;2;241;108;117m'       # R'lyeh Red for warnings

  CTX_BUCKET_EMPTY='\033[38;2;50;52;73m' # Shallow Depths Grey (empty bucket)

  USAGE_PRIMARY='\033[38;2;247;198;127m' # Dreaming Orange icon
  USAGE_RESET='\033[38;2;112;129;208m'   # The Old One Purple for reset time
  USAGE_EXTRA='\033[38;2;242;101;181m'   # Pustule Pink for overage

  PAI_A='\033[38;2;164;140;242m' # Lovecraft Purple

  # Gradient stops for get_bucket_color (green → yellow → orange → red)
  GRAD_LOW_R=55
  GRAD_LOW_G=244
  GRAD_LOW_B=153
  GRAD_MID_R=241
  GRAD_MID_G=252
  GRAD_MID_B=121
  GRAD_WRN_R=247
  GRAD_WRN_G=198
  GRAD_WRN_B=127
  GRAD_HI_R=241
  GRAD_HI_G=108
  GRAD_HI_B=117

  CACHE_TIER_HI="$EMERALD"
  CACHE_TIER_MH='\033[38;2;4;209;249m'   # Watery Tomb Blue (cyan)
  CACHE_TIER_MD='\033[38;2;241;252;121m' # Gold of Yuggoth (yellow)
  CACHE_TIER_ML='\033[38;2;247;198;127m' # Dreaming Orange
  CACHE_TIER_LO="$ROSE"

  USAGE_TIER_OK="$EMERALD"
  USAGE_TIER_MD='\033[38;2;251;191;36m'   # amber
  USAGE_TIER_WARN='\033[38;2;251;146;60m' # orange
  USAGE_TIER_CRIT="$ROSE"
fi

# ─────────────────────────────────────────────────────────────────────────────
# HELPER FUNCTIONS
# ─────────────────────────────────────────────────────────────────────────────

# Get gradient color for context bar bucket.
# Interpolates across 4 theme-defined stops: low → mid → warn → hi.
# Eldritch: green → yellow → orange → red.
# Rosé Pine Dawn: pine → gold → rose → love.
get_bucket_color() {
  local pos=$1 max=$2
  local pct=$((pos * 100 / max))
  local r g b

  if [ "$pct" -le 33 ]; then
    r=$((GRAD_LOW_R + (GRAD_MID_R - GRAD_LOW_R) * pct / 33))
    g=$((GRAD_LOW_G + (GRAD_MID_G - GRAD_LOW_G) * pct / 33))
    b=$((GRAD_LOW_B + (GRAD_MID_B - GRAD_LOW_B) * pct / 33))
  elif [ "$pct" -le 66 ]; then
    local t=$((pct - 33))
    r=$((GRAD_MID_R + (GRAD_WRN_R - GRAD_MID_R) * t / 33))
    g=$((GRAD_MID_G + (GRAD_WRN_G - GRAD_MID_G) * t / 33))
    b=$((GRAD_MID_B + (GRAD_WRN_B - GRAD_MID_B) * t / 33))
  else
    local t=$((pct - 66))
    r=$((GRAD_WRN_R + (GRAD_HI_R - GRAD_WRN_R) * t / 34))
    g=$((GRAD_WRN_G + (GRAD_HI_G - GRAD_WRN_G) * t / 34))
    b=$((GRAD_WRN_B + (GRAD_HI_B - GRAD_WRN_B) * t / 34))
  fi
  printf '\033[38;2;%d;%d;%dm' "$r" "$g" "$b"
}

# Get gradient color for cache hit rate (high quality → low quality)
get_cache_color() {
  local pct_int="$1"
  [ -z "$pct_int" ] && pct_int=0
  if [ "$pct_int" -ge 80 ]; then
    echo "$CACHE_TIER_HI"
  elif [ "$pct_int" -ge 60 ]; then
    echo "$CACHE_TIER_MH"
  elif [ "$pct_int" -ge 40 ]; then
    echo "$CACHE_TIER_MD"
  elif [ "$pct_int" -ge 20 ]; then
    echo "$CACHE_TIER_ML"
  else
    echo "$CACHE_TIER_LO"
  fi
}

# Get color for usage percentage (ok → critical)
get_usage_color() {
  local pct_int=${1%%.*}
  [ -z "$pct_int" ] && pct_int=0
  if [ "$pct_int" -ge 80 ]; then
    echo "$USAGE_TIER_CRIT"
  elif [ "$pct_int" -ge 60 ]; then
    echo "$USAGE_TIER_WARN"
  elif [ "$pct_int" -ge 40 ]; then
    echo "$USAGE_TIER_MD"
  else
    echo "$USAGE_TIER_OK"
  fi
}

# Render compact context bar (nano/micro modes)
render_context_bar() {
  local width=$1 pct=$2
  local output=""
  local filled=$((pct * width / 100))
  [ "$filled" -lt 0 ] && filled=0

  for i in $(seq 1 $width 2>/dev/null); do
    if [ "$i" -le "$filled" ]; then
      local color=$(get_bucket_color $i $width)
      output="${output}${color}⛁${RESET}"
    else
      output="${output}${CTX_BUCKET_EMPTY}⛁${RESET}"
    fi
  done
  echo "$output"
}

# Render the shared tail of LINE 2: context bar + percentage, then cache hit rate.
# Identical across every width mode — only the bar width differs.
# Reads display_pct, ctx_warn, pct_color, cache_total, cache_hit_rate from scope.
render_ctx_cache_tail() {
  local bar_width=$1 ctx_bar cache_num_color
  ctx_bar=$(render_context_bar "$bar_width" "$display_pct")
  printf " ${SLATE_600}·${RESET} ${ctx_warn}${ctx_bar} ${pct_color}${display_pct}%%${RESET}"
  if [ "$cache_total" -gt 0 ]; then
    cache_num_color=$(get_cache_color "$cache_hit_rate")
    printf " ${SLATE_600}·${RESET} ${CACHE_ICON}${NF_CACHE} ${cache_num_color}${cache_hit_rate}%%${RESET}"
  fi
  printf "\n"
}

# ═══════════════════════════════════════════════════════════════════════════════
# LINE 1: ATLAS HEADER
# ═══════════════════════════════════════════════════════════════════════════════

session_display=""
[ -n "$SESSION_LABEL" ] && session_display=$(echo "$SESSION_LABEL" | tr '[:lower:]' '[:upper:]')

# Display just the directory basename (e.g., /atlas-config)
# Tail-truncate to 20 chars with ellipsis if basename is long.
dir_display="/${current_dir##*/}"
if [ ${#dir_display} -gt 20 ]; then
  dir_display="${dir_display:0:19}…"
fi

# Get current git branch (runs each refresh — always up to date)
# Middle-truncate to 24 chars (head + … + tail) to preserve date suffixes
# on worktree branches like `worktree-pai-upgrade-2026-04-17`.
git_branch=$(git -C "$current_dir" rev-parse --abbrev-ref HEAD 2>/dev/null)
if [ -n "$git_branch" ] && [ ${#git_branch} -gt 24 ]; then
  git_branch="${git_branch:0:11}…${git_branch: -12}"
fi
git_display=""
git_display_plain=""
if [ -n "$git_branch" ]; then
  git_display=" ${SLATE_600}·${RESET} ${GIT_BRANCH_COLOR}${NF_BRANCH} ${git_branch}${RESET}"
  git_display_plain=" · ${NF_BRANCH} ${git_branch}"
fi

# Build version suffix: model + CC version
ver_suffix=""
ver_suffix_plain=""
ver_suffix="${PAI_A}${model_name_short}${RESET}"
ver_suffix_plain="${model_name_short}"
[ -n "$model_ver_str" ] && {
  ver_suffix="${ver_suffix} ${SLATE_300}${model_ver_str}${RESET}"
  ver_suffix_plain="${ver_suffix_plain} ${model_ver_str}"
}
[ -n "$cc_version" ] && {
  ver_suffix="${ver_suffix} ${SLATE_600}·${RESET} ${CC_ICON}✱ ${SLATE_300}${cc_version}${RESET}"
  ver_suffix_plain="${ver_suffix_plain} · ${cc_version}"
}

# Build context tags (worktree, agent, output style) for header
# Skip worktree tag when it duplicates the cwd basename (common case — the worktree
# folder IS the cwd, so showing both is redundant).
ctx_tags=""
ctx_tags_plain=""
cwd_basename="${current_dir##*/}"
[ -n "${worktree_name:-}" ] && [ "${worktree_name}" != "${cwd_basename}" ] && {
  ctx_tags="${ctx_tags} ${SLATE_600}·${RESET} ${WORKTREE_COLOR}⎇ ${worktree_name}${RESET}"
  ctx_tags_plain="${ctx_tags_plain} · ⎇ ${worktree_name}"
}
[ -n "${agent_name:-}" ] && {
  ctx_tags="${ctx_tags} ${SLATE_600}·${RESET} ${PAI_SESSION}⚙ ${agent_name}${RESET}"
  ctx_tags_plain="${ctx_tags_plain} · ⚙ ${agent_name}"
}
[ -n "${output_style:-}" ] && [ "${output_style}" != "default" ] && {
  ctx_tags="${ctx_tags} ${SLATE_600}·${RESET} ${SLATE_500}◉ ${output_style}${RESET}"
  ctx_tags_plain="${ctx_tags_plain} · ◉ ${output_style}"
}

case "$MODE" in
nano)
  printf "${GREET_PRIMARY}▲ Atlas ${SLATE_300}v${ATLAS_VERSION}${RESET} ${SLATE_600}·${RESET} ${ver_suffix}\n"
  ;;
micro)
  # Identity on line 1, location (pwd + branch) on line 2 — avoids wrap at narrow widths.
  printf "${GREET_PRIMARY}▲ Atlas ${SLATE_300}v${ATLAS_VERSION}${RESET} ${SLATE_600}·${RESET} ${ver_suffix}\n"
  printf "${DIR_DISPLAY}${dir_display}${RESET}${git_display}\n"
  ;;
mini)
  # Identity on line 1, location + context tags on line 2.
  printf "${GREET_PRIMARY}▲ Atlas ${SLATE_300}v${ATLAS_VERSION}${RESET} ${SLATE_600}·${RESET} ${ver_suffix}\n"
  printf "${DIR_DISPLAY}${dir_display}${RESET}${git_display}${ctx_tags}\n"
  ;;
*)
  # Session title intentionally omitted — Claude Code v2.1.114+ renders it natively
  # as a chip above the prompt input, so showing it here would duplicate.
  local_left="▲ Atlas v${ATLAS_VERSION} · ${ver_suffix_plain} · ${dir_display}${git_display_plain}${ctx_tags_plain} "
  local_left_len=${#local_left}
  local_fill=$((72 - local_left_len))
  [ "$local_fill" -lt 2 ] && local_fill=2
  local_dashes=$(printf '%*s' "$local_fill" '' | tr ' ' '─')
  printf "${GREET_PRIMARY}▲ Atlas ${SLATE_300}v${ATLAS_VERSION}${RESET} ${SLATE_600}·${RESET} ${ver_suffix} ${SLATE_600}·${RESET} ${DIR_DISPLAY}${dir_display}${RESET}${git_display}${ctx_tags} ${SLATE_600}${local_dashes}${RESET}\n"
  ;;
esac

# ═══════════════════════════════════════════════════════════════════════════════
# LINE 2: USAGE + CONTEXT + COST
# ═══════════════════════════════════════════════════════════════════════════════

# Context computation (scaled to compaction threshold if configured)
COMPACTION_THRESHOLD=$(jq -r '.contextDisplay.compactionThreshold // 100' "$SETTINGS_FILE" 2>/dev/null)
COMPACTION_THRESHOLD="${COMPACTION_THRESHOLD:-100}"
raw_pct="${context_pct%%.*}"
[ -z "$raw_pct" ] && raw_pct=0
if [ "$COMPACTION_THRESHOLD" -lt 100 ] && [ "$COMPACTION_THRESHOLD" -gt 0 ]; then
  display_pct=$((raw_pct * 100 / COMPACTION_THRESHOLD))
  [ "$display_pct" -gt 100 ] && display_pct=100
else
  display_pct="$raw_pct"
fi
pct_color=$(get_usage_color "$display_pct")

# exceeds_200k warning prefix
ctx_warn=""
[ "${exceeds_200k:-false}" = "true" ] && ctx_warn="${WARN_COLOR}⚠${RESET} "

# Usage colors
usage_5h_color=$(get_usage_color "$usage_5h_int")
usage_7d_color=$(get_usage_color "$usage_7d_int")

# Usage + context bar + cache + cost
case "$MODE" in
nano)
  [ "$has_rate_limits" = "true" ] && printf "${USAGE_PRIMARY}${NF_GAUGE}${RESET} ${usage_5h_color}${usage_5h_int}%%${RESET} ${USAGE_RESET}↻${reset_5h_time}${RESET}"
  render_ctx_cache_tail 6
  ;;
micro)
  [ "$has_rate_limits" = "true" ] && printf "${USAGE_PRIMARY}${NF_GAUGE}${RESET} ${USAGE_RESET}5H:${RESET}${usage_5h_color}${usage_5h_int}%%${RESET} ${USAGE_RESET}↻${reset_5h_time}${RESET}"
  render_ctx_cache_tail 8
  ;;
mini)
  if [ "$has_rate_limits" = "true" ]; then
    printf "${USAGE_PRIMARY}${NF_GAUGE}${RESET} ${USAGE_RESET}5H:${RESET}${usage_5h_color}${usage_5h_int}%%${RESET} ${USAGE_RESET}↻${SLATE_500}${reset_5h_time}${RESET} ${SLATE_600}·${RESET} ${USAGE_RESET}WK:${RESET}${usage_7d_color}${usage_7d_int}%%${RESET} ${USAGE_RESET}↻${SLATE_500}${reset_7d_time}${RESET}"
  fi
  render_ctx_cache_tail 16
  ;;
normal)
  if [ "$has_rate_limits" = "true" ]; then
    printf "${USAGE_PRIMARY}${NF_GAUGE}${RESET} ${USAGE_RESET}5H:${RESET}${usage_5h_color}${usage_5h_int}%%${RESET} ${USAGE_RESET}↻${SLATE_500}${reset_5h_time}${RESET} ${SLATE_600}·${RESET} ${USAGE_RESET}WK:${RESET}${usage_7d_color}${usage_7d_int}%%${RESET} ${USAGE_RESET}↻${SLATE_500}${reset_7d_time}${RESET}"
    [ -n "$extra_display" ] && printf " ${SLATE_600}·${RESET} ${USAGE_EXTRA}${extra_display}${RESET}"
  fi
  render_ctx_cache_tail 24
  ;;
esac
