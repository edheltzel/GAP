#!/usr/bin/env bash
#
# install-skills.sh — provision every skills.sh skill listed in skills.txt.
#
# Companion to the skills.txt manifest (dotfiles packages/*.txt model). For each
# manifest line `owner/repo=skill,skill`, installs those skills globally into all
# detected agents via the skills.sh CLI. Exits non-zero if any source fails.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST="$SCRIPT_DIR/skills.txt"
AGENTS="${SKILLS_AGENTS:-*}"   # override: SKILLS_AGENTS="claude-code,pi,codex"
DRY_RUN=0

usage() {
  cat <<'EOF'
Usage: ./install-skills.sh [-n|--dry-run] [manifest]

  Installs skills from a manifest of `owner/repo=skill,skill` lines
  (default: ./skills.txt) via `skills add <source> -g -a <agents> -s <skills> -y`.

Options:
  -n, --dry-run   Print the install commands without running them.
  -h, --help      Show this help.

Env:
  SKILLS_AGENTS   Comma-separated target agents (default: * = all detected).
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    -n|--dry-run) DRY_RUN=1 ;;
    -h|--help) usage; exit 0 ;;
    -*) printf 'Unknown option: %s\n' "$1" >&2; usage >&2; exit 1 ;;
    *) MANIFEST="$1" ;;
  esac
  shift
done

if [ -t 1 ]; then
  green=$'\e[32m'; yellow=$'\e[33m'; red=$'\e[31m'; cyan=$'\e[36m'; clr=$'\e[0m'
else
  green=''; yellow=''; red=''; cyan=''; clr=''
fi

# bun-first per repo convention; never npm/npx.
if command -v skills >/dev/null 2>&1; then
  runner=(skills)
elif command -v bunx >/dev/null 2>&1; then
  runner=(bunx skills)
else
  printf '%sNeed the skills CLI — run: bun install -g skills%s\n' "$red" "$clr" >&2
  exit 1
fi

[ -f "$MANIFEST" ] || { printf '%sManifest not found: %s%s\n' "$red" "$MANIFEST" "$clr" >&2; exit 1; }

ok=0
failed=()
while IFS= read -r line || [ -n "$line" ]; do
  line="${line%%#*}"                                 # strip comments
  line="$(printf '%s' "$line" | tr -d '[:space:]')"  # manifest tokens carry no spaces
  [ -z "$line" ] && continue
  source="${line%%=*}"
  skills="${line#*=}"
  { [ "$skills" = "$line" ] || [ -z "$skills" ]; } && skills='*'  # bare source → all skills
  printf '%s==>%s %s %s(%s)%s\n' "$cyan" "$clr" "$source" "$yellow" "$skills" "$clr"
  # The skills CLI wants one -s per skill; a comma-joined value is read as a single
  # literal name and matches nothing. Expand "a,b,c" into -s a -s b -s c.
  skill_args=()
  IFS=',' read -ra names <<< "$skills"
  for name in "${names[@]}"; do
    [ -n "$name" ] && skill_args+=(-s "$name")
  done
  if [ "$DRY_RUN" -eq 1 ]; then
    printf '    %s add %s -g -a %s %s -y\n' "${runner[*]}" "$source" "$AGENTS" "${skill_args[*]}"
    ok=$((ok + 1))
    continue
  fi
  # Redirect stdin from /dev/null so the interactive CLI can't consume the manifest.
  if "${runner[@]}" add "$source" -g -a "$AGENTS" "${skill_args[@]}" -y </dev/null; then
    ok=$((ok + 1))
  else
    printf '%s    failed: %s%s\n' "$red" "$source" "$clr" >&2
    failed+=("$source")
  fi
done < "$MANIFEST"

printf '\n%sOK: %d source(s)%s\n' "$green" "$ok" "$clr"
if [ "${#failed[@]}" -gt 0 ]; then
  printf '%sFailed (%d): %s%s\n' "$red" "${#failed[@]}" "${failed[*]}" "$clr" >&2
  exit 1
fi
