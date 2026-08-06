#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temp_home="$(mktemp -d "${TMPDIR:-/tmp}/atlas-stow-safety.XXXXXX")"
trap 'rm -rf "$temp_home"' EXIT
just_cmd=(just --justfile "$repo_root/justfile")

"${just_cmd[@]}" jcode-filter
if [ "$(git -C "$repo_root" config --get merge.ours.driver)" != 'true' ]; then
  printf 'repository-local merge.ours.driver was not configured\n' >&2
  exit 1
fi

HOME="$temp_home" "${just_cmd[@]}" stow-preflight >/dev/null
HOME="$temp_home" "${just_cmd[@]}" status >/dev/null

mkdir -p "$temp_home/.jcode" "$temp_home/external"
printf '{"terminal":"external"}\n' > "$temp_home/external/preferred_terminal.json"
ln -s ../external/preferred_terminal.json "$temp_home/.jcode/preferred_terminal.json"
if HOME="$temp_home" "${just_cmd[@]}" stow-preflight jcode >/dev/null 2>&1; then
  printf 'Stow preflight accepted a valid wrong-source link\n' >&2
  exit 1
fi
if HOME="$temp_home" "${just_cmd[@]}" status >/dev/null 2>&1; then
  printf 'status hid a genuine Stow ownership conflict\n' >&2
  exit 1
fi
[ -L "$temp_home/.jcode/preferred_terminal.json" ]
[ -e "$temp_home/.jcode/preferred_terminal.json" ]

rm "$temp_home/.jcode/preferred_terminal.json"
ln -s ../external/missing.json "$temp_home/.jcode/preferred_terminal.json"
HOME="$temp_home" "${just_cmd[@]}" prune-broken jcode >/dev/null
if [ -L "$temp_home/.jcode/preferred_terminal.json" ]; then
  printf 'prune-broken did not remove a Stow-blocking dangling link\n' >&2
  exit 1
fi

for pkg in claude omp pi jcode; do
  stow -d "$repo_root" -t "$temp_home" --no-folding --restow "$pkg"
done

for dir in "$temp_home/.claude" "$temp_home/.omp" "$temp_home/.pi/agent" "$temp_home/.jcode"; do
  if [ ! -d "$dir" ] || [ -L "$dir" ]; then
    printf 'no-folding package directory is not real: %s\n' "$dir" >&2
    exit 1
  fi
done

[ -f "$temp_home/.claude/Atlas/atlas-statusline.sh" ]
[ -f "$temp_home/.omp/agent/extensions/atlas-status-line.ts" ]
[ -f "$temp_home/.pi/agent/extensions/atlas-status-line.ts" ]
[ -f "$temp_home/.jcode/preferred_terminal.json" ]

dangling="$(find "$temp_home/.claude" "$temp_home/.omp" "$temp_home/.pi" "$temp_home/.jcode" \
  -type l ! -path "$temp_home/.omp/agent/extensions/echo-voice" ! -exec test -e {} \; -print)"
if [ -n "$dangling" ]; then
  printf 'temporary all-package installation contains dangling links:\n%s\n' "$dangling" >&2
  exit 1
fi

for recipe in update link unlink status; do
  dry_run="$("${just_cmd[@]}" --dry-run "$recipe" 2>&1)"
  for pkg in claude omp pi jcode; do
    if ! grep -Fq "$pkg" <<<"$dry_run"; then
      printf 'aggregate recipe %s omits package %s\n' "$recipe" "$pkg" >&2
      exit 1
    fi
  done
  case "$recipe" in
    update|link)
      preflight_line="$(grep -n -m1 'Stow preflight passed' <<<"$dry_run" | cut -d: -f1)"
      manifest_line="$(grep -n -m1 './install-skills.sh' <<<"$dry_run" | cut -d: -f1)"
      if [ -z "$preflight_line" ] || [ -z "$manifest_line" ] || [ "$preflight_line" -ge "$manifest_line" ]; then
        printf 'aggregate recipe %s does not preflight before manifest provisioning\n' "$recipe" >&2
        exit 1
      fi
      ;;
  esac
done

printf 'PASS: Stow preflight, pruning, status, no-folding, and aggregate coverage are safe\n'
