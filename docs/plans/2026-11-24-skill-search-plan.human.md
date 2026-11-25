# Skill search in gap tui

| Step | What you get | Status |
|---|---|---|
| U1 | `gap add owner/repo --skill name` works on repos without a gap-utility.toml | already on master |
| U2 | gap can search skills.sh | done |
| U3 | `/` in gap tui: type, see hits with installs and trust, enter, pick Everywhere or This project | done |
| U4 | README and AGENTS.md | done |

**Goal:** find any skills.sh skill in the TUI and install it globally or per project, like Empryo's browser.

**Now:** built and tested on branch feat/tui-aup-skill-search.

**Next:** review the PR.

**Risks:** non-GitHub hits cannot install and are hidden.
