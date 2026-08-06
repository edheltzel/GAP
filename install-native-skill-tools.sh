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
for required_command in jq bun; do
	if ! command -v "$required_command" >/dev/null 2>&1; then
		printf 'required native skill tool command is unavailable: %s\n' "$required_command" >&2
		errors=$((errors + 1))
	fi
done
if [ "$errors" -ne 0 ]; then
	exit 1
fi

check_installation() {
	local dir="$1"
	local dependency constraint dependency_rows installed_version
	if [ ! -d "$dir/node_modules" ]; then
		printf 'missing native skill tool dependencies: %s/node_modules\n' "$dir" >&2
		errors=$((errors + 1))
		return
	fi
	if ! dependency_rows="$(jq -r '[(.dependencies // {}), (.devDependencies // {}), (.optionalDependencies // {})] | add | to_entries[] | [.key, .value] | @tsv' "$dir/package.json")"; then
		printf 'could not enumerate native skill tool dependencies: %s/package.json\n' "$dir" >&2
		errors=$((errors + 1))
		return
	fi
	while IFS=$'\t' read -r dependency constraint; do
		[ -n "$dependency" ] || continue
		if [ ! -e "$dir/node_modules/$dependency/package.json" ]; then
			printf 'missing native skill tool dependency %s in %s\n' "$dependency" "$dir" >&2
			errors=$((errors + 1))
			continue
		fi
		if ! installed_version="$(jq -er '.version | select(type == "string" and length > 0)' "$dir/node_modules/$dependency/package.json")"; then
			printf 'native skill tool dependency %s has no valid installed version in %s\n' "$dependency" "$dir" >&2
			errors=$((errors + 1))
			continue
		fi
		if ! bun -e 'process.exit(Bun.semver.satisfies(process.argv[1], process.argv[2]) ? 0 : 1)' "$installed_version" "$constraint"; then
			printf 'native skill tool dependency %s@%s does not satisfy %s in %s\n' \
				"$dependency" "$installed_version" "$constraint" "$dir" >&2
			errors=$((errors + 1))
		fi
	done <<<"$dependency_rows"
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
	elif ! (
		cd "$dir"
		bun install --frozen-lockfile --dry-run --ignore-scripts >/dev/null
	); then
		printf 'native skill tool package and lockfile disagree: %s\n' "$dir" >&2
		errors=$((errors + 1))
		continue
	fi
	check_installation "$dir"
done

if [ "$errors" -ne 0 ]; then
	exit 1
fi

printf 'PASS native skill tools: %s locked package directories %s\n' \
	"${#tool_dirs[@]}" "$([ "$mode" = install ] && printf installed || printf verified)"
