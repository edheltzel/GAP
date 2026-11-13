#!/bin/sh
# Install gap and say whether the shell can find it.
# With Go on PATH and no GAP_VERSION or GAP_RELEASE_URL, build from a checkout (this one, or a fresh clone when piped).
# Otherwise download a release binary, verify its sha256, and keep a checkout at $GAP_HOME/checkout (~/.gap/checkout) for gap to publish.
# GAP_VERSION pins a release tag, GAP_BIN_DIR picks the binary folder (~/.local/bin), GAP_RELEASE_URL and GAP_REPO_URL override the download and clone sources.
set -eu

repo="${GAP_REPO_URL:-https://github.com/edheltzel/GAP.git}"

need() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "$1 is not installed. Install $1, then run this again." >&2
		exit 1
	fi
}

path_hint() {
	case ":$PATH:" in
	*":$1:"*)
		echo "Your shell can find it. Next: gap sync"
		;;
	*)
		echo "Your shell cannot find it yet. Add this line to your shell config, then open a new terminal:"
		echo "  export PATH=\"$1:\$PATH\""
		;;
	esac
}

sha256() {
	if command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | awk '{print $1}'
	elif command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		echo "Neither shasum nor sha256sum is installed; cannot verify the download." >&2
		exit 1
	fi
}

install_release() {
	need curl
	need git
	need tar
	case "$(uname -s)" in
	Darwin) os=darwin ;;
	Linux) os=linux ;;
	*)
		echo "No gap release for $(uname -s). Install Go 1.25 or newer, then run this again." >&2
		exit 1
		;;
	esac
	case "$(uname -m)" in
	x86_64|amd64) arch=amd64 ;;
	arm64|aarch64) arch=arm64 ;;
	*)
		echo "No gap release for $(uname -m). Install Go 1.25 or newer, then run this again." >&2
		exit 1
		;;
	esac
	if [ -n "${GAP_RELEASE_URL:-}" ]; then
		base="$GAP_RELEASE_URL"
	elif [ -n "${GAP_VERSION:-}" ]; then
		base="https://github.com/edheltzel/GAP/releases/download/$GAP_VERSION"
	else
		base="https://github.com/edheltzel/GAP/releases/latest/download"
	fi
	archive="gap_${os}_${arch}.tar.gz"
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	echo "Downloading $base/$archive"
	curl -fsSL "$base/$archive" -o "$tmp/$archive"
	curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"
	want="$(awk -v f="$archive" '$2 == f || $2 == "*" f {print $1}' "$tmp/checksums.txt")"
	if [ -z "$want" ]; then
		echo "checksums.txt has no entry for $archive; nothing installed." >&2
		exit 1
	fi
	got="$(sha256 "$tmp/$archive")"
	if [ "$got" != "$want" ]; then
		echo "Checksum mismatch for $archive (want $want, got $got); nothing installed." >&2
		exit 1
	fi
	mkdir -p "$tmp/x"
	tar -xzf "$tmp/$archive" -C "$tmp/x"
	if [ ! -f "$tmp/x/gap" ]; then
		echo "$archive has no gap binary; nothing installed." >&2
		exit 1
	fi

	gap_home="${GAP_HOME:-$HOME/.gap}"
	checkout="$gap_home/checkout"
	if [ -d "$checkout/.git" ]; then
		branch="$(git -C "$checkout" symbolic-ref --short HEAD 2>/dev/null || true)"
		if [ "$branch" != master ]; then
			echo "$checkout is on '$branch', not master. Switch it to master, then run this again." >&2
			exit 1
		fi
		git -C "$checkout" fetch --quiet origin master
		git -C "$checkout" merge --ff-only --quiet FETCH_HEAD
	elif [ -e "$checkout" ]; then
		echo "$checkout exists and is not a git checkout. Move it aside, then run this again." >&2
		exit 1
	else
		mkdir -p "$gap_home"
		git clone --quiet --branch master "$repo" "$checkout"
	fi
	git -C "$checkout" remote set-head origin master >/dev/null

	bin="${GAP_BIN_DIR:-$HOME/.local/bin}"
	mkdir -p "$bin"
	cp "$tmp/x/gap" "$bin/.gap.$$"
	chmod 755 "$bin/.gap.$$"
	mv -f "$bin/.gap.$$" "$bin/gap"

	echo "Installed $bin/gap"
	echo "Keep the checkout at $checkout; gap reads it when it runs."
	path_hint "$bin"
}

if [ -n "${GAP_VERSION:-}" ] || [ -n "${GAP_RELEASE_URL:-}" ] || ! command -v go >/dev/null 2>&1; then
	install_release
	exit 0
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
	# A single-branch clone leaves origin/HEAD unset, and gap sync refuses to publish without it.
	git -C "$dir" remote set-head origin master >/dev/null
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
path_hint "$bin"
