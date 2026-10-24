#!/bin/sh
# Install gap and say whether the shell can find it.
# Run from a checkout, or piped from GitHub, which clones one first.
set -eu

repo="https://github.com/edheltzel/GAP.git"

if ! command -v go >/dev/null 2>&1; then
	echo "Go is not installed. Install Go 1.25 or newer, then run this again." >&2
	exit 1
fi

case "$0" in
*/install.sh|install.sh)
	cd "$(dirname "$0")/.." || exit 1
	;;
*)
	if ! command -v git >/dev/null 2>&1; then
		echo "git is not installed. Install git, then run this again." >&2
		exit 1
	fi
	mkdir -p "$HOME/.gap/checkouts"
	dir="$(mktemp -d "$HOME/.gap/checkouts/gap.XXXXXX")"
	git clone --depth 1 --branch master "$repo" "$dir" >/dev/null
	cd "$dir" || exit 1
	;;
esac

go install ./cmd/gap

bin="$(go env GOBIN)"
if [ -z "$bin" ]; then
	bin="$(go env GOPATH)/bin"
fi

echo "Installed $bin/gap"
echo "Keep the source checkout at $PWD; gap reads it when it runs."
case ":$PATH:" in
*":$bin:"*)
	echo "Your shell can find it. Next: gap sync"
	;;
*)
	echo "Your shell cannot find it yet. Add this line to your shell config, then open a new terminal:"
	echo "  export PATH=\"$bin:\$PATH\""
	;;
esac
