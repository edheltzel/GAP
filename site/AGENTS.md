# site/AGENTS.md

## Purpose

The public website at `https://gap.edheltzel.codes`: the landing page at `/` and the docs at `/docs/`. One Astro build on [Nimbus](https://nimbus-docs.com) (`@cloudflare/nimbus-docs`), static output in `site/dist/`.

## Ownership

- `src/pages/index.astro`, `src/pages/404.astro`, `src/layouts/Landing.astro`, `src/components/landing/`, `src/styles/landing/`, `src/scripts/`, `src/site.js`, `public/`: the landing page, its artwork and the share card. Wording comes from `src/site.js`; the install line there must equal the README's.
- `src/pages/[...slug].astro`, `src/layouts/`, `src/components/` (Header, Logo, `ui/`), `src/styles/globals.css`, `src/styles/prose.css`: the Nimbus docs shell. It is the Nimbus starter (`nimbus.json` records its version), restyled with GAP's tokens, fonts, wordmark and theme key.
- Docs content is not in `site/`. `src/lib/repo-pages.ts` reads the repo's `README.md` (`/docs/`) and `CONCEPTS.md` (`/docs/concepts/`) in place; `src/lib/repo-links.ts` adapts them. Edit those root files, not a copy.
- `scripts/*.mjs`: site checks. CI: `.github/workflows/site.yml`.

## Local Contracts

- Publish only files listed in `repoPages` (`src/lib/repo-links.ts`). Never add `AGENTS.md`, `docs/plans/`, `docs/solutions/`, handoffs, `global/`, or other operational material.
- A README or CONCEPTS link to a published page (`README.md#…`, `CONCEPTS.md`) becomes its site URL; a link to any other repo file goes to that file on GitHub (`blob/master`). Images stay relative (`docs/images/…`) so Astro builds them in; the Markdown versions point them at `raw.githubusercontent.com`.
- Old `/docs/<page>` URLs from the earlier Starlight site redirect in `public/_redirects` to README anchors. A README heading rename breaks its anchor: update `_redirects` and the landing links in the same change. `npm run links` fails on a missing anchor.
- Every page shares `public/og.png` (1200x630); there are no per-page cards.
- Light and dark choice is stored under `gap-theme` on both the landing page and the docs.
- Node `22.23.4` or newer (Astro 7.4's minimum); `.nvmrc` pins the version CI and Cloudflare Pages use. Dependencies are exact versions in `package.json` with `package-lock.json`. Site tooling stays in `site/`; nothing here runs from the repo root or touches Go tooling.
- Deploy: Cloudflare Pages project `gap-website` (`gap-website-7fs.pages.dev`), git `edheltzel/GAP`, production branch `master`, root directory `site`, build command `npm run build`, output directory `dist`, env `NODE_VERSION=22.23.4` (keep equal to `.nvmrc`) and production `SITE_URL=https://gap.edheltzel.codes`, preview deployments off, build watch paths `site/*`, `README.md`, `CONCEPTS.md`, `docs/images/*`. It serves `gap.edheltzel.codes` (DNS `CNAME gap → gap-website-7fs.pages.dev`, proxied). The old project `gap-site` (`gap-site-292.pages.dev`, archived repo `edheltzel/gap-site`) has deployments off and still serves the pre-migration site as the rollback: move the domain back and point the CNAME at `gap-site-292.pages.dev`. `gap-site.pages.dev` belongs to someone else; never use it.

## Work Guidance

- Nimbus workflow (upgrade, `migrate`, `diff`, `add`): `node_modules/@cloudflare/nimbus-docs/dist/docs-for-agents.md`, matched to the installed release. After a Nimbus upgrade, run `npx @cloudflare/nimbus-docs migrate --dry-run --diff` and keep `nimbus.json` committed.
- `nimbus-docs lint` does not inspect the docs collection (its loader reads the repo root, not a folder under `src/content`); `npm run links` covers link checking instead.
- After changing `src/lib/`, delete `node_modules/.astro/` before building: the content cache keys on file contents, not on the loader, and keeps the old output.

## Verification

From `site/`, after `npm ci`:

- `npm run typecheck`
- `npm run build`
- `npm run links`, `npm run og-image`
- `npm run contrast`, `npm run toc`, `npm run logo-round`, `npm run flash`, `npm run hero-start`, `npm run hero-spin` (need Chrome; `CHROME_PATH` overrides `/usr/bin/google-chrome`; hero-start and hero-spin fail under macOS headless Chrome, so Linux CI is their gate)
