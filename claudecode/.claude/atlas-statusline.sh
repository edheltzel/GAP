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
# Output order: Header → Model+CC+Skills → Usage+Memory → Context Bar → Learning → Quote
#
# Context percentage scales to compaction threshold if configured in settings.json.
# When contextDisplay.compactionThreshold is set (e.g., 62), the bar shows 62% as 100%.
# Set threshold to 100 or remove the setting to show raw 0-100% from Claude Code.
# ═══════════════════════════════════════════════════════════════════════════════

set -o pipefail

# ─────────────────────────────────────────────────────────────────────────────
# CONFIGURATION
# ─────────────────────────────────────────────────────────────────────────────

PAI_DIR="${PAI_DIR:-$HOME/.claude}"
SETTINGS_FILE="$PAI_DIR/settings.json"
RATINGS_FILE="$PAI_DIR/MEMORY/LEARNING/SIGNALS/ratings.jsonl"
MODEL_CACHE="$PAI_DIR/MEMORY/STATE/model-cache.txt"
QUOTE_CACHE="$PAI_DIR/.quote-cache"
USAGE_CACHE="$PAI_DIR/MEMORY/STATE/usage-cache.json"
LEARNING_CACHE="$PAI_DIR/MEMORY/STATE/learning-cache.sh"
LEARNING_CACHE_TTL=30

# Cache TTL in seconds
COUNTS_CACHE_TTL=30 # 30 seconds
USAGE_CACHE_TTL=60  # 60 seconds

# Source .env for API keys
[ -f "${PAI_CONFIG_DIR:-$HOME/.config/PAI}/.env" ] && source "${PAI_CONFIG_DIR:-$HOME/.config/PAI}/.env"

# ─────────────────────────────────────────────────────────────────────────────
# PARSE INPUT (must happen before parallel block consumes stdin)
# ─────────────────────────────────────────────────────────────────────────────

input=$(cat)

# Get DA name from settings
DA_NAME=$(jq -r '.daidentity.name // .daidentity.displayName // .env.DA // "Assistant"' "$SETTINGS_FILE" 2>/dev/null)
DA_NAME="${DA_NAME:-Assistant}"

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
  "duration_ms=" + (.cost.total_duration_ms // 0 | tostring) + "\n" +
  "context_max=" + (.context_window.context_window_size // 200000 | tostring) + "\n" +
  "context_pct=" + (.context_window.used_percentage // 0 | tostring) + "\n" +
  "context_remaining=" + (.context_window.remaining_percentage // 100 | tostring) + "\n" +
  "total_input=" + (.context_window.total_input_tokens // 0 | tostring) + "\n" +
  "total_output=" + (.context_window.total_output_tokens // 0 | tostring) + "\n" +
  "cache_read=" + (.context_window.current_usage.cache_read_input_tokens // 0 | tostring) + "\n" +
  "cache_create=" + (.context_window.current_usage.cache_creation_input_tokens // 0 | tostring)
' 2>/dev/null)"

# Ensure defaults for critical numeric values
context_pct=${context_pct:-0}
context_max=${context_max:-200000}
context_remaining=${context_remaining:-100}
total_input=${total_input:-0}
total_output=${total_output:-0}
cache_read=${cache_read:-0}
cache_create=${cache_create:-0}

# If used_percentage is 0 but we have token data, calculate manually
if [ "$context_pct" = "0" ] && [ "$total_input" -gt 0 ]; then
  total_tokens=$((total_input + total_output))
  context_pct=$((total_tokens * 100 / context_max))
fi

# Cache model name for other tools
mkdir -p "$(dirname "$MODEL_CACHE")" 2>/dev/null
echo "$model_name" >"$MODEL_CACHE" 2>/dev/null

# Parse model version from ID (e.g. "claude-sonnet-4-6" → "4.6")
model_version=$(echo "${model_id:-}" | sed -E 's/claude-[a-z]+-([0-9]+)-([0-9]+).*/\1.\2/' 2>/dev/null)
[ -z "$model_version" ] || [ "$model_version" = "${model_id:-}" ] && model_version=""
# Don't show version if display_name already contains it (e.g. "Sonnet 4.6")
[ -n "$model_version" ] && echo "$model_name" | grep -qF "$model_version" 2>/dev/null && model_version=""
# Split model display: name prefix in PAI_A, version suffix in SLATE_300 white
# e.g. "Sonnet 4.6" → model_name_short="Sonnet", model_ver_str="4.6"
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
# PARALLEL PREFETCH - Launch ALL expensive operations immediately
# ─────────────────────────────────────────────────────────────────────────────

_parallel_tmp="/tmp/pai-parallel-$$"
mkdir -p "$_parallel_tmp"

# --- PARALLEL BLOCK START ---
{
  # 1. All counts from settings.json (updated by StopOrchestrator → UpdateCounts)
  if jq -e '.counts' "$SETTINGS_FILE" >/dev/null 2>&1; then
    jq -r '
            "skills_count=" + (.counts.skills // 0 | tostring) + "\n" +
            "workflows_count=" + (.counts.workflows // 0 | tostring) + "\n" +
            "hooks_count=" + (.counts.hooks // 0 | tostring) + "\n" +
            "learnings_count=" + (.counts.signals // 0 | tostring) + "\n" +
            "work_count=" + (.counts.work // 0 | tostring) + "\n" +
            "sessions_count=" + (.counts.sessions // 0 | tostring) + "\n" +
            "research_count=" + (.counts.research // 0 | tostring) + "\n" +
            "ratings_count=" + (.counts.ratings // 0 | tostring)
        ' "$SETTINGS_FILE" >"$_parallel_tmp/counts.sh" 2>/dev/null
  else
    cat >"$_parallel_tmp/counts.sh" <<COUNTSEOF
skills_count=65
workflows_count=339
hooks_count=18
learnings_count=3000
work_count=0
sessions_count=0
research_count=0
ratings_count=0
COUNTSEOF
  fi
} &

{
  # 2. Usage data — refresh from Anthropic API if cache is stale
  cache_age=999999
  [ -f "$USAGE_CACHE" ] && cache_age=$(($(date +%s) - $(stat -f %m "$USAGE_CACHE" 2>/dev/null || echo 0)))

  if [ "$cache_age" -gt "$USAGE_CACHE_TTL" ]; then
    keychain_data=$(security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null)
    token=$(echo "$keychain_data" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('claudeAiOauth',{}).get('accessToken',''))" 2>/dev/null)

    if [ -n "$token" ]; then
      usage_json=$(curl -s --max-time 3 \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -H "anthropic-beta: oauth-2025-04-20" \
        "https://api.anthropic.com/api/oauth/usage" 2>/dev/null)

      if [ -n "$usage_json" ] && echo "$usage_json" | jq -e '.five_hour' >/dev/null 2>&1; then
        if [ -f "$USAGE_CACHE" ]; then
          ws_cost=$(jq -r '.workspace_cost // empty' "$USAGE_CACHE" 2>/dev/null)
          if [ -n "$ws_cost" ] && [ "$ws_cost" != "null" ]; then
            usage_json=$(echo "$usage_json" | jq --argjson ws "$ws_cost" '. + {workspace_cost: $ws}' 2>/dev/null || echo "$usage_json")
          fi
        fi
        echo "$usage_json" | jq '.' >"$USAGE_CACHE" 2>/dev/null
      fi
    fi
  fi

  if [ -f "$USAGE_CACHE" ]; then
    jq -r '
            "usage_5h=" + (.five_hour.utilization // 0 | tostring) + "\n" +
            "usage_5h_reset=" + (.five_hour.resets_at // "" | @sh) + "\n" +
            "usage_7d=" + (.seven_day.utilization // 0 | tostring) + "\n" +
            "usage_7d_reset=" + (.seven_day.resets_at // "" | @sh) + "\n" +
            "usage_extra_enabled=" + (.extra_usage.is_enabled // false | tostring) + "\n" +
            "usage_extra_limit=" + (.extra_usage.monthly_limit // 0 | tostring) + "\n" +
            "usage_extra_used=" + (.extra_usage.used_credits // 0 | tostring)
        ' "$USAGE_CACHE" >"$_parallel_tmp/usage.sh" 2>/dev/null
  else
    echo -e "usage_5h=0\nusage_7d=0\nusage_extra_enabled=false" >"$_parallel_tmp/usage.sh"
  fi
} &

{
  # 3. Quote prefetch
  quote_age=$(($(date +%s) - $(stat -f %m "$QUOTE_CACHE" 2>/dev/null || echo 0)))
  if [ "$quote_age" -gt 300 ] || [ ! -f "$QUOTE_CACHE" ]; then
    if [ -n "${ZENQUOTES_API_KEY:-}" ]; then
      new_quote=$(curl -s --max-time 1 "https://zenquotes.io/api/random/${ZENQUOTES_API_KEY}" 2>/dev/null |
        jq -r '.[0] | select(.q | length < 80) | .q + "|" + .a' 2>/dev/null)
      [ -n "$new_quote" ] && [ "$new_quote" != "null" ] && echo "$new_quote" >"$QUOTE_CACHE"
    fi
  fi
} &

# --- PARALLEL BLOCK END ---
wait

# Source all parallel results
[ -f "$_parallel_tmp/counts.sh" ] && source "$_parallel_tmp/counts.sh"
[ -f "$_parallel_tmp/usage.sh" ] && source "$_parallel_tmp/usage.sh"
rm -rf "$_parallel_tmp" 2>/dev/null

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

  # Tier 1: Kitty IPC (most accurate for Kitty panes)
  if [ -n "$KITTY_WINDOW_ID" ] && command -v kitten >/dev/null 2>&1; then
    width=$(kitten @ ls 2>/dev/null | jq -r --argjson wid "$KITTY_WINDOW_ID" \
      '.[].tabs[].windows[] | select(.id == $wid) | .columns' 2>/dev/null)
  fi

  # Tier 2: Direct TTY query
  [ -z "$width" ] || [ "$width" = "0" ] || [ "$width" = "null" ] &&
    width=$(stty size </dev/tty 2>/dev/null | awk '{print $2}')

  # Tier 3: tput fallback
  [ -z "$width" ] || [ "$width" = "0" ] && width=$(tput cols 2>/dev/null)

  # Tier 4: Environment variable
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
SLATE_400='\033[38;2;164;140;242m' # Lovecraft Purple (labels)
SLATE_500='\033[38;2;112;129;208m' # The Old One Purple (muted)
SLATE_600='\033[38;2;50;52;73m'    # Shallow Depths Grey (separators)

# Semantic colors
EMERALD='\033[38;2;55;244;153m' # Great Old One Green (positive/success)
ROSE='\033[38;2;241;108;117m'   # R'lyeh Red (error/negative)

# Rating gradient (for get_rating_color)
RATING_10='\033[38;2;55;244;153m'   # 9-10: Great Old One Green
RATING_8='\033[38;2;4;209;249m'     # 8: Watery Tomb Blue (cyan)
RATING_7='\033[38;2;241;252;121m'   # 7: Gold of Yuggoth (yellow)
RATING_6='\033[38;2;247;198;127m'   # 6: Dreaming Orange
RATING_5='\033[38;2;247;198;127m'   # 5: Dreaming Orange (softer)
RATING_4='\033[38;2;242;101;181m'   # 4: Pustule Pink
RATING_LOW='\033[38;2;241;108;117m' # 0-3: R'lyeh Red

# Header / Branding (Lovecraft Purple theme)
GREET_PRIMARY='\033[38;2;164;140;242m' # Lovecraft Purple
PAI_SESSION='\033[38;2;242;101;181m'   # Pustule Pink for session name

# Model+Skills line (Cyan/Green theme)
WIELD_ACCENT='\033[38;2;4;209;249m'     # Cyan accent (skills)
WIELD_WORKFLOWS='\033[38;2;55;244;153m' # Green for workflows
WIELD_HOOKS='\033[38;2;242;101;181m'    # Pustule Pink for hooks
CACHE_ICON='\033[38;2;164;140;242m'     # Lovecraft Purple for cache icon
CC_ICON='\033[38;2;232;97;32m'          # Anthropic Orange for ✱ CC icon
# Nerd font icons — generated via printf to preserve exact UTF-8 bytes
NF_SKILLS=$(printf '\xf3\xb0\x83\x97')   # nf-md-briefcase_check  U+F00D7
NF_WORKFLOW=$(printf '\xf3\xb0\x93\xa6') # nf-md-sync             U+F04E6
NF_HOOK=$(printf '\xf3\xb0\x9b\xa2')     # nf-md-hook             U+F06E2
NF_CACHE=$(printf '\xef\x92\x9b')        # nf-oct-cache           U+F49B
NF_GAUGE=$(printf '\xf3\xb0\x8a\x9a')    # nf-md-gauge            U+F029A
NF_HAMMER=$(printf '\xf3\xb0\xa3\xaa')   # nf-md-hammer           U+F08EA  (work)
NF_WIFI=$(printf '\xf3\xb0\x96\xa9')     # nf-md-wifi             U+F05A9  (signals)
NF_MESSAGE=$(printf '\xf3\xb0\x8d\xa5')  # nf-md-message_outline  U+F0365  (sessions)
NF_SEARCH=$(printf '\xf3\xb0\x8d\x89')   # nf-md-magnify          U+F0349  (research)

# Memory counts (Purple theme)
LEARN_WORK='\033[38;2;4;209;249m'       # Cyan for work
LEARN_SIGNALS='\033[38;2;241;252;121m'  # Gold of Yuggoth for ratings
LEARN_RESEARCH='\033[38;2;55;244;153m'  # Green for research
LEARN_SESSIONS='\033[38;2;242;101;181m' # Pustule Pink for sessions

# Learning ratings
SIGNAL_PERIOD='\033[38;2;164;140;242m' # Lovecraft Purple (period labels)
LEARN_LABEL='\033[38;2;55;244;153m'    # Great Old One Green for ★

# Context bar
CTX_BUCKET_EMPTY='\033[38;2;50;52;73m' # Shallow Depths Grey (empty bucket)

# Usage (Orange/Yellow theme)
USAGE_PRIMARY='\033[38;2;247;198;127m' # Dreaming Orange icon
USAGE_RESET='\033[38;2;112;129;208m'   # The Old One Purple for reset time
USAGE_EXTRA='\033[38;2;242;101;181m'   # Pustule Pink for overage

# Quote (Yellow/Orange theme)
QUOTE_PRIMARY='\033[38;2;241;252;121m' # Gold of Yuggoth
QUOTE_AUTHOR='\033[38;2;247;198;127m'  # Dreaming Orange

# PAI Branding
PAI_A='\033[38;2;164;140;242m' # Lovecraft Purple

# ─────────────────────────────────────────────────────────────────────────────
# HELPER FUNCTIONS
# ─────────────────────────────────────────────────────────────────────────────

# Get color for rating value (handles "—" for no data)
get_rating_color() {
  local val="$1"
  [[ "$val" == "—" || -z "$val" ]] && {
    echo "$SLATE_400"
    return
  }
  local rating_int=${val%%.*}
  [[ ! "$rating_int" =~ ^[0-9]+$ ]] && {
    echo "$SLATE_400"
    return
  }

  if [ "$rating_int" -ge 9 ]; then
    echo "$RATING_10"
  elif [ "$rating_int" -ge 8 ]; then
    echo "$RATING_8"
  elif [ "$rating_int" -ge 7 ]; then
    echo "$RATING_7"
  elif [ "$rating_int" -ge 6 ]; then
    echo "$RATING_6"
  elif [ "$rating_int" -ge 5 ]; then
    echo "$RATING_5"
  elif [ "$rating_int" -ge 4 ]; then
    echo "$RATING_4"
  else
    echo "$RATING_LOW"
  fi
}

# Sparkline / gradient step renderer — takes comma-separated integers 1-10
# Each bar encodes both HEIGHT (▁→█ = rating magnitude) and COLOR (red→green = quality)
SPARKLINE_BARS=('▁' '▂' '▃' '▄' '▅' '▆' '▇' '█')
render_sparkline() {
  local vals="$1"
  [ -z "$vals" ] && return
  local spark="" r idx bar_color
  for r in $(echo "$vals" | tr ',' ' '); do
    idx=$(((r - 1) * 7 / 9))
    [ "$idx" -lt 0 ] && idx=0
    [ "$idx" -gt 7 ] && idx=7
    bar_color=$(get_rating_color "$r")
    spark="${spark}${bar_color}${SPARKLINE_BARS[$idx]}${RESET}"
  done
  printf "%s" "$spark"
}

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

# Render full-width context bar spanning terminal (mini/normal modes)
render_context_bar_full() {
  local pct=$1
  local right_str=" ${pct}%"
  local right_len=${#right_str}
  local bar_width=$((term_width - right_len))
  [ "$bar_width" -gt 72 ] && bar_width=72
  [ "$bar_width" -lt 8 ] && bar_width=8
  local filled=$((pct * bar_width / 100))
  [ "$filled" -lt 0 ] && filled=0
  [ "$filled" -gt "$bar_width" ] && filled="$bar_width"

  local i=1 color
  while [ "$i" -le "$bar_width" ]; do
    if [ "$i" -le "$filled" ]; then
      color=$(get_bucket_color $i $bar_width)
      printf "${color}⛁${RESET}"
    else
      printf "${CTX_BUCKET_EMPTY}⛁${RESET}"
    fi
    i=$((i + 1))
  done
  printf " ${pct_color}${pct}%%${RESET}\n"
}

# ═══════════════════════════════════════════════════════════════════════════════
# LINE 0: ATLAS HEADER
# ═══════════════════════════════════════════════════════════════════════════════

session_display=""
[ -n "$SESSION_LABEL" ] && session_display=$(echo "$SESSION_LABEL" | tr '[:lower:]' '[:upper:]')

# Build version suffix: model + CC version
ver_suffix=""
ver_suffix_plain=""
ver_suffix="${PAI_A}${model_name_short}${RESET}"
ver_suffix_plain="${model_name_short}"
[ -n "$model_ver_str" ] && { ver_suffix="${ver_suffix} ${SLATE_300}${model_ver_str}${RESET}"; ver_suffix_plain="${ver_suffix_plain} ${model_ver_str}"; }
[ -n "$cc_version" ] && { ver_suffix="${ver_suffix} ${SLATE_600}·${RESET} ${CC_ICON}✱ ${SLATE_300}${cc_version}${RESET}"; ver_suffix_plain="${ver_suffix_plain} · ${cc_version}"; }

case "$MODE" in
nano)
  printf "${SLATE_600}── ·${RESET} ${GREET_PRIMARY}▲ Atlas v${PAI_VERSION}${RESET} ${SLATE_600}·${RESET} ${ver_suffix}\n"
  ;;
*)
  if [ -n "$session_display" ]; then
    local_left="── · ▲ Atlas v${PAI_VERSION} · ${ver_suffix_plain} · ${session_display} "
    local_left_len=${#local_left}
    local_fill=$((72 - local_left_len))
    [ "$local_fill" -lt 2 ] && local_fill=2
    local_dashes=$(printf '%*s' "$local_fill" '' | tr ' ' '─')
    printf "${SLATE_600}── ·${RESET} ${GREET_PRIMARY}▲ Atlas v${PAI_VERSION}${RESET} ${SLATE_600}·${RESET} ${ver_suffix} ${SLATE_600}·${RESET} ${PAI_SESSION}${session_display}${RESET} ${SLATE_600}${local_dashes}${RESET}\n"
  else
    local_left="── · ▲ Atlas v${PAI_VERSION} · ${ver_suffix_plain} "
    local_left_len=${#local_left}
    local_fill=$((72 - local_left_len))
    [ "$local_fill" -lt 2 ] && local_fill=2
    local_dashes=$(printf '%*s' "$local_fill" '' | tr ' ' '─')
    printf "${SLATE_600}── ·${RESET} ${GREET_PRIMARY}▲ Atlas v${PAI_VERSION}${RESET} ${SLATE_600}·${RESET} ${ver_suffix} ${SLATE_600}${local_dashes}${RESET}\n"
  fi
  ;;
esac

# ═══════════════════════════════════════════════════════════════════════════════
# LINE 2: USAGE + MEMORY
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

# Usage data
usage_5h_int=${usage_5h%%.*}
usage_7d_int=${usage_7d%%.*}
[ -z "$usage_5h_int" ] && usage_5h_int=0
[ -z "$usage_7d_int" ] && usage_7d_int=0

reset_5h_time=""
reset_7d_time=""
extra_display=""
if [ "$usage_5h_int" -gt 0 ] || [ "$usage_7d_int" -gt 0 ] || [ -f "$USAGE_CACHE" ]; then
  usage_5h_color=$(get_usage_color "$usage_5h_int")
  usage_7d_color=$(get_usage_color "$usage_7d_int")

  eval "$(python3 -c "
from datetime import datetime, timezone
from zoneinfo import ZoneInfo

def parse_ts(ts):
    if not ts: return None
    try:
        if '+' in ts[10:]:
            return datetime.fromisoformat(ts)
        elif ts.endswith('Z'):
            return datetime.fromisoformat(ts.replace('Z', '+00:00'))
        else:
            return datetime.fromisoformat(ts + '+00:00')
    except: return None

def clock_time(ts, fmt):
    dt = parse_ts(ts)
    if not dt: return ''
    local_dt = dt.astimezone(ZoneInfo('America/Los_Angeles'))
    if fmt == 'weekly':
        return local_dt.strftime('%a %H:%M')
    return local_dt.strftime('%H:%M')

r5h = '$usage_5h_reset'
r7d = '$usage_7d_reset'
print(f\"clock_5h='{clock_time(r5h, 'hourly')}'\")
print(f\"clock_7d='{clock_time(r7d, 'weekly')}'\")
" 2>/dev/null)"

  reset_5h_time="${clock_5h:-}"
  reset_7d_time="${clock_7d:-}"

  if [ "$usage_extra_enabled" = "true" ]; then
    extra_limit_dollars=$((${usage_extra_limit:-0} / 100))
    extra_used_dollars=$((${usage_extra_used%%.*} / 100))
    if [ "${extra_limit_dollars:-0}" -ge 1000 ]; then
      extra_limit_fmt="\$$((extra_limit_dollars / 1000))K"
    else
      extra_limit_fmt="\$${extra_limit_dollars:-0}"
    fi
    extra_display="\$${extra_used_dollars:-0}/${extra_limit_fmt}"
  fi
fi

# Usage + context bar + cache
case "$MODE" in
nano)
  [ -n "$reset_5h_time" ] && printf "${USAGE_PRIMARY}${NF_GAUGE}${RESET} ${usage_5h_color}${usage_5h_int}%%${RESET} ${USAGE_RESET}↻${reset_5h_time}${RESET}"
  ctx_bar=$(render_context_bar 6 $display_pct)
  printf " ${SLATE_600}·${RESET} ${ctx_bar} ${pct_color}${display_pct}%%${RESET}"
  if [ "$cache_total" -gt 0 ]; then
    cache_num_color=$(get_cache_color "$cache_hit_rate")
    printf " ${SLATE_600}·${RESET} ${CACHE_ICON}${NF_CACHE} ${cache_num_color}${cache_hit_rate}%%${RESET}"
  fi
  printf "\n"
  ;;
micro)
  [ -n "$reset_5h_time" ] && printf "${USAGE_PRIMARY}${NF_GAUGE}${RESET} ${USAGE_RESET}5H:${RESET}${usage_5h_color}${usage_5h_int}%%${RESET} ${USAGE_RESET}↻${reset_5h_time}${RESET}"
  ctx_bar=$(render_context_bar 8 $display_pct)
  printf " ${SLATE_600}·${RESET} ${ctx_bar} ${pct_color}${display_pct}%%${RESET}"
  if [ "$cache_total" -gt 0 ]; then
    cache_num_color=$(get_cache_color "$cache_hit_rate")
    printf " ${SLATE_600}·${RESET} ${CACHE_ICON}${NF_CACHE} ${cache_num_color}${cache_hit_rate}%%${RESET}"
  fi
  printf "\n"
  ;;
mini)
  if [ -n "$reset_5h_time" ]; then
    printf "${USAGE_PRIMARY}${NF_GAUGE}${RESET} ${USAGE_RESET}5H:${RESET}${usage_5h_color}${usage_5h_int}%%${RESET} ${USAGE_RESET}↻${SLATE_500}${reset_5h_time}${RESET} ${SLATE_600}·${RESET} ${USAGE_RESET}WK:${RESET}${usage_7d_color}${usage_7d_int}%%${RESET} ${USAGE_RESET}↻${SLATE_500}${reset_7d_time}${RESET}"
  fi
  ctx_bar=$(render_context_bar 16 $display_pct)
  printf " ${SLATE_600}·${RESET} ${ctx_bar} ${pct_color}${display_pct}%%${RESET}"
  if [ "$cache_total" -gt 0 ]; then
    cache_num_color=$(get_cache_color "$cache_hit_rate")
    printf " ${SLATE_600}·${RESET} ${CACHE_ICON}${NF_CACHE} ${cache_num_color}${cache_hit_rate}%%${RESET}"
  fi
  printf "\n"
  ;;
normal)
  if [ -n "$reset_5h_time" ]; then
    printf "${USAGE_PRIMARY}${NF_GAUGE}${RESET} ${USAGE_RESET}5H:${RESET}${usage_5h_color}${usage_5h_int}%%${RESET} ${USAGE_RESET}↻${SLATE_500}${reset_5h_time}${RESET} ${SLATE_600}·${RESET} ${USAGE_RESET}WK:${RESET}${usage_7d_color}${usage_7d_int}%%${RESET} ${USAGE_RESET}↻${SLATE_500}${reset_7d_time}${RESET}"
    [ -n "$extra_display" ] && printf " ${SLATE_600}·${RESET} ${USAGE_EXTRA}${extra_display}${RESET}"
  fi
  ctx_bar=$(render_context_bar 24 $display_pct)
  printf " ${SLATE_600}·${RESET} ${ctx_bar} ${pct_color}${display_pct}%%${RESET}"
  if [ "$cache_total" -gt 0 ]; then
    cache_num_color=$(get_cache_color "$cache_hit_rate")
    printf " ${SLATE_600}·${RESET} ${CACHE_ICON}${NF_CACHE} ${cache_num_color}${cache_hit_rate}%%${RESET}"
  fi
  printf "\n"
  ;;
esac


# ═══════════════════════════════════════════════════════════════════════════════
# LINE 5: QUOTE (normal mode only)
# ═══════════════════════════════════════════════════════════════════════════════

if [ "$MODE" = "normal" ]; then
  if [ -f "$QUOTE_CACHE" ]; then
    IFS='|' read -r quote_text quote_author <"$QUOTE_CACHE"
    author_suffix="\" —${quote_author}"
    author_len=${#author_suffix}
    quote_len=${#quote_text}
    max_line=72
    full_len=$((quote_len + author_len + 4))

    if [ "$full_len" -le "$max_line" ]; then
      printf "${QUOTE_PRIMARY}✦${RESET} ${SLATE_400}\"${quote_text}\"${RESET} ${QUOTE_AUTHOR}—${quote_author}${RESET}\n"
    else
      line1_text_max=60
      min_line2=12
      target_line1=$line1_text_max
      [ "$target_line1" -gt "$quote_len" ] && target_line1=$((quote_len - min_line2))

      first_part="${quote_text:0:$target_line1}"
      remaining="${quote_text:$target_line1}"

      if [ -n "$remaining" ] && [ "${remaining:0:1}" != " " ]; then
        temp="$first_part"
        last_space_pos=0
        pos=0
        while [ $pos -lt ${#temp} ]; do
          [ "${temp:$pos:1}" = " " ] && last_space_pos=$pos
          pos=$((pos + 1))
        done
        [ $last_space_pos -gt 10 ] && first_part="${quote_text:0:$last_space_pos}"
      fi

      second_part="${quote_text:${#first_part}}"
      second_part="${second_part# }"

      if [ ${#second_part} -lt 10 ]; then
        printf "${QUOTE_PRIMARY}✦${RESET} ${SLATE_400}\"${quote_text}\"${RESET} ${QUOTE_AUTHOR}—${quote_author}${RESET}\n"
      else
        printf "${QUOTE_PRIMARY}✦${RESET} ${SLATE_400}\"${first_part}${RESET}\n"
        printf "  ${SLATE_400}${second_part}\"${RESET} ${QUOTE_AUTHOR}—${quote_author}${RESET}\n"
      fi
    fi
  fi
fi
