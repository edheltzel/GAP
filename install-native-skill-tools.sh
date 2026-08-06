#!/usr/bin/env bash
# Install or validate dependencies for repository-owned Claude skill tools.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
mode=install

usage() {
	cat <<'EOF'
Usage: install-native-skill-tools.sh [--check]

With no arguments, installs locked dependencies beside the real repository files.
--check validates package locks and installed runtime and development dependencies without mutation.
EOF
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	--check)
		mode=check
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		printf 'unknown option: %s\n' "$1" >&2
		usage >&2
		exit 2
		;;
	esac
done

if [ -n "${NATIVE_SKILL_TOOL_DIRS:-}" ]; then
	IFS=':' read -ra tool_dirs <<<"$NATIVE_SKILL_TOOL_DIRS"
else
	tool_dirs=(
		"claude/.claude/skills/Agents/Tools"
		"claude/.claude/skills/Apify"
		"claude/.claude/skills/Art/Tools"
		"claude/.claude/skills/Evals"
		"claude/.claude/skills/Prompting/Templates/Tools"
	)
fi

errors=0
check_installation() {
	local dir="$1"
	local dependency
	if [ ! -d "$dir/node_modules" ]; then
		printf 'missing native skill tool dependencies: %s/node_modules\n' "$dir" >&2
		errors=$((errors + 1))
		return
	fi
	while IFS= read -r dependency; do
		[ -n "$dependency" ] || continue
		if [ ! -e "$dir/node_modules/$dependency/package.json" ]; then
			printf 'missing native skill tool dependency %s in %s\n' "$dependency" "$dir" >&2
			errors=$((errors + 1))
		fi
	done < <(jq -r '[(.dependencies // {}), (.devDependencies // {}), (.optionalDependencies // {})] | add | keys[]' "$dir/package.json")
}

for relative in "${tool_dirs[@]}"; do
	[ -n "$relative" ] || continue
	if [[ "$relative" = /* ]]; then
		dir="$relative"
	else
		dir="$repo_root/$relative"
	fi
	if [ ! -f "$dir/package.json" ]; then
		printf 'missing native skill tool package: %s/package.json\n' "$dir" >&2
		errors=$((errors + 1))
		continue
	fi
	if [ ! -f "$dir/bun.lock" ] && [ ! -f "$dir/bun.lockb" ]; then
		printf 'missing native skill tool lockfile: %s/bun.lock\n' "$dir" >&2
		errors=$((errors + 1))
		continue
	fi
	if [ "$mode" = install ]; then
		printf '==> native skill tools: %s\n' "$relative"
		(
			cd "$dir"
			bun install --frozen-lockfile --ignore-scripts
		)
	fi
	check_installation "$dir"
done

if [ "$errors" -ne 0 ]; then
	exit 1
fi

printf 'PASS native skill tools: %s locked package directories %s\n' \
	"${#tool_dirs[@]}" "$([ "$mode" = install ] && printf installed || printf verified)"
