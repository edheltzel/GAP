#!/usr/bin/env bash
# Provision every user-scoped OMP marketplace plugin listed in omp-plugins.txt.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST="$SCRIPT_DIR/omp-plugins.txt"
DRY_RUN=0

usage() {
  cat <<'EOF'
Usage: ./install-omp-plugins.sh [-n|--dry-run] [manifest]

  Ensures each `source=plugin@marketplace` entry is installed user-wide in OMP.
  Existing marketplaces are updated and existing plugins are upgraded.

Options:
  -n, --dry-run   Print the commands without applying changes.
  -h, --help      Show this help.
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    -n|--dry-run) DRY_RUN=1 ;;
    -h|--help) usage; exit 0 ;;
    -*) printf 'Unknown option: %s\n' "$1" >&2; usage >&2; exit 1 ;;
    *) MANIFEST="$1" ;;
  esac
  shift
done

[ -f "$MANIFEST" ] || {
  printf 'Manifest not found: %s\n' "$MANIFEST" >&2
  exit 1
}

sources=()
plugin_ids=()
marketplace_ids=()

valid_source() {
  case "$1" in
    */*/*|/*|*/|*[!A-Za-z0-9._/-]*) return 1 ;;
    */*) return 0 ;;
    *) return 1 ;;
  esac
}

valid_omp_name() {
  [ "${#1}" -le 64 ] || return 1
  case "$1" in
    [a-z0-9]|[a-z0-9]*[a-z0-9]) ;;
    *) return 1 ;;
  esac
  case "$1" in
    *[!a-z0-9.-]*) return 1 ;;
    *) return 0 ;;
  esac
}

while IFS= read -r line || [ -n "$line" ]; do
  line="${line%%#*}"
  line="$(printf '%s' "$line" | tr -d '[:space:]')"
  [ -z "$line" ] && continue

  case "$line" in
    *=*=*)
      printf 'Invalid manifest entry: %s\n' "$line" >&2
      exit 1
      ;;
  esac
  source="${line%%=*}"
  plugin_id="${line#*=}"
  plugin_name="${plugin_id%%@*}"
  marketplace="${plugin_id#*@}"
  if [ "$source" = "$line" ] ||
    [ "$plugin_id" != "$plugin_name@$marketplace" ] ||
    [ "${#plugin_id}" -gt 128 ] ||
    ! valid_source "$source" ||
    ! valid_omp_name "$plugin_name" ||
    ! valid_omp_name "$marketplace"; then
    printf 'Invalid manifest entry: %s\n' "$line" >&2
    exit 1
  fi

  index=0
  while [ "$index" -lt "${#sources[@]}" ]; do
    if [ "${marketplace_ids[$index]}" = "$marketplace" ] &&
      [ "${sources[$index]}" != "$source" ]; then
      printf 'Conflicting sources for marketplace "%s": "%s" and "%s"\n' \
        "$marketplace" "${sources[$index]}" "$source" >&2
      exit 1
    fi
    if [ "${plugin_ids[$index]}" = "$plugin_id" ]; then
      printf 'Duplicate plugin entry: %s\n' "$plugin_id" >&2
      exit 1
    fi
    index=$((index + 1))
  done

  sources[${#sources[@]}]="$source"
  plugin_ids[${#plugin_ids[@]}]="$plugin_id"
  marketplace_ids[${#marketplace_ids[@]}]="$marketplace"
done < "$MANIFEST"

command -v omp >/dev/null 2>&1 || {
  printf 'Need OMP — install it before provisioning marketplace plugins.\n' >&2
  exit 1
}
command -v python3 >/dev/null 2>&1 || {
  printf 'Need python3 to inspect OMP plugin scope.\n' >&2
  exit 1
}

run() {
  printf '   '
  printf ' %q' "$@"
  printf '\n'
  if [ "$DRY_RUN" -eq 0 ]; then
    "$@"
  fi
}

marketplaces="$(omp plugin marketplace list)"
user_plugin_ids="$(
  omp plugin list --json | python3 -c '
import json
import sys

for plugin in json.load(sys.stdin).get("marketplace", []):
    entries = plugin.get("entries") or []
    if plugin.get("scope") == "user" or any(entry.get("scope") == "user" for entry in entries):
        plugin_id = plugin.get("id")
        if plugin_id:
            print(plugin_id)
'
)"

marketplace_source() {
  printf '%s\n' "$marketplaces" |
    awk -v wanted="$1" '$1 == wanted { print $2; exit }'
}

index=0
while [ "$index" -lt "${#sources[@]}" ]; do
  source="${sources[$index]}"
  marketplace="${marketplace_ids[$index]}"
  registered_source="$(marketplace_source "$marketplace")"
  if [ -n "$registered_source" ] && [ "$registered_source" != "$source" ]; then
    printf 'Marketplace "%s" is registered from "%s", not "%s".\n' \
      "$marketplace" "$registered_source" "$source" >&2
    printf 'Remove the existing marketplace and rerun provisioning.\n' >&2
    exit 1
  fi
  index=$((index + 1))
done

processed_marketplaces=""
index=0
while [ "$index" -lt "${#sources[@]}" ]; do
  source="${sources[$index]}"
  plugin_id="${plugin_ids[$index]}"
  marketplace="${marketplace_ids[$index]}"
  printf '==> %s (%s)\n' "$plugin_id" "$source"

  if ! printf '%s\n' "$processed_marketplaces" | grep -Fqx "$marketplace"; then
    if [ -n "$(marketplace_source "$marketplace")" ]; then
      run omp plugin marketplace update "$marketplace"
    else
      run omp plugin marketplace add "$source"
      marketplaces="${marketplaces}"$'\n'"  ${marketplace}  ${source}"
    fi
    processed_marketplaces="${processed_marketplaces}"$'\n'"$marketplace"
  fi

  if printf '%s\n' "$user_plugin_ids" | grep -Fqx "$plugin_id"; then
    run omp plugin upgrade --scope user "$plugin_id"
  else
    run omp plugin install --scope user "$plugin_id"
    user_plugin_ids="${user_plugin_ids}"$'\n'"$plugin_id"
  fi
  index=$((index + 1))
done

printf '\nOK: %d OMP marketplace plugin(s)\n' "${#plugin_ids[@]}"
