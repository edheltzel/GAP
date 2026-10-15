# Design spec (`DESIGN.md` + style guide)

Generate **two artifacts** for the current project in one PR:

1. **`DESIGN.md`** at the project root — the portable, framework-agnostic source of truth. Any coding agent (Claude, Codex, Cursor, Copilot) and any stack (React, Vue, Svelte, Astro, vanilla HTML) should be able to read it and produce UI that matches the brand.
2. **A visual style-guide page** that renders every section of `DESIGN.md` at real size using the project's own component primitives.

Work the phases in order — do not skip phases.

Discovery over invention: read tokens that already exist. If a token this spec needs is missing, stop and ask — do not invent one.

---

## Phase 0 · Resolve inputs (do this first)

You are not handed placeholders — discover them. Resolve each input below from the project, then apply any overrides the user passed in `$ARGUMENTS`. Ask the user ONLY for an input you genuinely cannot determine.

| Input | How to resolve |
|---|---|
| **Project root** | Git root of the cwd (`git rev-parse --show-toplevel`), else cwd. |
| **Project name** | `package.json` `name`, the site-config identity file, or the repo directory name. |
| **One-liner** | `package.json` `description`, the README's first line, or the site config. If absent, ask. |
| **Framework** | Infer from deps / config files: `Astro`, `Next.js`, `Remix`, `SvelteKit`, `Nuxt`, `Vite+React`, or `plain HTML/CSS`. |
| **Styling system** | Infer from deps / config: `Tailwind v4`, `Tailwind v3`, `CSS Modules`, `vanilla CSS`, `styled-components`, `Panda CSS`. |
| **Style-guide route** | Default `/style-guide`. For a routed app pick a path that fits its conventions (`/system/design-md`, `/_design`); for a non-routed/static project, a standalone `style-guide.html`. |
| **Branching rules** | Read `CLAUDE.md` / `AGENTS.md` / `CONTRIBUTING` for the branching/worktree policy. |
| **Mood** | `discover` — infer from the representative page — unless the user states one. |

`$ARGUMENTS` is a free-form override string. Parse `key:value` pairs from it (e.g. `route:/system/design mood:"calm editorial" name:"Acme" framework:Astro`); any override wins over auto-detection. If `$ARGUMENTS` is empty, auto-detect everything and discover the mood.

Restate the resolved inputs in ≤8 bullets before starting Phase 1. If a critical input is missing and undiscoverable (no color tokens defined anywhere, no one-liner), stop and ask.

---

## Phase 1 · Investigate (do not write anything yet)

Discover the brand tokens that already exist. Read in parallel:

- **Color / typography tokens.** Look for CSS custom-property files (`global.css`, `tokens.css`, `theme.ts`, `tailwind.config.*`, design-tokens JSON, etc.). Extract every named color scale with its hex values, every font family with its role.
- **Site configuration.** Look for a single file that centralizes identity (site name, URL, contact, social). Typical names: `site.ts`, `config/site.*`, `app.config.*`.
- **Component primitives.** Find the low-level Button, Text, Container/Wrapper, Card, Input, Icon components. Note their variants, sizes, states, prop names.
- **Brand constraints baked into docs.** Read any `CLAUDE.md`, `AGENTS.md`, `README.md`, `CONTRIBUTING.md`, or `docs/*` for written rules like "Heading color must stay dark" or "Primary CTA is always X." These constraints belong in `DESIGN.md`.
- **Existing design system pages.** If there are pages like `/system/*` or `/style-guide`, read them so you match existing tone and layout conventions.
- **A representative page** — usually the home or landing page — to infer atmosphere, density, and motion. This informs Section 1 of `DESIGN.md`.

If the resolved mood is `discover`, infer it from the representative page. Otherwise treat the stated mood as authoritative for Section 1.

Report what you found in ≤10 bullets before writing. If critical tokens are missing (e.g. no color scale defined anywhere), stop and ask.

---

## Phase 2 · Write `DESIGN.md`

Create `DESIGN.md` at the project root using the **VoltAgent Stitch 9-section schema** below. Every section is mandatory.

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

Open `DESIGN.md` with a short purpose block (substitute the resolved project name):

```md
# DESIGN.md — <Project Name>

> **Purpose.** This file is the source of truth for the visual design intent of <Project Name>. It is framework-agnostic by design: any coding agent (Claude, Codex, Cursor, Copilot, etc.) and any stack (React, Vue, Svelte, Astro, vanilla HTML) should be able to read this and produce UI that matches the brand.
>
> **Rule of translation.** This file states *what* the design is (hex values, numeric sizes, semantic roles, states). It does not state *how* to implement it (no utility-class names, no component import paths, no framework idioms). Agents translate intent into whatever primitives the current codebase uses.
```

---

## Phase 3 · Build the visual style-guide page

Create one page at the resolved style-guide route that renders each section of `DESIGN.md` visually, in order. This page is a **mirror** of `DESIGN.md`, not a redesign of it.

### Requirements

- **Use the project's own component primitives** (the Text / Button / Wrapper / Card found in Phase 1). The page dogfoods the system it documents — if a primitive drifts off-spec later, this page breaks visibly before production pages do.
- **Sticky table of contents** near the top with anchor links to each of the 9 sections. Each section target needs a scroll-margin offset so anchors don't land under the sticky bar.
- **Dark-mode aware** if the project supports dark mode. Every swatch, surface, and text role must render correctly in both modes.
- **Every swatch shows its hex value.** Designers copy hex, not token names.
- **Every type token is rendered at actual size** with the token name, breakpoint sizes, line-height, and weight beside it.
- **Every button variant × size** is visible simultaneously — reference pages must not require interaction.
- **Cards, form inputs, tag pills, and the hero-scrim pattern** each get a live example.
- **The Agent Prompt Guide section** renders the quick palette block as a selectable, monospaced code block — copy with one highlight.
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

- **Branching.** Follow the branching rules resolved in Phase 0. If the project requires a feature branch or worktree, create it *before* writing files.
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
