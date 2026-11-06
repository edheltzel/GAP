# GAP website

The landing page and docs at [gap.edheltzel.codes](https://gap.edheltzel.codes), built with [Astro](https://astro.build) and [Nimbus](https://nimbus-docs.com). The docs pages are the repo's own [README.md](../README.md) and [CONCEPTS.md](../CONCEPTS.md), read in place at build time.

## Local development

Needs Node 22.23.4 or newer (`.nvmrc`).

```bash
cd site
npm ci
npm run dev
```

`npm run build` writes the static site to `site/dist/`. Checks are listed in [AGENTS.md](AGENTS.md#verification).
