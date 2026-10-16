# Swift

Swift and Apple platforms - macOS desktop apps, iOS review, and cleanup.

## Installation

These skills live in this repo (`edheltzel/agent-config`, being renamed GAP). `gap sync` installs every skill listed in `global/skills.json`.

```bash
gap sync
```

One skill by exact name:

```bash
npx skills add edheltzel/agent-config -s <skill-name>
```

## Skills

| Skill | Description |
|-------|-------------|
| [`do-cleanup-swift`](./do-cleanup-swift/) | End-of-session cleanup pass for Swift code. |
| [`do-macos-swift-desktop`](./do-macos-swift-desktop/) | Build native macOS desktop applications in Swift using AppKit and SwiftUI. |
| [`do-review-ios`](./do-review-ios/) | Comprehensive iOS/SwiftUI code review with optional parallel agents |

## See Also

- [Project skills](../README.md)
- [Skill catalog](../../../README.md)
- [Docs](../../../docs/project/swift/) - human-facing pages for these skills
