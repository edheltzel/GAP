# YouTube Channel Mining — PAIUpgrade Run 2026-04-17

**Channels mined:** 6 (IndyDevDan, Simon Willison, Unsupervised Learning, AgriciDaniel, Chase-H-AI, The Diary of a CEO)
**Videos processed:** 30 (5 per channel)
**Techniques extracted:** ~56
**Skipped as non-relevant:** 10 (5 Diary-of-a-CEO + 4 UL security interviews + 1 AgriciDaniel contest)

---

## 🔴 CRITICAL — adopt immediately

### 1. Claude Code `/routines` — cloud-scheduled tasks
**Source:** Chase-H-AI — "/routines just completely changed Claude Code tasks" @ 00:09 (`Hd4Ck1BS4Kw`)

> "Now we can have Claude Code do things for us automatically in the cloud. We no longer are stuck with session-based loops."

- New Claude Code CLI command `/schedule` + `claude.ai/code/routines` web UI
- 3 trigger types: cron, API call, GitHub event
- Requires connected GitHub repo for output delivery
- 15 runs / 24h on Max plan

**Atlas relevance:** Atlas's `/schedule` and `/loop` skills currently wrap `CronCreate` (in-session only — dies when terminal closes). Routines are laptop-independent. Prime candidates for conversion: daily MEMORY consolidation, weekly PAIUpgrade runs, nightly IntegrityMaintenance.

### 2. Opus 4.7 default effort → `x-high` (silent cost bump)
**Source:** Chase-H-AI — "Claude Dropped Opus 4.7 and It's Not Even Close" @ 03:10 (`gc297hx4F7o`)

> "Claude Code now defaults to extra high... if you want to change that, all you need to do is do /effort."

- Tokenizer change + effort default shift = 1.0–1.35× higher input tokens vs 4.6
- Atlas sessions are burning more than they did pre-upgrade without any visible signal
- `/effort` levels: `low | medium | high | x-high | max`

**Atlas relevance:** Directly related to the earlier Tier 3 item #8 (ModelRouter xhigh) we marked N/A. Now that we see the cost implication, consider setting `effort` explicitly in `settings.json` or CLAUDE.md, at least for lightweight scheduled Routines.

### 3. 3-tier memory hierarchy (hot.md → index.md → wiki pages)
**Source:** AgriciDaniel — "Your AI Forgets Everything" @ 01:45 (`a2hgayvr-H4`) + Unsupervised Learning "Karpathy LLM Wiki" @ 29:10 (`uUForkn00mk`)

> "Hot I'm the is loaded automatically every session. Index is a one-line summary of every wiki page. Wiki pages are loaded only if needed."

- `hot.md` — always injected, tiny
- `index.md` — one-line-per-page catalog
- `wiki/{page}.md` — fetched on-demand via references
- Karpathy's "LLM wiki" pattern generalizes this

**Atlas relevance:** `MEMORY/` is currently a flat tree with `MEMORY.md` as the only index. Adopting the 3-tier model would cut per-session token cost significantly while preserving recall depth. Recall MCP's hybrid search complements, not replaces, this structure.

---

## 🟠 HIGH — evaluate this quarter

| # | Technique | Source | Atlas target |
|---|-----------|--------|--------------|
| 4 | **Till-done task list** (completion-enforced, not to-do) | IndyDevDan `RairMJflUSA` @ 06:33 | Algorithm EXECUTE loop / TaskList |
| 5 | **Per-agent expertise files** (boot + persist) | IndyDevDan `M30gp1315Y4` @ 10:08 | Subagent brief + `MEMORY/AGENTS/{name}.md` |
| 6 | **Prompt-complexity classifier** for model routing | IndyDevDan `M30gp1315Y4` @ 05:23 | ModelRouter.hook.ts (scikit-learn over heuristics) |
| 7 | **Domain-locked agent permissions** (scope per subagent) | IndyDevDan `M30gp1315Y4` @ 14:13 | SecurityValidator + PermissionDenied extension |
| 8 | **Budget/time ceilings** on spawned agents | IndyDevDan `TqjmTZRL31E` @ 11:28 | StopOrchestrator cost ceiling |
| 9 | **Library meta-skill YAML manifest** (cross-device distribution) | IndyDevDan `_vpNQ6IwP9w` @ 06:30 | atlas-config stow + manifest |
| 10 | **Cross-agent convergence scoring** (N/M agreement) | AgriciDaniel `aE295lLPO5A` @ 02:15 | Council skill / multi-agent aggregation |
| 11 | **Skill UI — reverse-engineer design systems into skills** | Chase-H-AI `Q9ty3eopOPs` @ 04:30 | frontend-design skill workflow |
| 12 | **Impeccable — anti-pattern names** (slop taxonomy) | Chase-H-AI `Q9ty3eopOPs` @ 01:10 | frontend-design SKILL.md |
| 13 | **Scheduled sync → SQLite → Datasette** | Simon Willison `BoPZltKDM-s` @ 09:03 | MEMORY/ queryable SQL layer |
| 14 | **CLAUDE.md as index pointer** (not context dump) | Chase-H-AI `kQu5pWKS8GA` @ 14:20 | Root CLAUDE.md refactor |
| 15 | **Context-clearing threshold @ 20-25% remaining** | Chase-H-AI `kQu5pWKS8GA` @ 08:15 | New hook / Algorithm rule |
| 16 | **Playwright CLI > Playwright MCP at scale** | Chase-H-AI `Q9ty3eopOPs` @ 18:30 | `qa` skill update |
| 17 | **`/review` native command** (may supersede Atlas skill) | Chase-H-AI `gc297hx4F7o` @ 03:45 | Audit Atlas /review skill |
| 18 | **Prime Directive: current-state → ideal-state gap** | UL `uUForkn00mk` @ 17:10 | Algorithm framing / TELOS integration |

---

## 🟡 MEDIUM — worth noting

| # | Technique | Source |
|---|-----------|--------|
| 19 | Write-failing-test-first → agent fixes against migration doc | Simon Willison `qy4ci7AoF9Y` @ 07:26 |
| 20 | Conformance suite as automated feedback loop | Simon Willison `bKrAcTf2pL4` @ 14:09 |
| 21 | Claude Code authors script once; cheaper model runs at scale | Simon Willison `T8xiMgmb8po` @ 21:43 |
| 22 | Example-JSON prompt + numeric checksum validation | Simon Willison `T8xiMgmb8po` @ 22:39 |
| 23 | Planner-worker tree with structured handoff | Simon Willison `bKrAcTf2pL4` @ 09:45 |
| 24 | Plain-file scratchpad for long-running continuity | Simon Willison `bKrAcTf2pL4` @ 46:13 |
| 25 | Terminal-to-HTML capture for reproducible sessions | Simon Willison `9o1_DL9uNlM` @ 06:10 |
| 26 | Tracking issue as agent work queue | Simon Willison `qy4ci7AoF9Y` @ 00:23 |
| 27 | Orchestrator-only input pattern (scale-invariant prompts) | IndyDevDan `RairMJflUSA` @ 05:54 |
| 28 | Adversarial board debate pattern (CEO + board personas) | IndyDevDan `TqjmTZRL31E` @ 04:43 |
| 29 | Brief-in / memo-out I/O contract | IndyDevDan `TqjmTZRL31E` @ 03:17 |
| 30 | Verification via action (not self-report) | IndyDevDan `RvowJ_hmLps` @ 19:07 |
| 31 | Hotload skills into system prompt at spawn | IndyDevDan `_vpNQ6IwP9w` @ 18:21 |
| 32 | Skills/Agents/Prompts as distinct primitives | IndyDevDan `_vpNQ6IwP9w` @ 03:00 |
| 33 | SERP-overlap clustering for research | AgriciDaniel `1C6-ZadOHW8` @ 01:10 |
| 34 | HTML diff report for state-drift detection | AgriciDaniel `1C6-ZadOHW8` @ 03:05 |
| 35 | Skill-submission security audit gate | AgriciDaniel `1C6-ZadOHW8` @ 00:15 |
| 36 | Recon → parallel specialist agent spawn | AgriciDaniel `aE295lLPO5A` @ 00:45 |
| 37 | `/canvas` — visual board from session context | AgriciDaniel `a2hgayvr-H4` @ 04:00 |
| 38 | Plugin marketplace compliance checklist | AgriciDaniel `aowxn2vHIuY` @ 03:20 |
| 39 | Roadmap section embedded in SKILL.md | AgriciDaniel `aowxn2vHIuY` @ 04:10 |
| 40 | Claude Design → Claude Code export handoff | Chase-H-AI `-tGH2tLwCEw` @ 08:10 |
| 41 | Question-driven pre-generation elicitation | Chase-H-AI `-tGH2tLwCEw` @ 03:45 |
| 42 | Obsidian vault hierarchy pattern for RAG | Chase-H-AI `kQu5pWKS8GA` @ 22:00 |
| 43 | General verifiability as eval target | UL `boqvjSoRaYQ` @ ~08:00 |

---

## ⏭️ Skipped

| Video | Channel | Reason |
|-------|---------|--------|
| 5 videos | The Diary of a CEO | All geopolitics/finance — no Atlas technique |
| `7TIShYx2iXs` | UL | Security product (Protegrity) — no PAI relevance |
| `psIkqnvhac0` | UL | Security product (Mallory) — no PAI relevance |
| `55jMLaTZ7Ho` | UL | SOC automation product — no PAI relevance |
| `woCapb_FYJY` | AgriciDaniel | Contest app build — no Atlas technique |

---

## 📊 Per-channel summary

| Channel | Videos mined | Techniques | Top pick |
|---------|:-:|:-:|---|
| IndyDevDan | 5 | 15 | Library meta-skill / till-done list / per-agent expertise files |
| Simon Willison | 5 | 11 | Scheduled sync → SQLite → Datasette |
| Unsupervised Learning | 1 + 4 skipped | 4 | Current-state → ideal-state gap / PI Upgrade loop |
| AgriciDaniel | 4 + 1 skipped | 13 | 3-tier memory hierarchy / cross-agent convergence |
| Chase-H-AI | 5 | 13 | `/routines` / Opus 4.7 effort / CLAUDE.md as index |
| The Diary of a CEO | 0 + 5 skipped | 0 | — |

---

## 🔍 Cross-cutting themes (appear across 3+ channels)

1. **Multi-agent orchestration with structured handoffs** — IndyDevDan, Simon Willison, AgriciDaniel, Chase-H-AI
2. **Memory/context as a hierarchy, not a flat file** — AgriciDaniel, UL, Chase-H-AI
3. **Cost/effort awareness as first-class concern** — IndyDevDan (budget ceilings), Chase-H-AI (effort levels), Simon Willison (cheaper-model execution)
4. **Skill system maturity** — IndyDevDan (library meta-skill), AgriciDaniel (marketplace compliance, roadmap embed), Chase-H-AI (Skill UI)

## Next actions

1. **Discuss with Ed:** Which Tier 1 items to pursue first (`/routines` adoption and memory hierarchy are the biggest wins).
2. **Related to earlier Tier 3 #8 (ModelRouter xhigh):** Revisit now that we know Opus 4.7's tokenizer/effort changes silently inflate cost — explicit effort control becomes higher priority.
3. **Queue Tier 1 items as separate issue tracking entries** rather than implementing all in this upgrade PR — this batch is already large.

---

**State file updated:** `claudecode/.claude/skills/PAIUpgrade/State/youtube-videos.json` — all 30 video IDs added to seen lists per channel, preserving legacy AI Jason / R Amjad entries for future reference.
