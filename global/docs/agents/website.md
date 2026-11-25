# Website pass

A change can make a project's website wrong. Check for one and keep it matching.

## Rules

- Detect: look for a site folder (`site/`, `website/`, `docs-site/`, `web/`, `www/`), a site generator config (Astro, Next, Docusaurus, VitePress, MkDocs, Hugo, Jekyll, Eleventy, Nimbus, and similar), a deploy config (wrangler, netlify, vercel, `CNAME`, a Pages workflow), or a site link in the README. Stop once one hit is confirmed. Do not scan the whole tree.
- If found, read the site's `AGENTS.md` or `FOLDER.md` first.
- Learn where its pages come from before editing. Sites often skip their framework's conventions: custom loaders, pages built from the README or other repo files, generated pages.
- Look up the framework's current docs, as of today, before relying on how it works.
- Update every page the change affects: pages generated from changed repo files, heading anchors, redirects, landing links, and navigation.
- Run the site's own build or check.
- If there is no website, say so in one line in the report.

## Done

Your report names the site you checked and the pages you updated, or says the project has no website.
