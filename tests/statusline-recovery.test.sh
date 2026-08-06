#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
claude_status="$repo_root/claude/.claude/Atlas/atlas-statusline.sh"
pi_status="$repo_root/pi/.pi/agent/extensions/atlas-status-line.ts"
omp_status="$repo_root/omp/.omp/agent/extensions/atlas-status-line.ts"
temp_home="$(mktemp -d "${TMPDIR:-/tmp}/atlas-statusline-recovery.XXXXXX")"
trap 'rm -rf "$temp_home"' EXIT

for file in "$claude_status" "$pi_status" "$omp_status"; do
  if grep -Eq 'LIFEOS/VERSION|ATLAS_VERSION|readAtlasVersion' "$file"; then
    printf 'status line still depends on removed LifeOS version state: %s\n' "$file" >&2
    exit 1
  fi
done

mkdir -p "$temp_home/.claude/MEMORY/STATE"
claude_stderr="$temp_home/claude-statusline.stderr"
claude_output="$(
  printf '%s' '{"workspace":{"current_dir":"/tmp/Atlas/Config"},"model":{"id":"claude-sonnet-4-6","display_name":"Sonnet 4.6"},"version":"2.1.0","context_window":{"used_percentage":42,"current_usage":{"cache_read_input_tokens":99,"cache_creation_input_tokens":1}}}' \
    | HOME="$temp_home" ATLAS_DIR="$temp_home/.claude" COLUMNS=120 bash "$claude_status" 2>"$claude_stderr"
)"
if [ -s "$claude_stderr" ]; then
  printf 'Claude status line wrote to stderr:\n%s\n' "$(<"$claude_stderr")" >&2
  exit 1
fi
claude_plain="$(sed $'s/\033\\[[0-9;]*m//g' <<<"$claude_output")"
grep -F '✱ 2.1.0 · Sonnet 4.6' <<<"$claude_plain" >/dev/null
grep -F '…/Atlas/Config' <<<"$claude_plain" >/dev/null

: > "$claude_stderr"
printf '%s' '{"workspace":{"current_dir":"/tmp/Atlas/Config"}}' \
  | env -u COLUMNS HOME="$temp_home" ATLAS_DIR="$temp_home/.claude" bash "$claude_status" \
    >/dev/null 2>"$claude_stderr"
if [ -s "$claude_stderr" ]; then
  printf 'non-TTY Claude status fallback wrote to stderr:\n%s\n' "$(<"$claude_stderr")" >&2
  exit 1
fi

if grep -Fq 'Atlas v' <<<"$claude_plain"; then
  printf 'Claude status line rendered retired Atlas/LifeOS identity\n' >&2
  exit 1
fi

grep -F 'cc_display_plain="✱ ${cc_version:-?}"' "$claude_status" >/dev/null
grep -F '${color(palette.pi, SYMBOLS.pi)} ${color(palette.text, VERSION)}' "$pi_status" >/dev/null
grep -F 'content: `${color(palette.omp, SYMBOLS.omp)} ${color(palette.text, sanitizeLabel(state.ompVersion))}`' "$omp_status" >/dev/null

if grep -Fq '▲ Atlas' "$pi_status" || grep -Fq '▲ Atlas' "$omp_status"; then
  printf 'Pi or OMP status line still renders retired Atlas/LifeOS identity\n' >&2
  exit 1
fi

[ -f "$repo_root/pi/.pi/agent/themes/eldritch.json" ]
[ ! -e "$repo_root/pi/.pi/agent/themes/atlas-eldritch.json" ]
[ "$(jq -r '.name' "$repo_root/pi/.pi/agent/themes/eldritch.json")" = 'eldritch' ]
[ "$(jq -r '.theme' "$repo_root/pi/.pi/agent/settings.json")" = 'eldritch' ]

[ -f "$repo_root/omp/.omp/agent/themes/atlas-default.json" ]
[ -f "$repo_root/omp/.omp/agent/themes/atlas-eldritch.json" ]
grep -F 'dark: atlas-eldritch' "$repo_root/omp/.omp/agent/config.yml" >/dev/null

printf 'PASS: Claude, Pi, and OMP use harness-native status identity and curated themes\n'
