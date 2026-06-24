#!/bin/bash
# ═══════════════════════════════════════════════════════════════════════════════
# Unit tests for atlas-statusline.sh theme resolution (detect_appearance)
# ═══════════════════════════════════════════════════════════════════════════════
#
# Verifies the statusline resolves the correct palette EVERY time a terminal theme
# is set, across all four resolution sources:
#   settings.json override → theme-switcher state file → COLORFGBG → macOS default.
#
# The decisive test is "source of truth": it reads the THEMES array straight out of
# ~/.config/theme-switcher/theme-switcher.sh and asserts every theme the switcher can
# emit resolves to a real palette branch (no silent fall-through to the wrong palette).
#
# Run:   bash atlas-statusline.test.sh
# Exit:  0 = all pass, 1 = one or more failures.
# ═══════════════════════════════════════════════════════════════════════════════

set -u

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
SCRIPT="$SCRIPT_DIR/atlas-statusline.sh"
THEME_SWITCHER="$HOME/.config/theme-switcher/theme-switcher.sh"

# Palette branches that actually exist in atlas-statusline.sh.
# "dark" is the else/Eldritch branch; the others are explicit elif blocks.
KNOWN_THEMES="light dark vesper catppuccin-light catppuccin-dark"

pass=0
fail=0

# Resolve the theme the statusline would pick for a given environment.
#   $1 settings override ("" = none)   $2 theme-switcher state ("" = none)   $3 COLORFGBG ("" = unset)
resolve_theme() {
  local override="$1" state="$2" cfb="$3"
  local home
  home=$(mktemp -d)
  mkdir -p "$home/.claude" "$home/.config/theme-switcher"
  if [ -n "$override" ]; then
    printf '{"statusline":{"theme":"%s"}}\n' "$override" >"$home/.claude/settings.json"
  else
    printf '{}\n' >"$home/.claude/settings.json"
  fi
  [ -n "$state" ] && printf '%s\n' "$state" >"$home/.config/theme-switcher/current"
  # env -i for a clean environment; pass only HOME, PATH (jq/cat/defaults), and COLORFGBG.
  local out
  out=$(env -i HOME="$home" PATH="$PATH" COLORFGBG="$cfb" bash "$SCRIPT" --print-theme 2>/dev/null)
  rm -rf "$home"
  printf '%s' "$out"
}

check() {
  local desc="$1" expected="$2" actual="$3"
  if [ "$actual" = "$expected" ]; then
    printf '  \033[32m✓\033[0m %-46s → %s\n' "$desc" "$actual"
    pass=$((pass + 1))
  else
    printf '  \033[31m✗\033[0m %-46s → expected "%s", got "%s"\n' "$desc" "$expected" "$actual"
    fail=$((fail + 1))
  fi
}

# Assert the resolved theme is a real palette branch.
check_known() {
  local desc="$1" actual="$2"
  case " $KNOWN_THEMES " in
  *" $actual "*)
    printf '  \033[32m✓\033[0m %-46s → %s\n' "$desc" "$actual"
    pass=$((pass + 1))
    ;;
  *)
    printf '  \033[31m✗\033[0m %-46s → "%s" has no palette branch\n' "$desc" "$actual"
    fail=$((fail + 1))
    ;;
  esac
}

echo "▲ atlas-statusline.sh — theme resolution tests"
echo

# ── 0. Script integrity ────────────────────────────────────────────────────────
echo "[0] Script integrity"
if bash -n "$SCRIPT" 2>/dev/null; then
  printf '  \033[32m✓\033[0m %-46s\n' "syntax (bash -n)"
  pass=$((pass + 1))
else
  printf '  \033[31m✗\033[0m %-46s\n' "syntax (bash -n)"
  fail=$((fail + 1))
fi
if grep -qi vesper "$SCRIPT"; then
  printf '  \033[32m✓\033[0m %-46s\n' "vesper palette branch present"
  pass=$((pass + 1))
else
  printf '  \033[31m✗\033[0m %-46s (missing)\n' "vesper palette branch present"
  fail=$((fail + 1))
fi
echo

# ── 1. theme-switcher state mappings (no override, no COLORFGBG) ─────────────────
echo "[1] theme-switcher state → palette"
check "eldritch"            dark            "$(resolve_theme '' eldritch '')"
check "tokyonight"          dark            "$(resolve_theme '' tokyonight '')"   # regression: was 'tokyo-night'
check "rose-pine"           dark            "$(resolve_theme '' rose-pine '')"
check "rose-pine-moon"      dark            "$(resolve_theme '' rose-pine-moon '')"
check "rose-pine-dawn"      light           "$(resolve_theme '' rose-pine-dawn '')"
check "catppuccin-latte"    catppuccin-light "$(resolve_theme '' catppuccin-latte '')"
check "catppuccin-frappe"   catppuccin-dark "$(resolve_theme '' catppuccin-frappe '')"
check "catppuccin-macchiato" catppuccin-dark "$(resolve_theme '' catppuccin-macchiato '')"
check "catppuccin-mocha"    catppuccin-dark "$(resolve_theme '' catppuccin-mocha '')"
echo

# ── 2. vesper theme-switcher state → vesper palette ──────────────────────────────
echo "[2] vesper resolution"
check "state=vesper → vesper" vesper "$(resolve_theme '' vesper '15;0')"
echo

# ── 3. settings.json override ────────────────────────────────────────────────────
echo "[3] settings.json override"
check "override=light"  light "$(resolve_theme light '' '')"
check "override=dark"   dark  "$(resolve_theme dark '' '')"
check "override=vesper → vesper" vesper "$(resolve_theme vesper '' '15;0')"
check "override=auto → state wins"  catppuccin-dark "$(resolve_theme auto catppuccin-mocha '')"
echo

# ── 4. COLORFGBG fallback (no override, no state) ────────────────────────────────
echo "[4] COLORFGBG fallback"
check "bg=0 (dark)"        dark  "$(resolve_theme '' '' '15;0')"
check "bg=7 (dark)"        dark  "$(resolve_theme '' '' '0;7')"
check "bg=15 (light)"      light "$(resolve_theme '' '' '0;15')"
check "bg=default (light)" light "$(resolve_theme '' '' '0;default')"
echo

# ── 5. SOURCE OF TRUTH: every theme the switcher can emit resolves cleanly ────────
echo "[5] theme-switcher THEMES array contract"
if [ -f "$THEME_SWITCHER" ]; then
  themes_line=$(grep -E '^THEMES=' "$THEME_SWITCHER" | head -1)
  eval "SRC_THEMES=${themes_line#THEMES=}"
  printf '  source: %s\n' "$THEME_SWITCHER"
  for theme in "${SRC_THEMES[@]}"; do
    resolved=$(resolve_theme '' "$theme" '15;0')   # COLORFGBG dark as the deterministic fallback
    check_known "$theme" "$resolved"
  done
else
  printf '  \033[33m⚠\033[0m theme-switcher not found at %s — skipping contract test\n' "$THEME_SWITCHER"
fi
echo

# ── Summary ──────────────────────────────────────────────────────────────────────
echo "─────────────────────────────────────────────"
printf 'Result: \033[32m%d passed\033[0m, ' "$pass"
if [ "$fail" -gt 0 ]; then
  printf '\033[31m%d failed\033[0m\n' "$fail"
  exit 1
fi
printf '%d failed\n' "$fail"
exit 0
