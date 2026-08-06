#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
checker="$repo_root/check-owned-skills.sh"
temp_root="$(mktemp -d "${TMPDIR:-/tmp}/atlas-ownership-status.XXXXXX")"
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
claude_skill="$home/.claude/skills/tldraw-offline/SKILL.md"
claude_agent="$home/.claude/agents/tldraw-offline.md"
mkdir -p "$shared" "$(dirname "$claude_skill")" "$(dirname "$claude_agent")"
for file in "$shared/SKILL.md" "$shared/README.md" "$shared/inject-server-context.sh" "$shared/tq" "$claude_skill" "$claude_agent"; do
  printf '%s\nfixture\n' "$marker" > "$file"
done
hook_command="sh '$shared/inject-server-context.sh' SubagentStart"
cat > "$home/.claude/settings.json" <<EOF
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
python3 - "$manifest" "$shared" "$claude_skill" "$claude_agent" "$home/.claude/settings.json" "$hook_command" <<'PY'
import json
import sys
from pathlib import Path

manifest, shared, claude_skill, claude_agent, settings, command = sys.argv[1:]
files = [
    f"{shared}/SKILL.md",
    f"{shared}/README.md",
    claude_skill,
    claude_agent,
    f"{shared}/inject-server-context.sh",
    f"{shared}/tq",
]
Path(manifest).write_text(
    json.dumps(
        {
            "appVersion": "test",
            "files": files,
            "hooks": [
                {"path": settings, "event": "SubagentStart", "command": command}
            ],
        },
        indent=2,
    )
    + "\n"
)
PY

FIRSTMATE_ROOT="$firstmate" TLDRAW_AGENT_SKILLS_MANIFEST="$manifest" "$checker" > "$temp_root/pass.out"
grep -F 'PASS Firstmate: 19 internal skills' "$temp_root/pass.out" >/dev/null
grep -F 'PASS tldraw: 6 declared files and 1 hook' "$temp_root/pass.out" >/dev/null

rm "$shared/tq"
if FIRSTMATE_ROOT="$firstmate" TLDRAW_AGENT_SKILLS_MANIFEST="$manifest" "$checker" --tldraw-only >"$temp_root/missing.out" 2>&1; then
	printf 'missing declared tldraw artifact should fail\n' >&2
	exit 1
fi
grep -F 'missing declared file:' "$temp_root/missing.out" >/dev/null
grep -F '/skills/tldraw-offline/tq' "$temp_root/missing.out" >/dev/null
printf '%s\nfixture\n' "$marker" >"$shared/tq"

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

printf 'PASS: external Firstmate and app-owned tldraw ownership checks are bounded and fail closed\n'
