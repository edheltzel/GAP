#!/usr/bin/env bash
# Usage: check-owned-skills.sh [--firstmate PATH] [--tldraw-manifest PATH] [--firstmate-only|--tldraw-only]
#
# Opt-in ownership status for external Firstmate internal skills and tldraw Desktop's
# declared agent-skill installation. This script only reads external/runtime state.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
git_common_dir="$(git -C "$repo_root" rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
if [[ "$git_common_dir" == */.git ]]; then
	canonical_root="${git_common_dir%/.git}"
else
	canonical_root="$repo_root"
fi
firstmate_root="${FIRSTMATE_ROOT:-$(dirname "$canonical_root")/Firstmate}"
if [ "$(uname -s)" = "Darwin" ]; then
	default_tldraw_manifest="$HOME/Library/Application Support/tldraw/agent-skills.json"
else
	default_tldraw_manifest="${XDG_CONFIG_HOME:-$HOME/.config}/tldraw/agent-skills.json"
fi
tldraw_manifest="${TLDRAW_AGENT_SKILLS_MANIFEST:-$default_tldraw_manifest}"
tldraw_home="${TLDRAW_AGENT_SKILLS_HOME:-$HOME}"
tldraw_version="${TLDRAW_AGENT_SKILLS_VERSION:-1.12.1}"
check_firstmate=1
check_tldraw=1
strict=0
errors=0
firstmate_expected=(
	afk
	ahoy
	ask-user-authority
	bearings
	bootstrap-diagnostics
	decision-hold-lifecycle
	diagnostic-reasoning
	firstmate-codexapp
	firstmate-coding-guidelines
	firstmate-orca
	fmx-respond
	harness-adapters
	process-event-sources
	project-management
	quota-array-dispatch
	secondmate-provisioning
	stow
	stuck-crewmate-recovery
	updatefirstmate
)

usage() {
	cat <<'EOF'
Usage: check-owned-skills.sh [options]

Options:
  --firstmate PATH         Validate this Firstmate checkout.
  --tldraw-manifest PATH   Validate this tldraw agent-skills.json.
  --firstmate-only         Skip tldraw validation.
  --tldraw-only            Skip Firstmate validation.
  --strict                  Fail when an external ownership store is absent.
  -h, --help               Show this help.

FIRSTMATE_ROOT, TLDRAW_AGENT_SKILLS_MANIFEST, TLDRAW_AGENT_SKILLS_HOME, and
TLDRAW_AGENT_SKILLS_VERSION provide environment overrides. Missing default
checkouts/manifests are reported as SKIP unless --strict is set.
EOF
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	--firstmate)
		[ "$#" -ge 2 ] || {
			printf 'missing value for --firstmate\n' >&2
			exit 2
		}
		firstmate_root="$2"
		shift 2
		;;
	--tldraw-manifest)
		[ "$#" -ge 2 ] || {
			printf 'missing value for --tldraw-manifest\n' >&2
			exit 2
		}
		tldraw_manifest="$2"
		shift 2
		;;
	--firstmate-only)
		check_tldraw=0
		shift
		;;
	--tldraw-only)
		check_firstmate=0
		shift
		;;
	--strict)
		strict=1
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

fail() {
	printf 'FAIL %s\n' "$1" >&2
	errors=$((errors + 1))
}

has_internal_metadata() {
	awk '
    NR == 1 && $0 == "---" { frontmatter = 1; next }
    frontmatter && $0 == "---" { exit found ? 0 : 1 }
    frontmatter && /^metadata:[[:space:]]*$/ { metadata = 1; next }
    metadata && /^[^[:space:]]/ { metadata = 0 }
    metadata && /^[[:space:]]+internal:[[:space:]]*true([[:space:]]*(#.*)?)?$/ { found = 1 }
    END { if (!frontmatter || !found) exit 1 }
  ' "$1"
}

check_firstmate_checkout() {
	if [ ! -e "$firstmate_root" ]; then
		if [ "$strict" -eq 1 ]; then
			fail "Firstmate: checkout not present at $firstmate_root"
		else
			printf 'SKIP Firstmate: checkout not present at %s\n' "$firstmate_root"
		fi
		return
	fi

	local before="$errors"
	local skills_root="$firstmate_root/.agents/skills"
	local claude_link="$firstmate_root/.claude/skills"
	local actual actual_count entry_count expected head

	if ! git -C "$firstmate_root" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
		fail "Firstmate: not a Git checkout: $firstmate_root"
	fi
	if [ ! -d "$skills_root" ] || [ -L "$skills_root" ]; then
		fail "Firstmate: expected real internal skill directory: $skills_root"
		return
	fi

	actual="$(find "$skills_root" -mindepth 1 -maxdepth 1 -type d ! -type l -exec basename {} \; | LC_ALL=C sort)"
	expected="$(printf '%s\n' "${firstmate_expected[@]}" | LC_ALL=C sort)"
	actual_count="$(printf '%s\n' "$actual" | awk 'NF { count++ } END { print count + 0 }')"
	entry_count="$(find "$skills_root" -mindepth 1 -maxdepth 1 -print | wc -l | tr -d '[:space:]')"
	if [ "$actual" != "$expected" ] || [ "$entry_count" != "${#firstmate_expected[@]}" ]; then
		fail "Firstmate: canonical internal skill inventory differs (found $actual_count directories and $entry_count entries)"
		diff -u <(printf '%s\n' "$expected") <(printf '%s\n' "$actual") >&2 || true
	fi

	while IFS= read -r skill_dir; do
		if [ -L "$skill_dir" ]; then
			fail "Firstmate: internal skill must not be a symlink: $skill_dir"
			continue
		fi
		if [ ! -f "$skill_dir/SKILL.md" ]; then
			fail "Firstmate: missing SKILL.md: $skill_dir/SKILL.md"
			continue
		fi
		if ! has_internal_metadata "$skill_dir/SKILL.md"; then
			fail "Firstmate: metadata.internal is not true: $skill_dir/SKILL.md"
		fi
	done < <(find "$skills_root" -mindepth 1 -maxdepth 1 -type d -print | LC_ALL=C sort)

	if [ ! -L "$claude_link" ]; then
		fail "Firstmate: Claude compatibility path is not a symlink: $claude_link"
	elif [ "$(readlink "$claude_link")" != "../.agents/skills" ]; then
		fail "Firstmate: Claude compatibility symlink must target ../.agents/skills: $claude_link"
	elif [ ! -d "$claude_link" ]; then
		fail "Firstmate: Claude compatibility symlink is dangling: $claude_link"
	fi

	while IFS= read -r dangling; do
		fail "Firstmate: dangling link: $dangling"
	done < <(find "$skills_root" "$firstmate_root/.claude" -type l ! -exec test -e {} \; -print)

	if [ "$errors" -eq "$before" ]; then
		head="$(git -C "$firstmate_root" rev-parse HEAD 2>/dev/null || printf unknown)"
		printf 'PASS Firstmate: 19 internal skills, Claude compatibility link, no dangling links (HEAD %s)\n' "$head"
	fi
}

check_tldraw_install() {
	if [ ! -e "$tldraw_manifest" ]; then
		if [ "$strict" -eq 1 ]; then
			fail "tldraw: app-owned manifest not present at $tldraw_manifest"
		else
			printf 'SKIP tldraw: app-owned manifest not present at %s\n' "$tldraw_manifest"
		fi
		return
	fi
	if ! command -v python3 >/dev/null 2>&1; then
		fail "tldraw: python3 is required to validate the app-owned JSON manifest"
		return
	fi

	if ! python3 - "$tldraw_manifest" "$repo_root" "$tldraw_home" "$tldraw_version" <<'PY'; then
import json
import os
import sys
from pathlib import Path

manifest_path = Path(sys.argv[1])
config_root = Path(sys.argv[2]).resolve()
home = Path(sys.argv[3]).expanduser().resolve()
expected_version = sys.argv[4]
marker = "installed-by:tldraw-desktop-agent-skills"
issues = []

try:
    manifest = json.loads(manifest_path.read_text())
except Exception as error:
    print(f"FAIL tldraw: invalid manifest {manifest_path}: {error}", file=sys.stderr)
    raise SystemExit(1)

if not isinstance(manifest, dict):
    print(f"FAIL tldraw: manifest root is not an object: {manifest_path}", file=sys.stderr)
    raise SystemExit(1)

app_version = manifest.get("appVersion")
files = manifest.get("files")
hooks = manifest.get("hooks")
expected_command = f"sh '{home}/skills/tldraw-offline/inject-server-context.sh' SubagentStart"
expected_files = {
    str(home / "skills/tldraw-offline/SKILL.md"),
    str(home / "skills/tldraw-offline/README.md"),
    str(home / "skills/tldraw-offline/inject-server-context.sh"),
    str(home / "skills/tldraw-offline/tq"),
    str(home / ".codex/skills/tldraw-offline/SKILL.md"),
    str(home / ".claude/skills/tldraw-offline/SKILL.md"),
    str(home / ".claude/agents/tldraw-offline.md"),
    str(home / ".gemini/skills/tldraw-offline/SKILL.md"),
    str(home / ".gemini/agents/tldraw-offline.md"),
    str(home / ".gemini/config/skills/tldraw-offline/SKILL.md"),
    str(home / ".pi/agent/skills/tldraw-offline/SKILL.md"),
    str(home / ".codex/agents/tldraw-offline.toml"),
}
expected_hooks = {
    (str(home / ".claude/settings.json"), "SubagentStart", expected_command),
    (str(home / ".codex/hooks.json"), "SubagentStart", expected_command),
}

if app_version != expected_version:
    issues.append(f"manifest appVersion must be {expected_version}, got {app_version!r}")
if not isinstance(files, list) or any(not isinstance(path, str) or not path for path in files):
    issues.append("manifest files must be a string array")
    files = []
if not isinstance(hooks, list) or any(not isinstance(hook, dict) for hook in hooks):
    issues.append("manifest hooks must be an object array")
    hooks = []

if len(files) != len(set(files)):
    issues.append("manifest contains duplicate file paths")
actual_files = set(files)
if actual_files != expected_files:
    for path in sorted(expected_files - actual_files):
        issues.append(f"manifest omits expected app artifact: {path}")
    for path in sorted(actual_files - expected_files):
        issues.append(f"manifest includes unexpected app artifact: {path}")

for declared in files:
    path = Path(declared).expanduser()
    if not os.path.lexists(path):
        issues.append(f"missing declared file: {path}")
        continue
    if path.is_symlink() and not path.exists():
        issues.append(f"dangling declared link: {path}")
        continue
    if not path.is_file():
        issues.append(f"declared artifact is not a file: {path}")
        continue
    try:
        resolved = path.resolve()
        resolved.relative_to(config_root)
    except ValueError:
        pass
    except OSError as error:
        issues.append(f"cannot resolve declared file {path}: {error}")
    else:
        issues.append(f"declared app artifact still resolves into Atlas Config: {path} -> {resolved}")
    try:
        content = path.read_text(errors="replace")
    except OSError as error:
        issues.append(f"cannot read declared file {path}: {error}")
        continue
    if marker not in content:
        issues.append(f"declared file lacks app ownership marker: {path}")

declared_hooks = []
for hook in hooks:
    path_value = hook.get("path")
    event = hook.get("event")
    command = hook.get("command")
    if not all(isinstance(value, str) and value for value in (path_value, event, command)):
        issues.append("manifest hook requires non-empty path, event, and command")
        continue
    declared_hooks.append((path_value, event, command))
if len(declared_hooks) != len(expected_hooks) or set(declared_hooks) != expected_hooks:
    issues.append("manifest hook inventory differs from the exact app-owned hook set")

for path_value, event, command in sorted(expected_hooks):
    path = Path(path_value)
    if not path.is_file():
        issues.append(f"missing hook config: {path}")
        continue
    try:
        config = json.loads(path.read_text())
    except Exception as error:
        issues.append(f"invalid hook config {path}: {error}")
        continue
    installed_entries = []
    config_hooks = config.get("hooks", {}) if isinstance(config, dict) else {}
    if isinstance(config_hooks, dict):
        for configured_event, entries in config_hooks.items():
            if not isinstance(entries, list):
                continue
            for entry in entries:
                if not isinstance(entry, dict):
                    continue
                matcher = entry.get("matcher")
                items = entry.get("hooks", [])
                if not isinstance(items, list):
                    continue
                for item in items:
                    if not isinstance(item, dict):
                        continue
                    installed_command = item.get("command")
                    if isinstance(installed_command, str) and "tldraw-offline/inject-server-context.sh" in installed_command:
                        installed_entries.append(
                            (configured_event, matcher, item.get("type"), installed_command)
                        )
    expected_entry = (event, "tldraw-offline", "command", command)
    if installed_entries != [expected_entry]:
        issues.append(
            f"tldraw hook installation differs in {path}: expected {expected_entry!r}, got {installed_entries!r}"
        )

if issues:
    for issue in issues:
        print(f"FAIL tldraw: {issue}", file=sys.stderr)
    raise SystemExit(1)

print(
    f"PASS tldraw: {len(files)} declared files and {len(hooks)} hook"
    f"{'s' if len(hooks) != 1 else ''} verified (appVersion {app_version})"
)
PY
		errors=$((errors + 1))
	fi
}

if [ "$check_firstmate" -eq 1 ]; then check_firstmate_checkout; fi
if [ "$check_tldraw" -eq 1 ]; then check_tldraw_install; fi

if [ "$errors" -ne 0 ]; then
	exit 1
fi
