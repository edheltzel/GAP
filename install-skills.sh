#!/usr/bin/env bash
#
# install-skills.sh - provision every skills.sh skill listed in skills.txt.
#
# Companion to the skills.txt manifest (dotfiles packages/*.txt model). For each
# manifest line `owner/repo[#commit]=skill,skill`, installs those skills globally
# into all detected agents via the skills.sh CLI. An optional `|path|git-blob`
# suffix verifies one installed SKILL.md against an immutable upstream blob.
# Full commit refs are fetched into a disposable local checkout before install,
# avoiding the skills CLI's branch-only ref interpretation. Exits non-zero if
# any source or integrity check fails.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST="$SCRIPT_DIR/skills.txt"
AGENTS="${SKILLS_AGENTS:-}"   # override: SKILLS_AGENTS="claude-code,pi,codex"
DRY_RUN=0

usage() {
  cat <<'EOF'
Usage: ./install-skills.sh [-n|--dry-run] [manifest]

  Installs skills from a manifest of `owner/repo[#commit]=skill,skill[|path|git-blob]` lines.
  A full commit ref requires the blob suffix because skills.sh treats #ref as a branch.
  (default: ./skills.txt) via `skills add <source> -g -s <skills> -y`.

Options:
  -n, --dry-run   Print the install commands without running them.
  -h, --help      Show this help.

Env:
  SKILLS_AGENTS   Comma-separated target agents (default: skills CLI detected agents).
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
  green=$'\e[32m'; yellow=$'\e[33m'; red=$'\e[31m'; cyan=$'\e[36m'; white=$'\e[37m'; clr=$'\e[0m'
else
  green=''; yellow=''; red=''; cyan=''; white=''; clr=''
fi

# bun-first per repo convention; never npm/npx.
if command -v skills >/dev/null 2>&1; then
  runner=(skills)
elif command -v bunx >/dev/null 2>&1; then
  runner=(bunx skills)
else
  printf '%sNeed the skills CLI - run: bun install -g skills%s\n' "$red" "$clr" >&2
  exit 1
fi

agent_args=()
if [ -n "$AGENTS" ]; then
  IFS=',' read -ra agent_names <<< "$AGENTS"
  for agent in "${agent_names[@]}"; do
    [ -n "$agent" ] && agent_args+=(-a "$agent")
  done
fi

[ -f "$MANIFEST" ] || { printf '%sManifest not found: %s%s\n' "$red" "$MANIFEST" "$clr" >&2; exit 1; }

ok=0
failed=()
pinned_root=''
prepared_source=''
cleanup() {
  [ -z "$pinned_root" ] || rm -rf "$pinned_root"
}
trap cleanup EXIT

prepare_pinned_source() {
  local source="$1" ref="$2"
  if ! command -v git >/dev/null 2>&1; then
    printf '%s    failed: git is required to install pinned source %s%s\n' "$red" "$source" "$clr" >&2
    return 1
  fi
  if [ -z "$pinned_root" ]; then
    pinned_root="$(mktemp -d "${TMPDIR:-/tmp}/atlas-pinned-skills.XXXXXX")"
  fi
  prepared_source="$pinned_root/${source//\//-}-${ref:0:12}"
  mkdir -p "$prepared_source"
  git -C "$prepared_source" init -q
  if ! git -C "$prepared_source" fetch --depth 1 -q "https://github.com/$source.git" "$ref" \
      || ! git -C "$prepared_source" checkout --detach -q FETCH_HEAD; then
    printf '%s    failed: could not fetch %s at %s%s\n' "$red" "$source" "$ref" "$clr" >&2
    return 1
  fi
}

verify_installed_blob() {
  local name="$1" source_path="$2" expected="$3" required="${4:-0}"
  local installed
  local actual

  installed="$HOME/.agents/skills/$name/$(basename "$source_path")"

  if [ ! -f "$installed" ]; then
    printf '%s    not installed: %s%s\n' "$yellow" "$installed" "$clr"
    [ "$required" -eq 0 ]
    return $?
  fi
  if ! command -v git >/dev/null 2>&1; then
    printf '%s    failed: git is required to verify %s%s\n' "$red" "$installed" "$clr" >&2
    return 1
  fi
  actual="$(git hash-object --path="$source_path" "$installed")"
  if [ "$actual" != "$expected" ]; then
    printf '%s    integrity mismatch: %s (expected git blob %s, got %s)%s\n' \
      "$red" "$installed" "$expected" "$actual" "$clr" >&2
    return 1
  fi
  printf '%s    integrity verified: %s (git blob %s)%s\n' "$green" "$installed" "$actual" "$clr"
}

while IFS= read -r line || [ -n "$line" ]; do
  # A # in an immutable source ref is data. Only a # at the beginning or after
  # whitespace starts a comment, preserving `owner/repo#<commit>` safely.
  if [[ "$line" =~ ^[[:space:]]*# ]]; then continue; fi
  if [[ "$line" =~ ^(.*[^[:space:]])[[:space:]]+#.*$ ]]; then line="${BASH_REMATCH[1]}"; fi
  line="$(printf '%s' "$line" | tr -d '[:space:]')"  # manifest tokens carry no spaces
  [ -z "$line" ] && continue
  source="${line%%=*}"
  descriptor="${line#*=}"
  { [ "$descriptor" = "$line" ] || [ -z "$descriptor" ]; } && descriptor='*'  # bare source → all skills

  ref=''
  if [[ "$source" == *#* ]]; then
    ref="${source#*#}"
    source="${source%%#*}"
    if [[ ! "$ref" =~ ^[0-9a-fA-F]{40}$ ]]; then
      printf '%sInvalid immutable ref in manifest entry: %s%s\n' "$red" "$line" "$clr" >&2
      failed+=("$line")
      continue
    fi
  fi

  integrity_path=''
  expected_blob=''
  IFS='|' read -ra descriptor_parts <<< "$descriptor"
  skills="${descriptor_parts[0]}"
  if [ "${#descriptor_parts[@]}" -ne 1 ]; then
    if [ "${#descriptor_parts[@]}" -ne 3 ] || [ -z "${descriptor_parts[1]}" ] || [ -z "${descriptor_parts[2]}" ]; then
      printf '%sInvalid integrity descriptor in manifest entry: %s%s\n' "$red" "$line" "$clr" >&2
      failed+=("$line")
      continue
    fi
    if [[ "$skills" == *,* ]]; then
      printf '%sIntegrity descriptors must select one skill: %s%s\n' "$red" "$line" "$clr" >&2
      failed+=("$line")
      continue
    fi
    integrity_path="${descriptor_parts[1]}"
    expected_blob="${descriptor_parts[2]}"
    if [[ "$integrity_path" = /* || "$integrity_path" == *..* || ! "$expected_blob" =~ ^[0-9a-fA-F]{40}$ ]]; then
      printf '%sInvalid integrity path or Git blob in manifest entry: %s%s\n' "$red" "$line" "$clr" >&2
      failed+=("$line")
      continue
    fi
  fi

  if [ -n "$ref" ] && [ -z "$integrity_path" ]; then
    printf '%sImmutable refs require an integrity descriptor: %s%s\n' "$red" "$line" "$clr" >&2
    failed+=("$line")
    continue
  fi

  display_source="$source"
  [ -n "$ref" ] && display_source="$source#$ref"
  install_source="$source"
  if [ -n "$ref" ]; then
    if [ "$DRY_RUN" -eq 1 ]; then
      install_source="<checkout:$source@$ref>"
    elif ! prepare_pinned_source "$source" "$ref"; then
      failed+=("$display_source")
      continue
    else
      install_source="$prepared_source"
    fi
  fi
  printf '%s==>%s %s %s(%s)%s' "$cyan" "$clr" "$display_source" "$yellow" "$skills" "$clr"
  if [ -n "$integrity_path" ]; then
    printf ' %s[integrity %s git-blob %s]%s' "$white" "$integrity_path" "$expected_blob" "$clr"
  fi
  printf '\n'
  # The skills CLI wants one -s per skill; a comma-joined value is read as a single
  # literal name and matches nothing. Expand "a,b,c" into -s a -s b -s c.
  skill_args=()
  IFS=',' read -ra names <<< "$skills"
  for name in "${names[@]}"; do
    [ -n "$name" ] && skill_args+=(-s "$name")
  done
  install_args=(add "$install_source" -g)
  if [ -n "$AGENTS" ]; then
    install_args+=("${agent_args[@]}")
  fi
  install_args+=("${skill_args[@]}" -y)
  if [ "$DRY_RUN" -eq 1 ]; then
    [ -n "$ref" ] && printf '    checkout: https://github.com/%s.git at %s\n' "$source" "$ref"
    printf '    %s' "${runner[*]}"
    printf ' %s' "${install_args[@]}"
    printf '\n'
    [ -n "$ref" ] && printf '    policy ref: %s (enforced by checkout and installed Git blob)\n' "$ref"
    if [ -n "$integrity_path" ]; then
      if verify_installed_blob "$skills" "$integrity_path" "$expected_blob"; then
        ok=$((ok + 1))
      else
        failed+=("$line")
      fi
    else
      ok=$((ok + 1))
    fi
    continue
  fi
  # Redirect stdin from /dev/null so the interactive CLI can't consume the manifest.
  if "${runner[@]}" "${install_args[@]}" </dev/null; then
    if [ -n "$integrity_path" ] && ! verify_installed_blob "$skills" "$integrity_path" "$expected_blob" 1; then
      failed+=("$line")
    else
      ok=$((ok + 1))
    fi
  else
    printf '%s    failed: %s%s\n' "$red" "$display_source" "$clr" >&2
    failed+=("$display_source")
  fi
done < "$MANIFEST"

printf '\n%sOK: %d source(s)%s\n' "$green" "$ok" "$clr"
if [ "${#failed[@]}" -gt 0 ]; then
  printf '%sFailed (%d): %s%s\n' "$red" "${#failed[@]}" "${failed[*]}" "$clr" >&2
  exit 1
fi
