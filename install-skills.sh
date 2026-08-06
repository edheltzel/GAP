#!/usr/bin/env bash
#
# install-skills.sh - provision every skills.sh skill listed in skills.txt.
#
# Companion to the skills.txt manifest (dotfiles packages/*.txt model). For each
# manifest line `owner/repo[#commit]=skill,skill`, installs those skills globally
# through the skills.sh CLI. `local:path#commit` selects a pinned sibling checkout.
# An optional `|path|git-blob` suffix verifies one installed SKILL.md, and an
# optional final `|agents=name,name` limits that entry to specific harnesses.
# Full commit refs are copied into a disposable checkout before installation.
# Exits non-zero if any source, target, or integrity check fails.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GIT_COMMON_DIR="$(git -C "$SCRIPT_DIR" rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
if [[ "$GIT_COMMON_DIR" == */.git ]]; then
	SOURCE_BASE="${GIT_COMMON_DIR%/.git}"
else
	SOURCE_BASE="$SCRIPT_DIR"
fi
MANIFEST="$SCRIPT_DIR/skills.txt"
AGENTS="${SKILLS_AGENTS:-}" # override: SKILLS_AGENTS="claude-code,pi,codex"
DRY_RUN=0

trim_whitespace() {
	local value="$1"
	value="${value#"${value%%[![:space:]]*}"}"
	value="${value%"${value##*[![:space:]]}"}"
	printf '%s' "$value"
}

usage() {
	cat <<'EOF'
Usage: ./install-skills.sh [-n|--dry-run] [manifest]

  Installs skills from manifest entries shaped as:
    owner/repo[#commit]=skill,skill[|path|git-blob][|agents=name,name]
    local:path#commit=skill[|path|git-blob][|agents=name,name]
  A full commit ref requires the blob suffix. Local paths are resolved relative to
  the canonical repository root and must be pinned. The default manifest is ./skills.txt.

Options:
  -n, --dry-run   Print the install commands without running them.
  -h, --help      Show this help.

Env:
  SKILLS_AGENTS   Optional comma-separated target filter. Per-entry agents are
                  intersected with this filter; otherwise the CLI detects agents.
EOF
}

while [ $# -gt 0 ]; do
	case "$1" in
	-n | --dry-run) DRY_RUN=1 ;;
	-h | --help)
		usage
		exit 0
		;;
	-*)
		printf 'Unknown option: %s\n' "$1" >&2
		usage >&2
		exit 1
		;;
	*) MANIFEST="$1" ;;
	esac
	shift
done

if [ -t 1 ]; then
	green=$'\e[32m'
	yellow=$'\e[33m'
	red=$'\e[31m'
	cyan=$'\e[36m'
	white=$'\e[37m'
	clr=$'\e[0m'
else
	green=''
	yellow=''
	red=''
	cyan=''
	white=''
	clr=''
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

[ -f "$MANIFEST" ] || {
	printf '%sManifest not found: %s%s\n' "$red" "$MANIFEST" "$clr" >&2
	exit 1
}

ok=0
failed=()
pinned_root=''
prepared_source=''
cleanup() {
	[ -z "$pinned_root" ] || rm -rf "$pinned_root"
}
trap cleanup EXIT

prepare_pinned_source() {
	local source="$1" ref="$2" local_source="${3:-}"
	local source_key archive
	if ! command -v git >/dev/null 2>&1; then
		printf '%s    failed: git is required to install pinned source %s%s\n' "$red" "$source" "$clr" >&2
		return 1
	fi
	if [ -z "$pinned_root" ]; then
		pinned_root="$(mktemp -d "${TMPDIR:-/tmp}/atlas-pinned-skills.XXXXXX")"
	fi
	source_key="$(printf '%s' "$source" | tr '/:.' '----')"
	prepared_source="$pinned_root/${source_key}-${ref:0:12}"
	if [ -f "$prepared_source/.atlas-prepared" ]; then
		return 0
	fi
	rm -rf "$prepared_source"
	mkdir -p "$prepared_source"
	if [ -n "$local_source" ]; then
		if ! git -C "$local_source" cat-file -e "$ref^{commit}" 2>/dev/null; then
			printf '%s    failed: local source %s does not contain commit %s%s\n' \
				"$red" "$local_source" "$ref" "$clr" >&2
			return 1
		fi
		archive="$pinned_root/${source_key}-${ref:0:12}.tar"
		if ! git -C "$local_source" archive --format=tar --output="$archive" "$ref" ||
			! tar -xf "$archive" -C "$prepared_source"; then
			rm -f "$archive"
			printf '%s    failed: could not prepare local source %s at %s%s\n' \
				"$red" "$local_source" "$ref" "$clr" >&2
			return 1
		fi
		rm -f "$archive"
	else
		git -C "$prepared_source" init -q
		if ! git -C "$prepared_source" fetch --depth 1 -q "https://github.com/$source.git" "$ref" ||
			! git -C "$prepared_source" checkout --detach -q FETCH_HEAD; then
			printf '%s    failed: could not fetch %s at %s%s\n' "$red" "$source" "$ref" "$clr" >&2
			return 1
		fi
	fi
	: >"$prepared_source/.atlas-prepared"
}

verify_installed_blob() {
	local name="$1" source_path="$2" expected="$3" required="${4:-0}" targets="${5:-}"
	local installed='' actual target base
	local candidates

	base="$(basename "$source_path")"
	candidates=()
	if [ -n "$targets" ]; then
		IFS=',' read -ra target_names <<<"$targets"
		for target in "${target_names[@]}"; do
			case "$target" in
			claude-code) candidates+=("$HOME/.claude/skills/$name/$base") ;;
			pi) candidates+=("$HOME/.pi/agent/skills/$name/$base") ;;
			codex) candidates+=("$HOME/.codex/skills/$name/$base") ;;
			esac
		done
	fi
	candidates+=("$HOME/.agents/skills/$name/$base")
	for candidate in "${candidates[@]}"; do
		if [ -f "$candidate" ]; then
			installed="$candidate"
			break
		fi
	done
	if [ -z "$installed" ]; then
		printf '%s    not installed: %s%s\n' "$yellow" "${candidates[*]}" "$clr"
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
	line="$(trim_whitespace "$line")"
	[ -z "$line" ] && continue
	source="${line%%=*}"
	descriptor="${line#*=}"
	{ [ "$descriptor" = "$line" ] || [ -z "$descriptor" ]; } && descriptor='*' # bare source → all skills

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

	local_source=''
	if [[ "$source" == local:* ]]; then
		local_token="${source#local:}"
		if [ -z "$local_token" ]; then
			printf '%sInvalid local source in manifest entry: %s%s\n' "$red" "$line" "$clr" >&2
			failed+=("$line")
			continue
		fi
		if [[ "$local_token" = /* ]]; then
			local_candidate="$local_token"
		else
			local_candidate="$SOURCE_BASE/$local_token"
		fi
		if ! local_source="$(cd "$local_candidate" 2>/dev/null && pwd -P)" ||
			! git -C "$local_source" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
			printf '%sLocal skill source is unavailable or not a Git checkout: %s%s\n' \
				"$red" "$local_candidate" "$clr" >&2
			failed+=("$line")
			continue
		fi
		if [ -z "$ref" ]; then
			printf '%sLocal skill sources require an immutable commit: %s%s\n' "$red" "$line" "$clr" >&2
			failed+=("$line")
			continue
		fi
	fi

	integrity_path=''
	expected_blob=''
	entry_agents=''
	IFS='|' read -ra descriptor_parts <<<"$descriptor"
	skills="${descriptor_parts[0]}"
	case "${#descriptor_parts[@]}" in
	1) ;;
	2)
		entry_agents="${descriptor_parts[1]}"
		;;
	3)
		integrity_path="${descriptor_parts[1]}"
		expected_blob="${descriptor_parts[2]}"
		;;
	4)
		integrity_path="${descriptor_parts[1]}"
		expected_blob="${descriptor_parts[2]}"
		entry_agents="${descriptor_parts[3]}"
		;;
	*)
		printf '%sInvalid descriptor in manifest entry: %s%s\n' "$red" "$line" "$clr" >&2
		failed+=("$line")
		continue
		;;
	esac
	if [ -n "$entry_agents" ]; then
		if [[ "$entry_agents" != agents=* ]]; then
			printf '%sInvalid per-entry agent selector: %s%s\n' "$red" "$line" "$clr" >&2
			failed+=("$line")
			continue
		fi
		entry_agents="${entry_agents#agents=}"
		if [[ ! "$entry_agents" =~ ^[A-Za-z0-9._-]+(,[A-Za-z0-9._-]+)*$ ]]; then
			printf '%sInvalid per-entry agent list: %s%s\n' "$red" "$line" "$clr" >&2
			failed+=("$line")
			continue
		fi
	fi
	if [ -n "$integrity_path" ]; then
		if [ -z "$expected_blob" ] || [[ "$skills" == *,* ]]; then
			printf '%sIntegrity descriptors must select one skill: %s%s\n' "$red" "$line" "$clr" >&2
			failed+=("$line")
			continue
		fi
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

	effective_agents=''
	if [ -n "$entry_agents" ]; then
		if [ -n "$AGENTS" ]; then
			IFS=',' read -ra entry_agent_names <<<"$entry_agents"
			IFS=',' read -ra requested_agent_names <<<"$AGENTS"
			matches=()
			for entry_agent in "${entry_agent_names[@]}"; do
				for requested_agent in "${requested_agent_names[@]}"; do
					if [ -n "$entry_agent" ] && [ "$entry_agent" = "$requested_agent" ]; then
						matches+=("$entry_agent")
					fi
				done
			done
			if [ "${#matches[@]}" -eq 0 ]; then
				printf '%s==>%s %s %s(%s)%s [skipped: agents=%s outside SKILLS_AGENTS=%s]\n' \
					"$cyan" "$clr" "$source" "$yellow" "$skills" "$clr" "$entry_agents" "$AGENTS"
				ok=$((ok + 1))
				continue
			fi
			effective_agents="$(
				IFS=,
				printf '%s' "${matches[*]}"
			)"
		else
			effective_agents="$entry_agents"
		fi
	elif [ -n "$AGENTS" ]; then
		effective_agents="$AGENTS"
	fi

	display_source="$source"
	[ -n "$ref" ] && display_source="$source#$ref"
	install_source="${local_source:-$source}"
	if [ -n "$ref" ]; then
		if [ "$DRY_RUN" -eq 1 ]; then
			install_source="<checkout:$source@$ref>"
		elif ! prepare_pinned_source "$source" "$ref" "$local_source"; then
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
	if [ -n "$entry_agents" ]; then
		printf ' %s[agents %s]%s' "$white" "$entry_agents" "$clr"
	fi
	printf '\n'
	# The skills CLI wants one -s per skill; a comma-joined value is read as a single
	# literal name and matches nothing. Expand "a,b,c" into -s a -s b -s c.
	skill_args=()
	IFS=',' read -ra names <<<"$skills"
	for name in "${names[@]}"; do
		[ -n "$name" ] && skill_args+=(-s "$name")
	done
	effective_agent_args=()
	if [ -n "$effective_agents" ]; then
		IFS=',' read -ra effective_agent_names <<<"$effective_agents"
		for agent in "${effective_agent_names[@]}"; do
			[ -n "$agent" ] && effective_agent_args+=(-a "$agent")
		done
	fi
	install_args=(add "$install_source" -g)
	if [ "${#effective_agent_args[@]}" -gt 0 ]; then
		install_args+=("${effective_agent_args[@]}")
	fi
	install_args+=("${skill_args[@]}" -y)
	if [ "$DRY_RUN" -eq 1 ]; then
		if [ -n "$ref" ]; then
			if [ -n "$local_source" ]; then
				printf '    checkout: %s at %s\n' "$local_source" "$ref"
			else
				printf '    checkout: https://github.com/%s.git at %s\n' "$source" "$ref"
			fi
		fi
		printf '    %s' "${runner[*]}"
		printf ' %s' "${install_args[@]}"
		printf '\n'
		[ -n "$ref" ] && printf '    policy ref: %s (enforced by checkout and installed Git blob)\n' "$ref"
		if [ -n "$integrity_path" ]; then
			if verify_installed_blob "$skills" "$integrity_path" "$expected_blob" 0 "$effective_agents"; then
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
		if [ -n "$integrity_path" ] && ! verify_installed_blob "$skills" "$integrity_path" "$expected_blob" 1 "$effective_agents"; then
			failed+=("$line")
		else
			ok=$((ok + 1))
		fi
	else
		printf '%s    failed: %s%s\n' "$red" "$display_source" "$clr" >&2
		failed+=("$display_source")
	fi
done <"$MANIFEST"

printf '\n%sOK: %d source(s)%s\n' "$green" "$ok" "$clr"
if [ "${#failed[@]}" -gt 0 ]; then
	printf '%sFailed (%d): %s%s\n' "$red" "${#failed[@]}" "${failed[*]}" "$clr" >&2
	exit 1
fi
