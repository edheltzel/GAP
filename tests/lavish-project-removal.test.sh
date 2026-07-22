#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temp_home="$(mktemp -d "${TMPDIR:-/tmp}/atlas-lavish-removal-test.XXXXXX")"
trap 'rm -rf "$temp_home"' EXIT

commit="55045850f61e7b23c8dbf8cc30d92bd6e31649d2"
blob="1e792b5bfc34127ca6f00bbb44fea306ce77ea52"
home="$temp_home/home"
fixture="$temp_home/lavish/SKILL.md"
manifest="$temp_home/skills.txt"
bin_dir="$temp_home/bin"
mkdir -p "$home" "$bin_dir" "$(dirname "$fixture")"

assert_absent() {
  local path="$1"
  if [ -e "$path" ] || [ -L "$path" ]; then
    printf 'expected path to be absent: %s\n' "$path" >&2
    exit 1
  fi
}

assert_absent "$repo_root/.agents/skills/lavish/SKILL.md"
assert_absent "$repo_root/.claude/skills/lavish"
if jq -e '.skills.lavish' "$repo_root/skills-lock.json" >/dev/null; then
  printf 'expected project Lavish lock entry to be absent\n' >&2
  exit 1
fi
grep -Fx "kunchenguid/lavish-axi#$commit=lavish|skills/lavish/SKILL.md|$blob" "$repo_root/skills.txt" >/dev/null
grep -F 'Fresh or disposable checkouts' "$repo_root/README.md" >/dev/null
grep -F 'roll back the project-ownership removal' "$repo_root/README.md" >/dev/null

source_commit="$(git -C "$repo_root" log -1 --format=%H --all --diff-filter=A -- .agents/skills/lavish/SKILL.md)"
[ -n "$source_commit" ]
git -C "$repo_root" show "$source_commit:.agents/skills/lavish/SKILL.md" > "$fixture"
test "$(git -C "$repo_root" hash-object "$fixture")" = "$blob"
printf 'kunchenguid/lavish-axi#%s=lavish|skills/lavish/SKILL.md|%s\n' "$commit" "$blob" > "$manifest"

cat > "$bin_dir/skills" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

[ "${1:-}" = add ]
mkdir -p "$HOME/.agents/skills/lavish" "$HOME/.claude/skills" "$HOME/.pi/agent/skills"
cp "$LAVISH_FIXTURE" "$HOME/.agents/skills/lavish/SKILL.md"
ln -sfn ../../.agents/skills/lavish "$HOME/.claude/skills/lavish"
ln -sfn ../../../.agents/skills/lavish "$HOME/.pi/agent/skills/lavish"
EOF
chmod +x "$bin_dir/skills"

mkdir -p "$home/.agents/skills/unrelated"
printf 'unrelated skill\n' > "$home/.agents/skills/unrelated/SKILL.md"
unrelated_hash="$(shasum -a 256 "$home/.agents/skills/unrelated/SKILL.md")"
package_inventory="$(git -C "$repo_root" ls-files --stage -- pi/.pi/agent)"
repo_status="$(git -C "$repo_root" status --porcelain)"

output="$(HOME="$home" PATH="$bin_dir:$PATH" LAVISH_FIXTURE="$fixture" SKILLS_AGENTS='claude-code,pi,codex' "$repo_root/install-skills.sh" "$manifest" 2>&1)"
grep -F "integrity verified" <<<"$output" >/dev/null
test "$(git -C "$repo_root" hash-object --path=skills/lavish/SKILL.md "$home/.agents/skills/lavish/SKILL.md")" = "$blob"
test "$(readlink "$home/.claude/skills/lavish")" = '../../.agents/skills/lavish'
test "$(readlink "$home/.pi/agent/skills/lavish")" = '../../../.agents/skills/lavish'

snapshot="$temp_home/rollback-snapshot"
cp -a "$home/.agents/skills/lavish" "$snapshot"
printf 'drift\n' > "$home/.agents/skills/lavish/SKILL.md"
rm "$home/.claude/skills/lavish" "$home/.pi/agent/skills/lavish"
rm -rf "$home/.agents/skills/lavish"
cp -a "$snapshot" "$home/.agents/skills/lavish"
ln -s ../../.agents/skills/lavish "$home/.claude/skills/lavish"
ln -s ../../../.agents/skills/lavish "$home/.pi/agent/skills/lavish"
cmp -s "$fixture" "$home/.agents/skills/lavish/SKILL.md"

output="$(HOME="$home" PATH="$bin_dir:$PATH" LAVISH_FIXTURE="$fixture" SKILLS_AGENTS='claude-code,pi,codex' "$repo_root/install-skills.sh" "$manifest" 2>&1)"
grep -F "integrity verified" <<<"$output" >/dev/null
test "$(shasum -a 256 "$home/.agents/skills/unrelated/SKILL.md")" = "$unrelated_hash"
test "$(git -C "$repo_root" ls-files --stage -- pi/.pi/agent)" = "$package_inventory"
test "$(git -C "$repo_root" status --porcelain)" = "$repo_status"

pi_bin="$(command -v pi || true)"
[ -n "$pi_bin" ] || { printf 'pi is required for the Lavish resolution test\n' >&2; exit 1; }
pi_dist="$(dirname "$(realpath "$pi_bin")")"
HOME="$home" PI_DIST="$pi_dist" TEST_REPO="$repo_root" TEST_HOME="$home" node --input-type=module <<'NODE'
import assert from "node:assert/strict";
import { realpathSync } from "node:fs";
import { basename, dirname, join } from "node:path";
import { pathToFileURL } from "node:url";

const dist = process.env.PI_DIST;
const repo = process.env.TEST_REPO;
const home = process.env.TEST_HOME;
const { SettingsManager } = await import(pathToFileURL(join(dist, "core/settings-manager.js")).href);
const { DefaultPackageManager } = await import(pathToFileURL(join(dist, "core/package-manager.js")).href);
const { loadSkills } = await import(pathToFileURL(join(dist, "core/skills.js")).href);
const globalSkill = join(home, ".agents/skills/lavish/SKILL.md");

for (const projectTrusted of [true, false]) {
  const settings = SettingsManager.inMemory({}, { projectTrusted });
  const manager = new DefaultPackageManager({
    cwd: repo,
    agentDir: join(home, ".pi/agent"),
    settingsManager: settings,
  });
  const resolved = await manager.resolve();
  const lavish = resolved.skills.filter(({ path }) => basename(dirname(path)) === "lavish");
  assert.equal(lavish.length, 1, `expected one Lavish path for trusted=${projectTrusted}`);
  assert.equal(lavish[0].metadata.scope, "user");
  assert.equal(realpathSync(lavish[0].path), realpathSync(globalSkill));

  const loaded = loadSkills({
    cwd: repo,
    agentDir: join(home, ".pi/agent"),
    skillPaths: lavish.map(({ path }) => path),
    includeDefaults: false,
  });
  assert.deepEqual(loaded.skills.map(({ name }) => name), ["lavish"]);
  assert.equal(loaded.diagnostics.filter(({ type }) => type === "collision").length, 0);
}
NODE

printf 'PASS: project ownership is absent; pinned global provisioning, rollback, and trusted/untrusted resolution are collision-free\n'
