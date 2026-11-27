#!/bin/sh
# Regenerates docs/images/tui-*.png from the current gap tui.
#
# Builds ./cmd/gap with -X main.sourceRootFor pointed at a checkout inside
# /tmp/gap-tui-demo, because a plain build falls through to ~/.gap/checkout.
# That checkout lists no components, so nothing is fetched and no distro hook
# runs. The demo then holds two local example utilities, docs-tools at project
# scope and formatter globally, plus one registered project.
#
# gap runs with HOME set inside the demo, both when called directly and when
# launched inside the tmux pane, and that HOME holds gap's config and lock. XDG_STATE_HOME holds the skills.sh lock that gap and
# npx skills write, and XDG_CONFIG_HOME plus the agent-specific homes
# (CLAUDE_CONFIG_DIR, CODEX_HOME, and the rest) hold every agent lookup, so an
# inherited value cannot point an install at the real home. formatter is the global
# utility, and its global copies land under CLAUDE_CONFIG_DIR inside the demo.
# Fixture git and gap's own git get an empty GIT_CONFIG_GLOBAL, no system config,
# and an empty template dir, and fixture commits disable hooks.
#
# The only gap commands used are add, update of the local formatter, and the
# status the TUI runs itself. Never gap sync, and never an update of a distro
# component: launchd labels are per user, so a component install hook would
# re-point the real user's agent at the temp home.
#
# Capture: tmux drives the TUI and capture-pane records the pane; freeze renders
# that ANSI to a PNG, which is then scaled to 1400x760. vhs is installed but its
# GIF output is blank on this machine, so it is not used.
#
# / search always goes through the local proxy, never straight to skills.sh. The
# proxy serves the fixture from cmd/gap/tui_search_test.go, and the live catalog
# only when SHOTS_SEARCH_LIVE=1. Go dials through Network.framework and ignores
# HOSTALIASES and RESOLV_CONF, so that one recording sets GAP_SKILL_SEARCH_URL.
set -eu

root=$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)
home=$(mktemp -d "${TMPDIR:-/tmp}/gap-tui-demo.XXXXXX")
out="$root/docs/images"
# 150 columns by 42 rows keeps the footer on one line at the rendered width.
cols=150
rows=42
# A private socket, so another demo or an earlier run cannot attach to this session.
sock=gaptui-$$
# Screenshots that rendered, listed at the end.
wrote=
# Empty until the search proxy starts. Set before the trap: set -u reads it on any early exit.
proxy=
# Same set git rev-parse --local-env-vars prints (cmd/gap/main.go gitLocalEnv), plus the prompt.
# An inherited GIT_DIR would make the fixture's git add and commit write the caller's index.
# GIT_CONFIG_GLOBAL, GIT_CONFIG_NOSYSTEM and GIT_TEMPLATE_DIR keep the caller's global
# config, system config and template hooks out of both the fixture and gap's own git.
git_env()
{
	env -u GIT_ALTERNATE_OBJECT_DIRECTORIES -u GIT_CONFIG -u GIT_CONFIG_PARAMETERS \
		-u GIT_CONFIG_COUNT -u GIT_OBJECT_DIRECTORY -u GIT_DIR -u GIT_WORK_TREE \
		-u GIT_IMPLICIT_WORK_TREE -u GIT_GRAFT_FILE -u GIT_INDEX_FILE \
		-u GIT_NO_REPLACE_OBJECTS -u GIT_REPLACE_REF_BASE -u GIT_PREFIX \
		-u GIT_SHALLOW_FILE -u GIT_COMMON_DIR GIT_TERMINAL_PROMPT=0 \
		GIT_CONFIG_GLOBAL="$home/.gitconfig" GIT_CONFIG_NOSYSTEM=1 \
		GIT_TEMPLATE_DIR="$home/git-template" "$@"
}

for tool in tmux freeze magick go git python3; do
	command -v "$tool" >/dev/null || { echo "tui-shots: $tool is not installed" >&2; exit 1; }
done
case "$home" in
/tmp/gap-tui-demo.*|"${TMPDIR:-/tmp}"/gap-tui-demo.*) ;;
*) echo "tui-shots: refusing to use $home" >&2; exit 1 ;;
esac
trap 'tmux -L "$sock" kill-server 2>/dev/null || true; if [ -n "$proxy" ]; then kill "$proxy" 2>/dev/null || true; wait "$proxy" 2>/dev/null || true; fi; rm -rf "$home"' EXIT

build() {
	echo "building gap"
	# Built from a copy. sourceRoot() walks runtime.Caller back to the checkout
	# the binary was compiled in, and a binary built inside the real repo would
	# publish that repo's gap.lock instead of the demo's.
	rm -rf "$home/src" "$home/.gap"
	mkdir -p "$home/src" "$home/.gap"
	cp "$root/go.mod" "$root/go.sum" "$home/src/"
	cp -R "$root/cmd" "$home/src/cmd"
	# The installed gap bakes its checkout in with -X main.sourceRootFor, and a
	# plain build records no path, so sourceRoot falls through to checkoutRoot
	# and reads ~/.gap/checkout. Point it at a checkout inside the demo.
	# A copy, not the working tree: the demo's status reads this checkout's
	# gap.lock, and the working tree's lock must not be what it reads.
	mkdir -p "$home/.gap/checkout/global" "$home/.gap/checkout/cmd/gap"
	printf 'schema = "2"\nname = "demo"\n' >"$home/.gap/checkout/gap.toml"
	printf 'schema = "2"\n' >"$home/.gap/checkout/gap.lock"
	printf 'policy\n' >"$home/.gap/checkout/global/AGENTS.md"
	cp "$home/src/cmd/gap/main.go" "$home/.gap/checkout/cmd/gap/main.go"
	git_env git -C "$home/.gap/checkout" init -q -b master
	(cd "$home/src" && go build -ldflags "-X main.sourceRootFor=${home}/.gap/checkout" -o "$home/bin/gap" ./cmd/gap)
}



# A local utility: one skill, no hooks, so add/update/remove run nothing.
repo() {
	name=$1
	skill=$2
	dir=$home/repos/$name
	rm -rf "$dir"
	mkdir -p "$dir/skills/$skill"
	cat >"$dir/gap-utility.toml" <<EOF
schema = "1"
name = "$name"
skills = ["$skill"]
EOF
	cat >"$dir/skills/$skill/SKILL.md" <<EOF
---
name: $skill
description: Example skill for the gap tui screenshots.
---

# $skill
EOF
	git_env git -C "$dir" init -q -b master
	git_env git -C "$dir" add .
	git_env git -C "$dir" -c user.name=gap -c user.email=gap@example.com -c core.hooksPath=/dev/null commit -q -m "example"
}

# Second commit, so `u` has something to move. formatter's update hook pauses,
# which is what holds the run screen on its spinner and an unfinished bar.
bump() {
	dir=$home/repos/$1
	printf '\nChecked.\n' >>"$dir/skills/$2/SKILL.md"
	git_env git -C "$dir" add .
	git_env git -C "$dir" -c user.name=gap -c user.email=gap@example.com -c core.hooksPath=/dev/null commit -q -m "check"
	# The hook has to arrive in the commit the update moves to. One added to the
	# commit already pinned never runs, because gap update sees nothing to move.
	if [ "$1" = "formatter" ]; then
		cat >"$dir/slow-update.sh" <<'EOF'
#!/bin/sh
echo "fetching formatter"
sleep 1
echo "comparing trees"
sleep 1
echo "writing the new pin"
sleep 120
EOF
		chmod +x "$dir/slow-update.sh"
		printf '\n[hooks]\nupdate = "./slow-update.sh"\n' >>"$dir/gap-utility.toml"
		git_env git -C "$dir" add .
		git_env git -C "$dir" -c user.name=gap -c user.email=gap@example.com -c core.hooksPath=/dev/null commit -q -m "slow update"
	fi
}

demo() {
	# Keep the directory mktemp made. Removing it and recreating it would let
	# another user on a shared temp parent claim the path in between.
	mkdir -p "$home/bin" "$home/repos" "$out" \
		"$home/.config" "$home/.local/state" "$home/.local/share" "$home/.cache" \
		"$home/git-template"
	# Empty on purpose: fixture git and gap's own git read this instead of the caller's.
	: >"$home/.gitconfig"
	build
	repo docs-tools check-links
	repo formatter format-go
	git_env git clone -q "$home/repos/docs-tools" "$home/site"

	# The demo checkout lists no components, so there is nothing to fetch and no
	# hook to run. Trust keeps the two example utilities' hooks quiet; neither
	# declares one. The checkout has to exist before the adds below, because
	# they read it to record the pins.
	mkdir -p "$home/.config/gap"
	printf '[trust]\nrepos = ["%s", "%s"]\n' "$home/repos/docs-tools" "$home/repos/formatter" \
		>"$home/.config/gap/gap.toml"

	# docs-tools is the project utility. formatter is the global one, installed for
	# claude-code only: its folder comes from CLAUDE_CONFIG_DIR, set inside the demo
	# below, so nothing lands in the real home. --agent with -y skips the prompts.
	gap add "$home/repos/docs-tools" --project -y --agent pi --skill check-links
	gap add "$home/repos/formatter" --global -y --agent claude-code --skill format-go
	# After the pin, so `u` has a commit to move to and formatter's update hook
	# runs. A commit made before the add is what the pin records.
	bump docs-tools check-links
	bump formatter format-go
	# Registered because of the project add above; show it under its own name.
	printf '%s\n' "# Docs site" >"$home/site/README.md"
}

# HOME keeps gap's config and lock inside the demo, since userConfigDir builds them
# from it. XDG_STATE_HOME keeps the skills.sh lock there too. The agent homes,
# CLAUDE_CONFIG_DIR included, override inherited agent paths, which agent detection
# honors before HOME.
gap() {
	(cd "$home/site" && HOME="$home" CLAUDE_CONFIG_DIR="$home/.claude" \
		XDG_CONFIG_HOME="$home/.config" XDG_STATE_HOME="$home/.local/state" \
		XDG_DATA_HOME="$home/.local/share" XDG_CACHE_HOME="$home/.cache" \
		CODEX_HOME="$home/.codex" GROK_HOME="$home/.grok" JCODE_HOME="$home/.jcode" \
		PI_CODING_AGENT_DIR="$home/.pi/agent" AUTOHAND_HOME="$home/.autohand" \
		HERMES_HOME="$home/.hermes" VIBE_HOME="$home/.vibe" REASONIX_HOME="$home/.reasonix" \
		SARVAM_HOME="$home/.sarvam" \
		GIT_CONFIG_GLOBAL="$home/.gitconfig" GIT_CONFIG_NOSYSTEM=1 \
		GIT_TEMPLATE_DIR="$home/git-template" \
		PATH="$home/bin:/opt/homebrew/bin:/usr/bin:/bin" TERM=xterm-256color NO_COLOR='' \
		"$home/bin/gap" "$@")
}

# searchSkills calls https://skills.sh/api/search unless GAP_SKILL_SEARCH_URL names
# another endpoint, so the fixture has to answer there. This proxy does. It serves
# the fixture from cmd/gap/tui_search_test.go, and the live catalog only when
# SHOTS_SEARCH_LIVE=1. Those hits are owner/repo sources, which is all searchSkills
# keeps, and vercel-labs and anthropics are two of the owners it marks trusted.
search_proxy() {
	# exec replaces this backgrounded function, so $! is python and the EXIT trap can reap it.
	exec python3 - "$1" <<'PY'
import json, os, sys, urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

fixture = {"skills": [
    {"id": "react", "source": "vercel-labs/agent-skills", "skillId": "react", "name": "react", "installs": 932043},
    {"id": "agent-reviews", "source": "pbakaus/agent-reviews", "skillId": "agent-reviews", "name": "agent-reviews", "installs": 1200},
    {"id": "frontend-design", "source": "anthropics/skills", "skillId": "frontend-design", "name": "frontend-design", "installs": 4180},
]}

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        body, code = json.dumps(fixture).encode(), 200
        # The readiness probe asks for /health and must not wait on skills.sh.
        if self.path != "/health" and os.environ.get("SHOTS_SEARCH_LIVE") == "1":
            try:
                with urllib.request.urlopen("https://skills.sh" + self.path, timeout=6) as resp:
                    body, code = resp.read(), resp.status
            except Exception:
                pass
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)
    def log_message(self, *a):
        pass

ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), Handler).serve_forever()
PY
}

# vhs on this machine writes a blank GIF and no Screenshot, so the pane is
# captured from tmux and drawn to a PNG. Keys are sent with `tmux send-keys`,
# one per line of the second argument, each followed by a short pause.
shot() {
	name=$1
	keys=$2
	url=${3:-}
	echo "recording tui-$name"
	tmux -L "$sock" kill-server 2>/dev/null || true
	# GAP_SKILL_SEARCH_URL points / search at the fixture proxy for the search shot.
	# Empty for every other shot, so they call skills.sh as usual.
	# NO_COLOR='' is empty on purpose: a nonempty value makes gap tui exit before it opens.
	tmux -L "$sock" -f /dev/null new-session -d -s shot -x "$cols" -y "$rows" -c "$home/site" \
		env GAP_SKILL_SEARCH_URL="$url" HOME="$home" CLAUDE_CONFIG_DIR="$home/.claude" \
		XDG_CONFIG_HOME="$home/.config" XDG_STATE_HOME="$home/.local/state" \
		XDG_DATA_HOME="$home/.local/share" XDG_CACHE_HOME="$home/.cache" \
		CODEX_HOME="$home/.codex" GROK_HOME="$home/.grok" JCODE_HOME="$home/.jcode" \
		PI_CODING_AGENT_DIR="$home/.pi/agent" AUTOHAND_HOME="$home/.autohand" \
		HERMES_HOME="$home/.hermes" VIBE_HOME="$home/.vibe" REASONIX_HOME="$home/.reasonix" \
		SARVAM_HOME="$home/.sarvam" \
		GIT_CONFIG_GLOBAL="$home/.gitconfig" GIT_CONFIG_NOSYSTEM=1 \
		GIT_TEMPLATE_DIR="$home/git-template" \
		PATH="$home/bin:/opt/homebrew/bin:/usr/bin:/bin" \
		TERM=xterm-256color COLORTERM=truecolor NO_COLOR='' "$home/bin/gap" tui
	sleep 4
	printf '%s\n' "$keys" | while IFS= read -r key; do
		[ -n "$key" ] || continue
		# send-keys treats each argument as one key, so a word is one keystroke.
		tmux -L "$sock" send-keys -t shot -l "$key"
		sleep 0.8
	done
	# The update shot has to land while its hook is still sleeping.
	if [ "$name" = "update" ]; then
		sleep 1
	else
		sleep 2
	fi
	if [ "$name" = "search" ]; then
		seen=0
		n=0
		while [ "$n" -lt 50 ]; do
			# The selected hit row starts with the pointer and ends in its install count, even
			# when the source is truncated, and the empty state has its own message. The typed
			# query, the spinner or an error matches neither.
			if tmux -L "$sock" capture-pane -t shot -p | LC_ALL=C grep -Eq '▌ +(✓ +)?[^ ]+ +[^ ]+ +[0-9][0-9.]*[kKmMbB]?( |│|$)|no installable hits for'; then
				seen=1
				break
			fi
			n=$((n + 1))
			sleep 0.2
		done
		if [ "$seen" -ne 1 ]; then
			echo "tui-shots: search results never appeared; tui-search.png left as it was" >&2
			tmux -L "$sock" capture-pane -t shot -p | sed -n '1,8p' >&2
			tmux -L "$sock" kill-server 2>/dev/null || true
			return 1
		fi
	fi
	tmux -L "$sock" capture-pane -t shot -p -e >"$home/$name.vt"
	tmux -L "$sock" kill-server 2>/dev/null || true
	# freeze renders the captured pane, ANSI colors included, to the PNG.
	freeze --language ansi --theme catppuccin-mocha \
		--width 1400 --font.size 15 --margin 0 --padding 16 \
		-o "$out/tui-$name.png" "$home/$name.vt" >/dev/null \
		|| { echo "tui-shots: freeze failed for $name" >&2; return 1; }
	[ -s "$out/tui-$name.png" ] || { echo "tui-shots: $name produced no still" >&2; return 1; }
	# freeze sizes to the text, so scale down to the gallery's 1400x760.
	magick "$out/tui-$name.png" -resize 1400x760 -background '#1e1e2e' -gravity center -extent 1400x760 "$out/tui-$name.png" \
		|| { echo "tui-shots: magick failed for $name" >&2; return 1; }
	wrote="$wrote $out/tui-$name.png"
}

# Always rebuild from this checkout. Reusing a binary left under /tmp would
# record whatever revision last wrote it, including another user's.
demo

fail=0
# Items opens on docs-tools, so the detail pane shows check-links.
only=${SHOTS_ONLY:-}
want() { [ -z "$only" ] || [ "$only" = "$1" ]; }
want items && { shot items "" || fail=1; }
# Status runs `gap status`, which only reads the local pins.
want status && { shot status "2" || fail=1; }
# Projects lists the one registered repo.
want projects && { shot projects "3" || fail=1; }
# Remove asks before it does anything; the session ends before enter, so
# nothing is removed.
want delete && { shot delete "d" || fail=1; }
# u runs `gap update formatter`. Its update hook sleeps, so the shot lands while
# the run is still going. y approves that hook.
want update && { shot update "j
u
y" || fail=1; }

if want search; then
# Search always goes through the proxy. It serves the fixture from
# cmd/gap/tui_search_test.go, and the live catalog only when SHOTS_SEARCH_LIVE=1.
# Go dials through Network.framework and ignores HOSTALIASES and RESOLV_CONF, so the
# only override that reaches it is GAP_SKILL_SEARCH_URL, set for this one recording.
port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
search_proxy "$port" &
proxy=$!
ready=0
i=0
while [ "$i" -lt 50 ]; do
	curl -fsS --max-time 2 "http://127.0.0.1:$port/health" >/dev/null 2>&1 && { ready=1; break; }
	i=$((i + 1))
	sleep 0.1
done
if [ "$ready" -ne 1 ]; then
	echo "tui-shots: the search proxy never answered" >&2
	fail=1
else
	shot search "/
r
e
a
c
t" "http://127.0.0.1:$port/api/search" || fail=1
fi
fi

[ -z "$wrote" ] || echo "wrote$wrote"
exit "$fail"
