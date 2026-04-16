# Design System Bootstrap — Reusable Prompt

**Purpose.** Generate a `DESIGN.md` source-of-truth file at the project root **plus** a live HTML style guide page that visualizes it. Produces a framework-agnostic design spec any coding agent (Claude, Codex, Cursor, Copilot, etc.) can read to generate matching UI in any stack.

**When to use.** Any time you start (or adopt) a project that needs a portable, machine-readable design system — typically once per project, early.

---

## How to use

1. Fill in the placeholders in the `# Inputs` block below.
2. Copy everything from `# BEGIN PROMPT` to `# END PROMPT` into your agent.
3. Let the agent investigate, write, verify. Review the generated files.
4. Iterate on anything that doesn't match your brand (palette naming, moods, do/don'ts).

## Placeholders

| Placeholder             | What to fill in                                                                 |
|-------------------------|----------------------------------------------------------------------------------|
| `{{PROJECT_NAME}}`      | Human-readable name, e.g. *Acme Logistics*.                                      |
| `{{PROJECT_ONE_LINER}}` | One sentence describing what the product does and who it's for.                  |
| `{{PROJECT_ROOT}}`      | Absolute path to the repo, e.g. `/Users/me/Sites/acme`.                          |
| `{{FRAMEWORK}}`         | Framework in use: `Astro`, `Next.js`, `Remix`, `SvelteKit`, `Nuxt`, `Vite+React`, `plain HTML/CSS`, etc. |
| `{{STYLING_SYSTEM}}`    | CSS approach: `Tailwind v4`, `Tailwind v3`, `CSS Modules`, `vanilla CSS`, `styled-components`, `Panda CSS`, etc. |
| `{{STYLE_GUIDE_ROUTE}}` | URL path where the visual page should mount, e.g. `/system/design-md`, `/style-guide`, `/_design`. |
| `{{BRANCHING_RULES}}`   | This project's branching policy (feature-branch required? worktree required? base branch name?). Leave `none` if no rules. |
| `{{MOOD_HINT}}` *(optional)* | One-line mood the brand aspires to (e.g. *"calm institutional, editorial, medical-journal"*). Leave `discover` to have the agent infer from existing UI. |

---

# BEGIN PROMPT

## Task

You are bootstrapping a design system for **{{PROJECT_NAME}}** — *{{PROJECT_ONE_LINER}}*.

Repo root: `{{PROJECT_ROOT}}`
Framework: `{{FRAMEWORK}}`
Styling system: `{{STYLING_SYSTEM}}`
Style-guide route: `{{STYLE_GUIDE_ROUTE}}`
Branching rules: `{{BRANCHING_RULES}}`
Mood hint: `{{MOOD_HINT}}`

You will produce **two artifacts** in one PR:

1. **`DESIGN.md`** at the project root — the portable, framework-agnostic source of truth.
2. **A visual style-guide page** at `{{STYLE_GUIDE_ROUTE}}` — renders every section of `DESIGN.md` at real size using the project's own component primitives.

Work in this order — do not skip phases.

---

## Phase 1 · Investigate (do not write anything yet)

Discover the brand tokens that already exist. Read in parallel:

- **Color / typography tokens.** Look for CSS custom-property files (`global.css`, `tokens.css`, `theme.ts`, `tailwind.config.*`, design-tokens JSON, etc.). Extract every named color scale with its hex values, every font family with its role.
- **Site configuration.** Look for a single file that centralizes identity (site name, URL, contact, social). Typical names: `site.ts`, `config/site.*`, `app.config.*`.
- **Component primitives.** Find the low-level Button, Text, Container/Wrapper, Card, Input, Icon components. Note their variants, sizes, states, prop names.
- **Brand constraints baked into docs.** Read any `CLAUDE.md`, `AGENTS.md`, `README.md`, `CONTRIBUTING.md`, or `docs/*` for written rules like "Heading color must stay dark" or "Primary CTA is always X." These constraints belong in `DESIGN.md`.
- **Existing design system pages.** If there are pages like `/system/*` or `/style-guide`, read them so you match existing tone and layout conventions.
- **A representative page** — usually the home or landing page — to infer atmosphere, density, and motion. This informs Section 1 of `DESIGN.md`.

If `{{MOOD_HINT}}` is `discover`, infer the mood from the representative page. Otherwise treat the hint as authoritative for Section 1.

Report what you found in ≤10 bullets before writing. If critical tokens are missing (e.g. no color scale defined anywhere), stop and ask.

---

## Phase 2 · Write `DESIGN.md`

Create `{{PROJECT_ROOT}}/DESIGN.md` using the **VoltAgent Stitch 9-section schema** below. Every section is mandatory.

### Writing rules (these make the file portable)

- **Hex values + semantic roles + numeric sizes only.** Never CSS-variable names. Never utility-class names. Never framework-specific component names.
  - ✅ `Primary button fill: #XXXXXX (Brand/800), 60 px tall, 24 px horizontal padding, 8 px corner radius.`
  - ❌ `Primary button: className="bg-brand-800 h-15 px-6 rounded-lg"`.
- **Use tables for palettes and scales**, bullets for principles, fenced code blocks only for the Agent Prompt Guide.
- **State the *why* alongside the *what*** when a rule is non-obvious (e.g. "Accent must stay dark because every heading on the site relies on it for ≥ 4.5:1 contrast on white").
- **Include contrast math** for at least 4 color pairs (heading on background, body on background, primary button, one failure case the team must avoid).
- **Prefer ranges to magic numbers** for responsive type (mobile / tablet / desktop / wide columns).
- **Do NOT invent tokens that aren't in the codebase.** If the codebase has no shadow system, the Depth section says "site is flat, no shadow system defined — propose one before using shadows."
- **Do NOT recommend a third-party UI library.** Components are built from the primitives you found in Phase 1.
- **Preserve existing brand constraints** verbatim from `CLAUDE.md` / docs. These are the load-bearing rules.

### The 9 sections

**1. Visual Theme & Atmosphere.**
One-line mood. Then 4–6 bulleted design principles specific to this brand (not generic). Density statement. Motion stance. Copy-tone stance.

**2. Color Palette & Roles.**
One sub-section per color scale found in Phase 1, each with: name + role statement, full step table (`step | hex | role`), and a hard rule or two about the scale ("never use above step 400 as heading ink" etc.). Then a **Semantic Roles** table mapping functional roles (page background, surface 1, heading ink, body ink, muted ink, divider, brand highlight, primary button fill/text/hover, scrim) to hex + token name in both light and dark mode (if dark mode exists — otherwise light only and note it). Then a **Contrast & Accessibility** subsection listing ≥ 4 contrast pairs with computed ratios and WCAG ratings.

**3. Typography Rules.**
Family table (role → family → fallback). Loading method in a single sentence. Display scale (token → mobile/tablet/desktop/wide sizes). Text scale (token → size, line-height, weight, role). Then conventions: default heading weight, body weight, eyebrow pattern, link treatment, list marker style, blockquote style.

**4. Component Stylings.**
One sub-section per discovered primitive. For each: absolute specs (height, padding, corner radius, font, transition, focus), variant table (fill/text/hover per variant), size table where applicable, state list. Cover at minimum: Buttons, Cards, Form inputs, Navigation (top nav + mobile), Footer, Tag/Pill, Section eyebrow pattern.

**5. Layout Principles.**
Container max width + horizontal padding at breakpoints. Section vertical rhythm (desktop + mobile). Grid note. Spacing scale (multiples of 4 px is a safe default — list preferred stops). Content width caps (prose line length, hero headline line count).

**6. Depth & Elevation.**
Shadow tokens as a table (`name | spec | use`). If the project is flat, say so explicitly — don't invent shadows. State that surfaces should differ by tone first, shadow second.

**7. Do's and Don'ts.**
Two parallel bulleted lists, 5–8 items each. Concrete and project-specific, not generic.

**8. Responsive Behavior.**
Breakpoint table (`name | min-width | note`). Mobile-first rule. Touch target minimum. Typography scaling rule. Navigation collapse rule. Grid collapse rule. Section padding behavior.

**9. Agent Prompt Guide.**
Start with a **Quick palette block** (fenced code block) that lists the top ~6 hex values with roles and the fonts — paste-ready context any agent can drop into its system prompt. Then 3–5 **Ready-to-use agent prompts** (landing section, primary CTA, form, dark-mode variant, etc.), each written in second person and grounded in `DESIGN.md` section numbers. End with **Guardrails for agents**: a short list of "when to stop and ask" rules (new color? new font? new shadow? new component? conflict between looks-cool and on-brand?).

### Front-matter

Open `DESIGN.md` with a short purpose block:

```md
# DESIGN.md — {{PROJECT_NAME}}

> **Purpose.** This file is the source of truth for the visual design intent of {{PROJECT_NAME}}. It is framework-agnostic by design: any coding agent (Claude, Codex, Cursor, Copilot, etc.) and any stack (React, Vue, Svelte, Astro, vanilla HTML) should be able to read this and produce UI that matches the brand.
>
> **Rule of translation.** This file states *what* the design is (hex values, numeric sizes, semantic roles, states). It does not state *how* to implement it (no utility-class names, no component import paths, no framework idioms). Agents translate intent into whatever primitives the current codebase uses.
```

---

## Phase 3 · Build the visual style-guide page

Create one page at `{{STYLE_GUIDE_ROUTE}}` that renders each section of `DESIGN.md` visually, in order. This page is a **mirror** of `DESIGN.md`, not a redesign of it.

### Requirements

- **Use the project's own component primitives** (the Text / Button / Wrapper / Card found in Phase 1). The page dogfoods the system it documents — if a primitive drifts off-spec later, this page breaks visibly before production pages do.
- **Sticky table of contents** near the top with anchor links to each of the 9 sections. Each section target needs a scroll-margin offset so anchors don't land under the sticky bar.
- **Dark-mode aware** if the project supports dark mode. Every swatch, surface, and text role must render correctly in both modes.
- **Every swatch shows its hex value.** Designers copy hex, not token names.
- **Every type token is rendered at actual size** with the token name, breakpoint sizes, line-height, and weight beside it.
- **Every button variant × size** is visible simultaneously — reference pages must not require interaction.
- **Cards, form inputs, tag pills, and the hero-scrim pattern** each get a live example.
- **The Agent Prompt Guide section** renders the quick palette block as a selectable, monospaced code block — Ed (or future users) need to copy it with one highlight.
- **Link to `/DESIGN.md`** from the hero so the raw spec is one click away.
- **Discoverability.** If there's an existing sitemap/overview page in the project (e.g. `/system/overview`), add a link to the new page there too.

### Non-requirements

- Do not add client-side JavaScript for this page beyond what the project already uses. Anchors + CSS are enough.
- Do not add design-system management features (editable tokens, live color pickers, export buttons). This is a *reference* page, not a tool.

---

## Phase 4 · Verify

1. **Build.** Run the project's build command end-to-end. Zero errors. Confirm the new route is in the built output.
2. **HTTP.** Start the dev server and `curl` the route. Expect `HTTP 200`. Save the HTML to a temp file.
3. **Anchors.** `grep` the saved HTML for each of the 9 section `id` attributes. All nine must be present.
4. **Content markers.** `grep` the HTML for: at least one hex value from every color scale, at least one token name from every type scale, and a "DESIGN.md" reference in the hero or agent-guide section.
5. **Dark-mode sanity.** If dark mode is in scope, grep for ≥ 10 dark-mode utility combos or equivalent CSS rules on the page.
6. **Dogfood.** Confirm the page imports the same Button / Text / Wrapper the rest of the site uses. If not, explain why.

Report the verification output inline in your summary.

---

## Rules of engagement

- **Branching.** Follow `{{BRANCHING_RULES}}`. If the project requires a feature branch or worktree, create it *before* writing files.
- **One PR.** Both files (and any overview-page link) land in the same PR. `DESIGN.md` and the visual page must not drift apart.
- **No assumptions.** If Phase 1 doesn't yield a token (no shadow system, no spacing scale, no existing primitive for something you need), stop and ask — do not invent.
- **Preserve existing brand rules.** If a pre-existing `CLAUDE.md` / docs file states a load-bearing rule (e.g. *"never use the accent scale for body text,"* *"primary heading color must stay dark to maintain contrast"*), copy it into `DESIGN.md` §2 verbatim.
- **Be terse in commit messages, thorough in the files themselves.** The file *is* the documentation; commit messages just say what changed.
- **Never use third-party UI kits** (Material, Ant, Chakra, shadcn) whose tokens conflict with what Phase 1 discovered. Build from the discovered primitives.

---

## Output contract

When you finish, report:

1. List of files created / modified (paths).
2. Phase 1 summary: palette scales found, fonts found, primitives found, brand constraints captured.
3. Verification results (build status, HTTP status, anchor grep counts, content-marker counts).
4. Any questions or unresolved ambiguities flagged during investigation.
5. A one-sentence spoken summary of what landed.

# END PROMPT

---

## Appendix · Why this prompt is structured this way

- **Inline schema.** The 9-section schema is quoted in full so the prompt works even with no internet access.
- **Investigate-before-writing** prevents the agent from inventing a design system from thin air. The codebase already has tokens; the prompt forces the agent to find them first.
- **Hex + semantic roles + numeric sizes** is the single rule that makes `DESIGN.md` portable between agents and frameworks. It's repeated because it's the most-violated rule.
- **Visual page mirrors `DESIGN.md` exactly** so drift between spec and reality surfaces as a visible bug on a page people actually load.
- **Verification by grep** (not a human reviewing screenshots) means the prompt can be run by any agent and the check reproduced identically.
- **Placeholders are the minimum set** that vary between projects. Palette, typography, and components vary too, but those are *discovered* not *specified* — that's what keeps this prompt short and reusable.

---

## Source

This prompt's 9-section schema is drawn from the [VoltAgent awesome-design-md](https://github.com/VoltAgent/awesome-design-md) collection, which packages design-system references from popular websites so coding agents can generate matching UI from a plain markdown file.
