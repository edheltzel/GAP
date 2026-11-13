#!/bin/sh
# Run scripts/install.sh against a local fake release and repo in a throwaway HOME.
# Usage: sh scripts/install_test.sh
set -eu

here="$(cd "$(dirname "$0")" && pwd)"
installer="$here/install.sh"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
fails=0

pass() { echo "ok - $1"; }
fail() { echo "not ok - $1"; fails=$((fails + 1)); }

case "$(uname -s)" in Darwin) os=darwin ;; *) os=linux ;; esac
case "$(uname -m)" in x86_64|amd64) arch=amd64 ;; *) arch=arm64 ;; esac
archive="gap_${os}_${arch}.tar.gz"

g() { git -c user.name=test -c user.email=test@example.com -c init.defaultBranch=master "$@"; }

src="$work/src"
mkdir -p "$src/global" "$src/cmd/gap"
echo policy > "$src/global/AGENTS.md"
echo "package main" > "$src/cmd/gap/main.go"
g init --quiet "$src"
g -C "$src" add -A
g -C "$src" commit --quiet -m one

rel="$work/release"
mkdir -p "$rel/stage"
printf '#!/bin/sh\necho fake-gap\n' > "$rel/stage/gap"
chmod 755 "$rel/stage/gap"
tar -czf "$rel/$archive" -C "$rel/stage" gap
if command -v shasum >/dev/null 2>&1; then
	sum="$(shasum -a 256 "$rel/$archive" | awk '{print $1}')"
else
	sum="$(sha256sum "$rel/$archive" | awk '{print $1}')"
fi

run_install() {
	env HOME="$work/home" GAP_HOME= GAP_VERSION= GAP_BIN_DIR="$work/bin" \
		GAP_RELEASE_URL="file://$rel" GAP_REPO_URL="$src" \
		PATH="/usr/bin:/bin:/usr/sbin:/sbin:$(dirname "$(command -v git)")" \
		sh "$installer" > "$work/out" 2>&1
}

mkdir -p "$work/home"
echo "0000000000000000000000000000000000000000000000000000000000000000  $archive" > "$rel/checksums.txt"
if run_install; then
	fail "bad checksum aborts"
else
	if grep -q "Checksum mismatch" "$work/out"; then pass "bad checksum aborts"; else fail "bad checksum message"; cat "$work/out"; fi
fi
if [ -e "$work/bin/gap" ] || [ -e "$work/home/.gap/checkout" ]; then fail "bad checksum installs nothing"; else pass "bad checksum installs nothing"; fi

echo "other  gap_plan9_mips.tar.gz" > "$rel/checksums.txt"
if run_install; then fail "missing checksum entry aborts"; else pass "missing checksum entry aborts"; fi

echo "$sum  $archive" > "$rel/checksums.txt"
if run_install; then pass "good checksum installs"; else fail "good checksum installs"; cat "$work/out"; fi
if [ -x "$work/bin/gap" ] && [ "$("$work/bin/gap")" = fake-gap ]; then pass "binary installed"; else fail "binary installed"; fi
if [ -f "$work/home/.gap/checkout/global/AGENTS.md" ] && [ "$(git -C "$work/home/.gap/checkout" symbolic-ref --short HEAD)" = master ]; then pass "checkout cloned on master"; else fail "checkout cloned on master"; fi
if git -C "$work/home/.gap/checkout" rev-parse --verify --quiet refs/remotes/origin/HEAD >/dev/null; then pass "origin/HEAD set"; else fail "origin/HEAD set"; fi
if grep -q 'export PATH=' "$work/out"; then pass "PATH hint printed"; else fail "PATH hint printed"; cat "$work/out"; fi

echo two > "$src/global/AGENTS.md"
g -C "$src" commit --quiet -am two
if run_install && [ "$(cat "$work/home/.gap/checkout/global/AGENTS.md")" = two ]; then pass "rerun fast-forwards checkout"; else fail "rerun fast-forwards checkout"; cat "$work/out"; fi

if [ "$fails" -ne 0 ]; then
	echo "$fails failed"
	exit 1
fi
echo "all passed"
