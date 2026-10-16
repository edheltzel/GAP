# macOS Swift Desktop

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-macos-swift-desktop
```

```bash
npx skills update do-macos-swift-desktop
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/swift/do-macos-swift-desktop)

## What it does

`macos-swift-desktop` supplies production-grade patterns for building native
macOS apps in Swift — `.app` bundles, document apps, menu bar utilities,
persistence, distribution, and GPU-accelerated rendering. It is organized as an
index and routing table: `SKILL.md` decides which of a dozen `references/` files
the task needs and loads only that. The defining constraint is that non-trivial
apps are always built as an AppKit + SwiftUI hybrid — AppKit owns the structural
shell, SwiftUI owns the content panels — never pure SwiftUI, which is missing
the multi-window, borderless-window, and low-level rendering APIs a real desktop
app needs.

## When to reach for it

Type `/do-macos-swift-desktop`, or the agent reaches for it automatically when a
task touches AppKit, `NSDocument`, `NSApplicationDelegate`, Swift+Metal, macOS
code signing, or Sparkle auto-updates.

Reach for it while *building or modifying* a macOS app and you want patterns that
match how shipping apps (CodeEdit, Rectangle, IINA, Ghostty) actually do it. For
a final tidy-and-review pass over Swift you've already written, use
[cleanup-swift](./cleanup-swift.md) — this skill authors, that one polishes.

## The hybrid mental model

AppKit owns the shell: `NSApplicationDelegate`, `NSWindow`, `NSWindowController`,
`NSDocument`, split views, toolbars. SwiftUI owns the panels: sidebars,
inspectors, settings, forms. Performance-critical content — text editors,
terminals, Metal canvases — is a custom `NSView` embedded via
`NSViewRepresentable`. This split is the skill's central rule of thumb; most
routing decisions flow from deciding which layer a piece of UI belongs to.

## How the routing works

`SKILL.md` is a table, not a tutorial. Each task shape maps to one reference
file — `document-apps.md`, `persistence.md`, `distribution.md`,
`advanced-rendering.md`, `swift-concurrency.md`, and more — so the agent pulls in
only the material it needs instead of a monolithic prompt. It also carries a
top-10 anti-patterns list (popovers for menu bar UI, pure SwiftUI shells,
missing notarization, force-unwrapping optional Apple APIs) and modern Swift
defaults: `@MainActor` by default, `async`/`await` over completion handlers, the
`Observable` macro, structured concurrency, deployment target macOS 14+.

## Where it fits

A reach-for-it-anytime standalone for macOS desktop work — the counterpart to the
web-focused skills in [`frontend/`](../frontend/), aimed at a platform they
don't cover. It pairs with [cleanup-swift](./cleanup-swift.md), which uses this
skill as its platform-conventions review lens at the end of a session.
