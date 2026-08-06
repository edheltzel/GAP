#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
skills_root="$repo_root/claude/.claude/skills"
agents_root="$repo_root/claude/.claude/agents"

expected=(
  Agents
  ApertureOscillation
  Aphorisms
  Apify
  ArXiv
  Art
  AudioEditor
  BeCreative
  BiasCheck
  BitterPillEngineering
  BrightData
  Browser
  ContextSearch
  Council
  CreateCLI
  Delegation
  Evals
  ExtractWisdom
  Fabric
  FirstPrinciples
  HTML
  Hardening
  Ideate
  Interceptor
  IterativeDepth
  Loop
  PrivateInvestigator
  Prompting
  RedTeam
  Remotion
  RootCauseAnalysis
  Sales
  Science
  SystemsThinking
  Telos
  USMetrics
  Webdesign
  WorldThreatModel
  WriteStory
  find-docs
  hunk-review
  no-mistakes
  recall-add
  recall-dump
  recall-recent
  recall-scout
  recall-search
  tldraw-offline
  todoist-cli
)

for skill in "${expected[@]}"; do
  path="$skills_root/$skill"
  if [ ! -d "$path" ] || [ -L "$path" ]; then
    printf 'expected native skill directory: %s\n' "$path" >&2
    exit 1
  fi
  case "$skill" in
    recall-add|recall-dump|recall-recent|recall-scout|recall-search)
      if [ ! -L "$path/SKILL.md" ]; then
        printf 'expected recovered Recall skill pointer: %s\n' "$path/SKILL.md" >&2
        exit 1
      fi
      ;;
    *)
      if [ ! -f "$path/SKILL.md" ]; then
        printf 'expected SKILL.md: %s\n' "$path/SKILL.md" >&2
        exit 1
      fi
      ;;
  esac
done

if ! grep -Fq '\.claude/skills/recall-(add|dump|recent|scout|search)' "$repo_root/claude/.stow-local-ignore"; then
  printf 'Recall skill pointers must remain excluded from GNU Stow\n' >&2
  exit 1
fi

actual="$(find "$skills_root" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | LC_ALL=C sort)"
expected_sorted="$(printf '%s\n' "${expected[@]}" | LC_ALL=C sort)"
if [ "$actual" != "$expected_sorted" ]; then
  printf 'native skill inventory differs from the recovered snapshot\n' >&2
  diff -u <(printf '%s\n' "$expected_sorted") <(printf '%s\n' "$actual") >&2 || true
  exit 1
fi

[ -f "$skills_root/CLAUDE.md" ]
[ -f "$agents_root/tldraw-offline.md" ]
if find "$skills_root" -mindepth 1 -maxdepth 1 -type l | grep -q .; then
  printf 'shared skill pointers must be manifest-managed, not tracked by Stow\n' >&2
  exit 1
fi

expected_digest='ea71dddeb54c24a6c4182729889f17abf243c51f26ddd6e20ffb5eadd7f8abea'
actual_digest="$(python3 - "$repo_root" <<'PY'
from pathlib import Path
import hashlib
import os
import subprocess
import sys

root = Path(sys.argv[1])
tracked = {
    raw.decode()
    for raw in subprocess.check_output(
        [
            'git',
            'ls-files',
            '-z',
            '--',
            'claude/.claude/skills',
            'claude/.claude/agents',
        ],
        cwd=root,
    ).split(b'\0')
    if raw
}
skills = root / 'claude/.claude/skills'
paths = []
for child in sorted(
    (entry for entry in skills.iterdir() if entry.is_dir() and not entry.is_symlink()),
    key=lambda entry: entry.name,
):
    prefix = child.relative_to(root).as_posix() + '/'
    paths.extend(root / relative for relative in sorted(tracked) if relative.startswith(prefix))
paths.append(skills / 'CLAUDE.md')
paths.extend(
    root / relative
    for relative in sorted(tracked)
    if relative.startswith('claude/.claude/agents/')
)

digest = hashlib.sha256()
for path in paths:
    relative = path.relative_to(root).as_posix().encode()
    if path.is_symlink():
        mode = b'120000'
        data = os.readlink(path).encode()
    else:
        mode = b'100755' if path.stat().st_mode & 0o111 else b'100644'
        data = path.read_bytes()
    digest.update(mode + b' ' + relative + b'\0' + data + b'\0')

print(digest.hexdigest())
PY
)"
if [ "$actual_digest" != "$expected_digest" ]; then
  printf 'native skill snapshot content, modes, or symlink targets drifted\n' >&2
  printf 'expected: %s\nactual:   %s\n' "$expected_digest" "$actual_digest" >&2
  exit 1
fi

excluded=(
  LifeOS
  LifeOS.backup-20260712-081944
  CreateSkill
  Daemon
  Harvest
  ISA
  Interview
  Knowledge
  LocalIntelligence
  Migrate
  Trim
  Upgrade
  Optimize
  graphify
)
for skill in "${excluded[@]}"; do
  if [ -e "$skills_root/$skill" ] || [ -L "$skills_root/$skill" ]; then
    printf 'deliberately retired skill was restored: %s\n' "$skill" >&2
    exit 1
  fi
done

temp_home="$(mktemp -d "${TMPDIR:-/tmp}/atlas-claude-stow.XXXXXX")"
trap 'rm -rf "$temp_home"' EXIT
if ! HOME="$temp_home" just --justfile "$repo_root/justfile" claude >/dev/null; then
  printf 'Claude package failed to stow into a fresh target\n' >&2
  exit 1
fi
if [ -L "$temp_home/.claude" ]; then
  printf 'Claude package folded the runtime directory on a fresh target\n' >&2
  exit 1
fi
for skill in "${expected[@]}"; do
  case "$skill" in
    recall-add|recall-dump|recall-recent|recall-scout|recall-search)
      if [ -e "$temp_home/.claude/skills/$skill" ] || [ -L "$temp_home/.claude/skills/$skill" ]; then
        printf 'Recall-owned skill pointer was stowed from the recovered snapshot: %s\n' "$skill" >&2
        exit 1
      fi
      ;;
    *)
      if [ ! -f "$temp_home/.claude/skills/$skill/SKILL.md" ]; then
        printf 'native skill was not installed into the temporary HOME: %s\n' "$skill" >&2
        exit 1
      fi
      ;;
  esac
done
[ -f "$temp_home/.claude/agents/tldraw-offline.md" ]
if find "$temp_home/.claude" -type l ! -exec test -e {} \; -print | grep -q .; then
  printf 'temporary Claude installation contains a dangling managed link\n' >&2
  exit 1
fi

printf 'PASS: recovered native skill and agent snapshot matches the curated final lineage\n'
