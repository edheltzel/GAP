#!/usr/bin/env bash
# Link only the globally installed harness modules needed for local type checking.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GLOBAL_MODULES="${BUN_GLOBAL_MODULES:-$HOME/.bun/install/global/node_modules}"
NODE_MODULES="$SCRIPT_DIR/node_modules"
CHECK_ONLY=0

usage() {
  printf 'Usage: %s [--check]\n' "${0##*/}"
}

case "${1:-}" in
'') ;;
--check) CHECK_ONLY=1 ;;
-h | --help)
  usage
  exit 0
  ;;
*)
  usage >&2
  exit 1
  ;;
esac

if [ -L "$NODE_MODULES" ] || { [ -e "$NODE_MODULES" ] && [ ! -d "$NODE_MODULES" ]; }; then
  printf 'refusing unexpected node_modules entry: %s\n' "$NODE_MODULES" >&2
  exit 1
fi

links=(
  '@earendil-works'
  '@oh-my-pi'
  '@types/node'
  'typescript'
)
failed=0

for relative in "${links[@]}"; do
  source_path="$GLOBAL_MODULES/$relative"
  destination="$NODE_MODULES/$relative"

  if [ ! -e "$source_path" ]; then
    printf 'required global module is unavailable: %s\n' "$source_path" >&2
    failed=1
    continue
  fi

  if [ -L "$destination" ]; then
    if [ "$(readlink "$destination")" != "$source_path" ]; then
      printf 'unexpected development-module link: %s -> %s\n' \
        "$destination" "$(readlink "$destination")" >&2
      failed=1
    fi
    continue
  fi

  if [ -e "$destination" ]; then
    printf 'refusing existing development-module path: %s\n' "$destination" >&2
    failed=1
    continue
  fi

  if [ "$CHECK_ONLY" -eq 1 ]; then
    printf 'missing development-module link: %s\n' "$destination" >&2
    failed=1
    continue
  fi

  mkdir -p "$(dirname "$destination")"
  ln -s "$source_path" "$destination"
done

if [ "$failed" -ne 0 ]; then
  exit 1
fi

if [ "$CHECK_ONLY" -eq 1 ]; then
  printf 'Development module links are valid\n'
else
  printf 'Development module links are ready\n'
fi
