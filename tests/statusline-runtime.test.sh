#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
global_modules="$HOME/.bun/install/global/node_modules"
node_modules="$repo_root/node_modules"
created_link=0
cleanup() {
  if [ "$created_link" -eq 1 ]; then rm "$node_modules"; fi
}
trap cleanup EXIT

for package in @earendil-works/pi-coding-agent @oh-my-pi/pi-coding-agent @oh-my-pi/pi-tui; do
  if [ ! -d "$global_modules/$package" ]; then
    printf 'required installed harness module is unavailable: %s\n' "$package" >&2
    exit 1
  fi
done

if [ ! -e "$node_modules" ] && [ ! -L "$node_modules" ]; then
  ln -s "$global_modules" "$node_modules"
  created_link=1
fi

bun test \
  "$repo_root/tests/pi/atlas-status-line.test.ts" \
  "$repo_root/tests/omp/atlas-status-line.test.ts"

theme_module="$global_modules/@earendil-works/pi-coding-agent/dist/modes/interactive/theme/theme.js"
theme_path="$repo_root/pi/.pi/agent/themes/eldritch.json"
PI_THEME_MODULE="$theme_module" PI_THEME_PATH="$theme_path" bun -e '
  const module = await import(process.env.PI_THEME_MODULE);
  const theme = module.loadThemeFromPath(process.env.PI_THEME_PATH);
  if (theme.name !== "eldritch") {
    throw new Error(`unexpected loaded theme: ${theme.name}`);
  }
'

printf 'PASS: Pi and OMP status lines execute and the installed Pi loader accepts Eldritch\n'
