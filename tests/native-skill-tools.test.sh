#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

first="$tmp/first"
second="$tmp/second"
mkdir -p "$first" "$second" "$tmp/bin"
printf '%s\n' '{"dependencies":{"alpha":"1.0.0"},"devDependencies":{"gamma":"1.0.0"}}' >"$first/package.json"
printf '%s\n' '{"dependencies":{"@scope/beta":"1.0.0"}}' >"$second/package.json"
printf '%s\n' 'lockfileVersion = 1' >"$first/bun.lock"
printf '%s\n' 'lockfileVersion = 1' >"$second/bun.lock"

cat >"$tmp/bin/bun" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
install)
	if [[ " $* " == *" --dry-run "* ]]; then
		[ "${FAKE_BUN_FROZEN_FAIL:-0}" -eq 0 ]
		exit
	fi
	mkdir -p node_modules
	while IFS=$'\t' read -r dependency constraint; do
		mkdir -p "node_modules/$dependency"
		printf '{"version":"%s"}\n' "$constraint" >"node_modules/$dependency/package.json"
	done < <(jq -r '[(.dependencies // {}), (.devDependencies // {}), (.optionalDependencies // {})] | add | to_entries[] | [.key, .value] | @tsv' package.json)
	;;
-e)
	[ "${3:-}" = "${4:-}" ]
	;;
*)
	exit 2
	;;
esac
EOF
chmod +x "$tmp/bin/bun"

run_tools() {
	PATH="$tmp/bin:$PATH" NATIVE_SKILL_TOOL_DIRS="$first:$second" \
		"$repo_root/install-native-skill-tools.sh" "$@"
}

if run_tools --check >"$tmp/precheck.out" 2>&1; then
	printf 'native skill tool check accepted missing dependencies\n' >&2
	exit 1
fi
grep -q 'missing native skill tool dependencies' "$tmp/precheck.out"

run_tools >/dev/null
run_tools --check >/dev/null

printf '%s\n' '{"version":"9.9.9"}' >"$first/node_modules/alpha/package.json"
if run_tools --check >"$tmp/version.out" 2>&1; then
	printf 'native skill tool check accepted an incompatible installed version\n' >&2
	exit 1
fi
grep -q 'alpha@9.9.9 does not satisfy 1.0.0' "$tmp/version.out"
printf '%s\n' '{"version":"1.0.0"}' >"$first/node_modules/alpha/package.json"

if FAKE_BUN_FROZEN_FAIL=1 run_tools --check >"$tmp/frozen.out" 2>&1; then
	printf 'native skill tool check accepted package and lockfile drift\n' >&2
	exit 1
fi
grep -q 'package and lockfile disagree' "$tmp/frozen.out"

mkdir -p "$tmp/no-jq-bin"
cat >"$tmp/no-jq-bin/jq" <<'EOF'
#!/usr/bin/env bash
exit 127
EOF
chmod +x "$tmp/no-jq-bin/jq"
if PATH="$tmp/no-jq-bin:$tmp/bin:$PATH" NATIVE_SKILL_TOOL_DIRS="$first:$second" \
	"$repo_root/install-native-skill-tools.sh" --check >"$tmp/jq.out" 2>&1; then
	printf 'native skill tool check succeeded when jq failed\n' >&2
	exit 1
fi
grep -q 'could not enumerate native skill tool dependencies' "$tmp/jq.out"

rm "$first/node_modules/alpha/package.json"
if run_tools --check >"$tmp/dependency.out" 2>&1; then
	printf 'native skill tool check accepted a missing direct dependency\n' >&2
	exit 1
fi
grep -q 'missing native skill tool dependency alpha' "$tmp/dependency.out"

printf '%s\n' '{"version":"1.0.0"}' >"$first/node_modules/alpha/package.json"
rm "$first/node_modules/gamma/package.json"
if run_tools --check >"$tmp/development-dependency.out" 2>&1; then
	printf 'native skill tool check accepted a missing development dependency\n' >&2
	exit 1
fi
grep -q 'missing native skill tool dependency gamma' "$tmp/development-dependency.out"

printf '%s\n' '{"version":"1.0.0"}' >"$first/node_modules/gamma/package.json"
rm "$second/bun.lock"
if run_tools --check >"$tmp/lock.out" 2>&1; then
	printf 'native skill tool check accepted a missing lockfile\n' >&2
	exit 1
fi
grep -q 'missing native skill tool lockfile' "$tmp/lock.out"

printf 'PASS: native skill tool installer enforces locks and declared dependencies\n'
