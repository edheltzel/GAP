#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temp_home="$(mktemp -d "${TMPDIR:-/tmp}/atlas-stow-safety.XXXXXX")"
just_cmd=(just --justfile "$repo_root/justfile")
filter_keys=(
	filter.jcode-config.clean
	filter.jcode-config.smudge
	filter.jcode-config.required
	merge.ours.driver
)
original_filter_values=()
original_filter_presence=()

for key in "${filter_keys[@]}"; do
	if value="$(git -C "$repo_root" config --local --get "$key")"; then
		original_filter_presence+=(1)
		original_filter_values+=("$value")
	else
		original_filter_presence+=(0)
		original_filter_values+=("")
	fi
done

cleanup() {
	local index key
	rm -rf "$temp_home"
	for index in "${!filter_keys[@]}"; do
		key="${filter_keys[$index]}"
		if [ "${original_filter_presence[$index]}" -eq 1 ]; then
			git -C "$repo_root" config --local "$key" "${original_filter_values[$index]}"
		else
			git -C "$repo_root" config --local --unset-all "$key" 2>/dev/null || :
		fi
	done
}
trap cleanup EXIT

"${just_cmd[@]}" jcode-filter
if [ "$(git -C "$repo_root" config --get merge.ours.driver)" != 'true' ]; then
	printf 'repository-local merge.ours.driver was not configured\n' >&2
	exit 1
fi

HOME="$temp_home" "${just_cmd[@]}" stow-preflight >/dev/null
HOME="$temp_home" "${just_cmd[@]}" status >/dev/null
space_home="$temp_home/home with spaces"
mkdir -p "$space_home"
HOME="$space_home" "${just_cmd[@]}" stow-preflight >/dev/null
if grep -Fq '"local-packages/pi-extensions"' "$repo_root/pi/.pi/agent/settings.json"; then
  printf 'tracked Pi settings still register the retired local package path\n' >&2
  exit 1
fi

mkdir -p "$temp_home/.jcode" "$temp_home/external"
printf '{"terminal":"external"}\n' >"$temp_home/external/preferred_terminal.json"
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

mkdir -p "$temp_home/.claude/skills/Apify"
ln -s missing "$temp_home/.claude/skills/Apify/node_modules"
HOME="$temp_home" "${just_cmd[@]}" prune-native-dependency-links >/dev/null
if [ -L "$temp_home/.claude/skills/Apify/node_modules" ]; then
	printf 'native dependency pruning left a dangling ignored node_modules link\n' >&2
	exit 1
fi
mkdir -p "$temp_home/external/valid-node-modules"
ln -s "$temp_home/external/valid-node-modules" "$temp_home/.claude/skills/Apify/node_modules"
HOME="$temp_home" "${just_cmd[@]}" prune-native-dependency-links >/dev/null
if [ ! -e "$temp_home/.claude/skills/Apify/node_modules" ]; then
	printf 'native dependency pruning removed a valid node_modules link\n' >&2
	exit 1
fi
rm "$temp_home/.claude/skills/Apify/node_modules"

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
if find "$temp_home/.claude" \( -type d -o -type l \) -name node_modules -print -quit | grep -q .; then
	printf 'Claude package stowed repository-local native tool dependencies\n' >&2
	exit 1
fi

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
	update | link)
		prune_dependencies_line="$(grep -n -m1 './prune-stale-native-dependency-links.sh' <<<"$dry_run" | cut -d: -f1)"
		preflight_line="$(grep -n -m1 'Stow preflight passed' <<<"$dry_run" | cut -d: -f1)"
		native_tools_line="$(grep -n -m1 './install-native-skill-tools.sh' <<<"$dry_run" | cut -d: -f1)"
		manifest_line="$(grep -n -m1 './install-skills.sh' <<<"$dry_run" | cut -d: -f1)"
		if [ -z "$prune_dependencies_line" ] || [ -z "$preflight_line" ] || [ -z "$native_tools_line" ] || [ -z "$manifest_line" ] ||
			[ "$prune_dependencies_line" -ge "$preflight_line" ] || [ "$preflight_line" -ge "$native_tools_line" ] || [ "$native_tools_line" -ge "$manifest_line" ]; then
			printf 'aggregate recipe %s does not preflight before native tool and manifest provisioning\n' "$recipe" >&2
			exit 1
		fi
		;;
	esac
done

cleanup
for index in "${!filter_keys[@]}"; do
  key="${filter_keys[$index]}"
  if [ "${original_filter_presence[$index]}" -eq 1 ]; then
    current_value="$(git -C "$repo_root" config --local --get "$key")"
    if [ "$current_value" != "${original_filter_values[$index]}" ]; then
      printf 'Stow safety test did not restore Git config key %s\n' "$key" >&2
      exit 1
    fi
  elif git -C "$repo_root" config --local --get "$key" >/dev/null 2>&1; then
    printf 'Stow safety test introduced Git config key %s\n' "$key" >&2
    exit 1
  fi
done
trap - EXIT

printf 'PASS: Stow preflight, pruning, status, no-folding, and aggregate coverage are safe\n'
