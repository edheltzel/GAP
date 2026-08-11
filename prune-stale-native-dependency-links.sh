#!/usr/bin/env bash
# Remove only dangling links under ignored native Claude node_modules paths.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source_root="${NATIVE_SKILLS_SOURCE_ROOT:-$repo_root/claude/.claude/skills}"
runtime_root="${CLAUDE_SKILLS_ROOT:-$HOME/.claude/skills}"
removed=0

remove_link() {
	local link="$1"
	printf 'pruned stale native dependency link: %s -> %s\n' "$link" "$(readlink "$link")"
	if command -v trash >/dev/null 2>&1; then
		trash "$link"
	else
		rm "$link"
	fi
	removed=$((removed + 1))
}

while IFS= read -r -d '' source_skill; do
	name="${source_skill##*/}"
	runtime_skill="$runtime_root/$name"
	[ -d "$runtime_skill" ] || [ -L "$runtime_skill" ] || continue
	while IFS= read -r -d '' link; do
		case "$link" in
		*/node_modules | */node_modules/*)
			if [ -L "$link" ] && [ ! -e "$link" ]; then
				remove_link "$link"
			fi
			;;
		esac
	done < <(find "$runtime_skill" -type l -print0 2>/dev/null)
done < <(find "$source_root" -mindepth 1 -maxdepth 1 -type d -print0)

if [ "$removed" -eq 0 ]; then
	printf 'No stale native dependency links found\n'
else
	printf 'Pruned %d stale native dependency link(s)\n' "$removed"
fi
