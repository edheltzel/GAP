#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
checker="$repo_root/check-owned-skills.sh"
temp_root="$(mktemp -d "${TMPDIR:-/tmp}/atlas-ownership-status.XXXXXX")"
temp_root="$(cd "$temp_root" && pwd -P)"
trap 'rm -rf "$temp_root"' EXIT

firstmate="$temp_root/Firstmate"
home="$temp_root/home"
manifest="$temp_root/agent-skills.json"
marker='<!-- installed-by:tldraw-desktop-agent-skills -->'
firstmate_skills=(
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

mkdir -p "$firstmate/.agents/skills" "$firstmate/.claude"
for name in "${firstmate_skills[@]}"; do
	skill="$firstmate/.agents/skills/$name"
	mkdir -p "$skill"
	cat >"$skill/SKILL.md" <<EOF
---
name: $name
metadata:
  internal: true
---
EOF
done
ln -s ../.agents/skills "$firstmate/.claude/skills"

git -C "$firstmate" init -q
git -C "$firstmate" config user.name test
git -C "$firstmate" config user.email test@example.invalid
git -C "$firstmate" add .
git -C "$firstmate" commit -qm fixture

shared="$home/skills/tldraw-offline"
tldraw_files=(
	"$shared/SKILL.md"
	"$shared/README.md"
	"$shared/inject-server-context.sh"
	"$shared/tq"
	"$home/.codex/skills/tldraw-offline/SKILL.md"
	"$home/.claude/skills/tldraw-offline/SKILL.md"
	"$home/.claude/agents/tldraw-offline.md"
	"$home/.gemini/skills/tldraw-offline/SKILL.md"
	"$home/.gemini/agents/tldraw-offline.md"
	"$home/.gemini/config/skills/tldraw-offline/SKILL.md"
	"$home/.pi/agent/skills/tldraw-offline/SKILL.md"
	"$home/.codex/agents/tldraw-offline.toml"
)
for file in "${tldraw_files[@]}"; do
	mkdir -p "$(dirname "$file")"
	printf '%s\nfixture\n' "$marker" >"$file"
done
hook_command="sh '$shared/inject-server-context.sh' SubagentStart"
for settings in "$home/.claude/settings.json" "$home/.codex/hooks.json"; do
	mkdir -p "$(dirname "$settings")"
	cat >"$settings" <<EOF
{
  "hooks": {
    "SubagentStart": [
      {
        "matcher": "tldraw-offline",
        "hooks": [{"type": "command", "command": "$hook_command"}]
      }
    ]
  }
}
EOF
done
python3 - "$manifest" "$home" "$hook_command" <<'PY'
import json
import sys
from pathlib import Path

manifest, home_value, command = sys.argv[1:]
home = Path(home_value)
files = [
    home / "skills/tldraw-offline/SKILL.md",
    home / "skills/tldraw-offline/README.md",
    home / "skills/tldraw-offline/inject-server-context.sh",
    home / "skills/tldraw-offline/tq",
    home / ".codex/skills/tldraw-offline/SKILL.md",
    home / ".claude/skills/tldraw-offline/SKILL.md",
    home / ".claude/agents/tldraw-offline.md",
    home / ".gemini/skills/tldraw-offline/SKILL.md",
    home / ".gemini/agents/tldraw-offline.md",
    home / ".gemini/config/skills/tldraw-offline/SKILL.md",
    home / ".pi/agent/skills/tldraw-offline/SKILL.md",
    home / ".codex/agents/tldraw-offline.toml",
]
hooks = [
    home / ".claude/settings.json",
    home / ".codex/hooks.json",
]
Path(manifest).write_text(
    json.dumps(
        {
            "appVersion": "test",
            "files": [str(path) for path in files],
            "hooks": [
                {"path": str(path), "event": "SubagentStart", "command": command}
                for path in hooks
            ],
        },
        indent=2,
    )
    + "\n"
)
PY
export TLDRAW_AGENT_SKILLS_HOME="$home"
export TLDRAW_AGENT_SKILLS_VERSION=test

FIRSTMATE_ROOT="$firstmate" TLDRAW_AGENT_SKILLS_MANIFEST="$manifest" "$checker" --strict >"$temp_root/pass.out"
grep -F 'PASS Firstmate: 19 internal skills' "$temp_root/pass.out" >/dev/null
grep -F 'PASS tldraw: 12 declared files and 2 hooks' "$temp_root/pass.out" >/dev/null

rm "$shared/tq"
if FIRSTMATE_ROOT="$firstmate" TLDRAW_AGENT_SKILLS_MANIFEST="$manifest" "$checker" --tldraw-only >"$temp_root/missing.out" 2>&1; then
	printf 'missing declared tldraw artifact should fail\n' >&2
	exit 1
fi
grep -F 'missing declared file:' "$temp_root/missing.out" >/dev/null
grep -F '/skills/tldraw-offline/tq' "$temp_root/missing.out" >/dev/null
printf '%s\nfixture\n' "$marker" >"$shared/tq"

python3 - "$home/.claude/settings.json" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
path.write_text(path.read_text().replace('"matcher": "tldraw-offline"', '"matcher": "wrong-skill"'))
PY
if TLDRAW_AGENT_SKILLS_MANIFEST="$manifest" "$checker" --tldraw-only >"$temp_root/hook.out" 2>&1; then
	printf 'incorrect tldraw matcher should fail\n' >&2
	exit 1
fi
grep -F 'tldraw hook installation differs' "$temp_root/hook.out" >/dev/null
python3 - "$home/.claude/settings.json" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
path.write_text(path.read_text().replace('"matcher": "wrong-skill"', '"matcher": "tldraw-offline"'))
PY

python3 - "$manifest" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
path.write_text(path.read_text().replace('"appVersion": "test"', '"appVersion": "drift"'))
PY
if TLDRAW_AGENT_SKILLS_MANIFEST="$manifest" "$checker" --tldraw-only >"$temp_root/version.out" 2>&1; then
	printf 'incorrect tldraw app version should fail\n' >&2
	exit 1
fi
grep -F 'manifest appVersion must be test' "$temp_root/version.out" >/dev/null
python3 - "$manifest" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
path.write_text(path.read_text().replace('"appVersion": "drift"', '"appVersion": "test"'))
PY

mv "$firstmate/.agents/skills/afk" "$firstmate/.agents/skills/unexpected"
if FIRSTMATE_ROOT="$firstmate" "$checker" --firstmate-only >"$temp_root/inventory.out" 2>&1; then
	printf 'incorrect Firstmate skill inventory should fail\n' >&2
	exit 1
fi
grep -F 'canonical internal skill inventory differs' "$temp_root/inventory.out" >/dev/null
mv "$firstmate/.agents/skills/unexpected" "$firstmate/.agents/skills/afk"

python3 - "$firstmate/.agents/skills/afk/SKILL.md" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
path.write_text(path.read_text().replace("internal: true", "internal: false"))
PY
if FIRSTMATE_ROOT="$firstmate" "$checker" --firstmate-only >"$temp_root/metadata.out" 2>&1; then
	printf 'non-internal Firstmate skill should fail\n' >&2
	exit 1
fi
grep -F 'metadata.internal is not true' "$temp_root/metadata.out" >/dev/null

rm -rf "$firstmate"
FIRSTMATE_ROOT="$firstmate" "$checker" --firstmate-only >"$temp_root/skip.out"
grep -F 'SKIP Firstmate: checkout not present' "$temp_root/skip.out" >/dev/null
if FIRSTMATE_ROOT="$firstmate" "$checker" --firstmate-only --strict >"$temp_root/strict.out" 2>&1; then
	printf 'strict ownership validation accepted a missing Firstmate checkout\n' >&2
	exit 1
fi
grep -F 'FAIL Firstmate: checkout not present' "$temp_root/strict.out" >/dev/null

printf 'PASS: external Firstmate and app-owned tldraw ownership checks are bounded and fail closed\n'
