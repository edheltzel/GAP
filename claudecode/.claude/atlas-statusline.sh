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

PAI_DIR="${PAI_DIR:-$HOME/.claude}"
SETTINGS_FILE="$PAI_DIR/settings.json"
MODEL_CACHE="$PAI_DIR/MEMORY/STATE/model-cache.txt"
EXTRA_USAGE_CACHE="$PAI_DIR/MEMORY/STATE/extra-usage-cache.json"

# Extra usage cache TTL — only for Max plan overage (API-only field)
EXTRA_USAGE_CACHE_TTL=300

# ─────────────────────────────────────────────────────────────────────────────
# PARSE INPUT (single jq call extracts everything from stdin)
# ─────────────────────────────────────────────────────────────────────────────

input=$(cat)

# Get PAI version from settings
PAI_VERSION=$(jq -r '.pai.version // "—"' "$SETTINGS_FILE" 2>/dev/null)
PAI_VERSION="${PAI_VERSION:-—}"

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
  "cost_usd=" + (.cost.total_cost_usd // 0 | tostring) + "\n" +
  "duration_ms=" + (.cost.total_duration_ms // 0 | tostring) + "\n" +
  "lines_added=" + (.cost.total_lines_added // 0 | tostring) + "\n" +
  "lines_removed=" + (.cost.total_lines_removed // 0 | tostring) + "\n" +
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
cost_usd=${cost_usd:-0}
duration_ms=${duration_ms:-0}

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
cc_version="${cc_version_json:-}"

# Get session label — authoritative source: Claude Code's sessions-index.json customTitle
SESSION_LABEL=""
SESSION_NAMES_FILE="$PAI_DIR/MEMORY/STATE/session-names.json"
SESSION_CACHE="$PAI_DIR/MEMORY/STATE/session-name-cache.sh"
if [ -n "$session_id" ]; then
  project_slug=$(echo "$current_dir" | tr '/.' '-')
  SESSIONS_INDEX="$PAI_DIR/projects/${project_slug}/sessions-index.json"

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

# Format cost and duration
cost_fmt=$(printf '$%.2f' "$cost_usd" 2>/dev/null || echo '$0.00')
duration_sec=$((${duration_ms%%.*} / 1000))
duration_min=$((duration_sec / 60))
duration_secs=$((duration_sec % 60))

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
# Eldritch theme — https://github.com/eldritch-theme
# bg:#212337  line:#323449  fg:#ebfafa
# cyan:#04d1f9  green:#37f499  purple:#a48cf2  muted-purple:#7081d0
# orange:#f7c67f  pink:#f265b5  red:#f16c75  yellow:#f1fc79

RESET='\033[0m'

# Structural (chrome, labels, separators)
SLATE_300='\033[38;2;235;250;250m' # Lighthouse White (fg)
SLATE_500='\033[38;2;112;129;208m' # The Old One Purple (muted)
SLATE_600='\033[38;2;50;52;73m'    # Shallow Depths Grey (separators)

# Semantic colors
EMERALD='\033[38;2;55;244;153m' # Great Old One Green (positive/success)
ROSE='\033[38;2;241;108;117m'   # R'lyeh Red (error/negative)

# Header / Branding (Lovecraft Purple theme)
GREET_PRIMARY='\033[38;2;164;140;242m' # Lovecraft Purple
PAI_SESSION='\033[38;2;4;209;249m'     # Watery Tomb Cyan for session name

# Icons and accents
CACHE_ICON='\033[38;2;164;140;242m'     # Lovecraft Purple for cache icon
CC_ICON='\033[38;2;232;97;32m'          # Anthropic Orange for ✱ CC icon
DIR_DISPLAY='\033[38;2;242;101;181m'    # Pustule Pink for PWD
COST_COLOR='\033[38;2;247;198;127m'     # Dreaming Orange for cost
DURATION_COLOR='\033[38;2;112;129;208m' # The Old One Purple for duration
WORKTREE_COLOR='\033[38;2;4;209;249m'   # Watery Tomb Cyan for worktree
WARN_COLOR='\033[38;2;241;108;117m'     # R'lyeh Red for warnings

# Nerd font icons
NF_CACHE=$(printf '\xef\x92\x9b')        # nf-oct-cache           U+F49B
NF_GAUGE=$(printf '\xf3\xb0\x8a\x9a')    # nf-md-gauge            U+F029A

# Context bar
CTX_BUCKET_EMPTY='\033[38;2;50;52;73m' # Shallow Depths Grey (empty bucket)

# Usage (Orange/Yellow theme)
USAGE_PRIMARY='\033[38;2;247;198;127m' # Dreaming Orange icon
USAGE_RESET='\033[38;2;112;129;208m'   # The Old One Purple for reset time
USAGE_EXTRA='\033[38;2;242;101;181m'   # Pustule Pink for overage

# PAI Branding
PAI_A='\033[38;2;164;140;242m' # Lovecraft Purple

# ─────────────────────────────────────────────────────────────────────────────
# HELPER FUNCTIONS
# ─────────────────────────────────────────────────────────────────────────────

# Get gradient color for context bar bucket
# Green(55,244,153) → Yellow(241,252,121) → Orange(247,198,127) → Red(241,108,117)
get_bucket_color() {
  local pos=$1 max=$2
  local pct=$((pos * 100 / max))
  local r g b

  if [ "$pct" -le 33 ]; then
    r=$((55 + (241 - 55) * pct / 33))
    g=$((244 + (252 - 244) * pct / 33))
    b=$((153 + (121 - 153) * pct / 33))
  elif [ "$pct" -le 66 ]; then
    local t=$((pct - 33))
    r=$((241 + (247 - 241) * t / 33))
    g=$((252 + (198 - 252) * t / 33))
    b=$((121 + (127 - 121) * t / 33))
  else
    local t=$((pct - 66))
    r=$((247 + (241 - 247) * t / 34))
    g=$((198 + (108 - 198) * t / 34))
    b=$((127 + (117 - 127) * t / 34))
  fi
  printf '\033[38;2;%d;%d;%dm' "$r" "$g" "$b"
}

# Get gradient color for cache hit rate (green=high → red=low)
get_cache_color() {
  local pct_int="$1"
  [ -z "$pct_int" ] && pct_int=0
  if [ "$pct_int" -ge 80 ]; then
    echo "$EMERALD"
  elif [ "$pct_int" -ge 60 ]; then
    echo '\033[38;2;4;209;249m' # Watery Tomb Blue (cyan)
  elif [ "$pct_int" -ge 40 ]; then
    echo '\033[38;2;241;252;121m' # Gold of Yuggoth (yellow)
  elif [ "$pct_int" -ge 20 ]; then
    echo '\033[38;2;247;198;127m' # Dreaming Orange
  else
    echo "$ROSE"
  fi
}

# Get color for usage percentage (green→yellow→orange→red)
get_usage_color() {
  local pct_int=${1%%.*}
  [ -z "$pct_int" ] && pct_int=0
  if [ "$pct_int" -ge 80 ]; then
    echo "$ROSE"
  elif [ "$pct_int" -ge 60 ]; then
    echo '\033[38;2;251;146;60m'
  elif [ "$pct_int" -ge 40 ]; then
    echo '\033[38;2;251;191;36m'
  else
    echo "$EMERALD"
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

# ═══════════════════════════════════════════════════════════════════════════════
# LINE 1: ATLAS HEADER
# ═══════════════════════════════════════════════════════════════════════════════

session_display=""
[ -n "$SESSION_LABEL" ] && session_display=$(echo "$SESSION_LABEL" | tr '[:lower:]' '[:upper:]')

# Display just the directory basename (e.g., /atlas-config)
dir_display="/${current_dir##*/}"

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
ctx_tags=""
ctx_tags_plain=""
[ -n "${worktree_name:-}" ] && {
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
  printf "${GREET_PRIMARY}▲ Atlas ${SLATE_300}v${PAI_VERSION}${RESET} ${SLATE_600}·${RESET} ${ver_suffix}\n"
  ;;
micro)
  printf "${GREET_PRIMARY}▲ Atlas ${SLATE_300}v${PAI_VERSION}${RESET} ${SLATE_600}·${RESET} ${ver_suffix} ${SLATE_600}·${RESET} ${DIR_DISPLAY}${dir_display}${RESET}\n"
  ;;
*)
  if [ -n "$session_display" ]; then
    local_left="▲ Atlas v${PAI_VERSION} · ${ver_suffix_plain} · ${dir_display} · ${session_display}${ctx_tags_plain} "
    local_left_len=${#local_left}
    local_fill=$((72 - local_left_len))
    [ "$local_fill" -lt 2 ] && local_fill=2
    local_dashes=$(printf '%*s' "$local_fill" '' | tr ' ' '─')
    printf "${GREET_PRIMARY}▲ Atlas ${SLATE_300}v${PAI_VERSION}${RESET} ${SLATE_600}·${RESET} ${ver_suffix} ${SLATE_600}·${RESET} ${DIR_DISPLAY}${dir_display}${RESET} ${SLATE_600}·${RESET} ${PAI_SESSION}${session_display}${RESET}${ctx_tags} ${SLATE_600}${local_dashes}${RESET}\n"
  else
    local_left="▲ Atlas v${PAI_VERSION} · ${ver_suffix_plain} · ${dir_display}${ctx_tags_plain} "
    local_left_len=${#local_left}
    local_fill=$((72 - local_left_len))
    [ "$local_fill" -lt 2 ] && local_fill=2
    local_dashes=$(printf '%*s' "$local_fill" '' | tr ' ' '─')
    printf "${GREET_PRIMARY}▲ Atlas ${SLATE_300}v${PAI_VERSION}${RESET} ${SLATE_600}·${RESET} ${ver_suffix} ${SLATE_600}·${RESET} ${DIR_DISPLAY}${dir_display}${RESET}${ctx_tags} ${SLATE_600}${local_dashes}${RESET}\n"
  fi
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
if [ "$display_pct" -ge 80 ]; then
  pct_color="$ROSE"
elif [ "$display_pct" -ge 60 ]; then
  pct_color='\033[38;2;251;146;60m'
elif [ "$display_pct" -ge 40 ]; then
  pct_color='\033[38;2;251;191;36m'
else
  pct_color="$EMERALD"
fi

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
  ctx_bar=$(render_context_bar 6 $display_pct)
  printf " ${SLATE_600}·${RESET} ${ctx_warn}${ctx_bar} ${pct_color}${display_pct}%%${RESET}"
  if [ "$cache_total" -gt 0 ]; then
    cache_num_color=$(get_cache_color "$cache_hit_rate")
    printf " ${SLATE_600}·${RESET} ${CACHE_ICON}${NF_CACHE} ${cache_num_color}${cache_hit_rate}%%${RESET}"
  fi
  printf "\n"
  ;;
micro)
  [ "$has_rate_limits" = "true" ] && printf "${USAGE_PRIMARY}${NF_GAUGE}${RESET} ${USAGE_RESET}5H:${RESET}${usage_5h_color}${usage_5h_int}%%${RESET} ${USAGE_RESET}↻${reset_5h_time}${RESET}"
  ctx_bar=$(render_context_bar 8 $display_pct)
  printf " ${SLATE_600}·${RESET} ${ctx_warn}${ctx_bar} ${pct_color}${display_pct}%%${RESET}"
  if [ "$cache_total" -gt 0 ]; then
    cache_num_color=$(get_cache_color "$cache_hit_rate")
    printf " ${SLATE_600}·${RESET} ${CACHE_ICON}${NF_CACHE} ${cache_num_color}${cache_hit_rate}%%${RESET}"
  fi
  printf "\n"
  ;;
mini)
  if [ "$has_rate_limits" = "true" ]; then
    printf "${USAGE_PRIMARY}${NF_GAUGE}${RESET} ${USAGE_RESET}5H:${RESET}${usage_5h_color}${usage_5h_int}%%${RESET} ${USAGE_RESET}↻${SLATE_500}${reset_5h_time}${RESET} ${SLATE_600}·${RESET} ${USAGE_RESET}WK:${RESET}${usage_7d_color}${usage_7d_int}%%${RESET} ${USAGE_RESET}↻${SLATE_500}${reset_7d_time}${RESET}"
  fi
  ctx_bar=$(render_context_bar 16 $display_pct)
  printf " ${SLATE_600}·${RESET} ${ctx_warn}${ctx_bar} ${pct_color}${display_pct}%%${RESET}"
  if [ "$cache_total" -gt 0 ]; then
    cache_num_color=$(get_cache_color "$cache_hit_rate")
    printf " ${SLATE_600}·${RESET} ${CACHE_ICON}${NF_CACHE} ${cache_num_color}${cache_hit_rate}%%${RESET}"
  fi
  [ "$duration_sec" -gt 0 ] && printf " ${SLATE_600}·${RESET} ${COST_COLOR}${cost_fmt}${RESET} ${DURATION_COLOR}${duration_min}m${duration_secs}s${RESET}"
  printf "\n"
  ;;
normal)
  if [ "$has_rate_limits" = "true" ]; then
    printf "${USAGE_PRIMARY}${NF_GAUGE}${RESET} ${USAGE_RESET}5H:${RESET}${usage_5h_color}${usage_5h_int}%%${RESET} ${USAGE_RESET}↻${SLATE_500}${reset_5h_time}${RESET} ${SLATE_600}·${RESET} ${USAGE_RESET}WK:${RESET}${usage_7d_color}${usage_7d_int}%%${RESET} ${USAGE_RESET}↻${SLATE_500}${reset_7d_time}${RESET}"
    [ -n "$extra_display" ] && printf " ${SLATE_600}·${RESET} ${USAGE_EXTRA}${extra_display}${RESET}"
  fi
  ctx_bar=$(render_context_bar 24 $display_pct)
  printf " ${SLATE_600}·${RESET} ${ctx_warn}${ctx_bar} ${pct_color}${display_pct}%%${RESET}"
  if [ "$cache_total" -gt 0 ]; then
    cache_num_color=$(get_cache_color "$cache_hit_rate")
    printf " ${SLATE_600}·${RESET} ${CACHE_ICON}${NF_CACHE} ${cache_num_color}${cache_hit_rate}%%${RESET}"
  fi
  [ "$duration_sec" -gt 0 ] && printf " ${SLATE_600}·${RESET} ${COST_COLOR}${cost_fmt}${RESET} ${DURATION_COLOR}${duration_min}m${duration_secs}s${RESET}"
  printf "\n"
  ;;
esac
