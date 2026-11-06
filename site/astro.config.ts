import { defineConfig } from "astro/config";
import tailwindcss from "@tailwindcss/vite";
import nimbus, {
  defineConfig as defineNimbusConfig,
} from "@cloudflare/nimbus-docs";
import { tableScroll } from "@cloudflare/nimbus-docs/markdown";
import { siteDescription, siteName, ogImage } from "./src/site.js";
import { repoLinks } from "./src/lib/repo-links.ts";

/** Public origin. Override with SITE_URL. The subdomain serves the site at /, so there is no `base`. */
const site = process.env.SITE_URL || "https://gap.edheltzel.codes";

const nimbusConfig = defineNimbusConfig({
  site,
  title: siteName,
  description: siteDescription,
  locale: "en",
  homeLabel: "GAP",
  github: "https://github.com/edheltzel/GAP",
  // `{path}` is the page file relative to site/ (`../README.md`); the docs
  // route resolves the `..` so the link opens README.md at the repo root.
  editPattern: "https://github.com/edheltzel/GAP/edit/master/site/{path}",
  socialImage: ogImage.path,
  socialImageAlt: ogImage.alt,
  sidebar: {
    items: [
      { label: "Overview", link: "/docs/" },
      { label: "Concepts", link: "/docs/concepts/" },
    ],
  },
});

export default defineConfig({
  output: "static",
  vite: {
    plugins: [tailwindcss()],
  },
  prefetch: {
    prefetchAll: true,
    defaultStrategy: "hover",
  },
  integrations: [
    nimbus(nimbusConfig, {
      rules: {
        "nimbus/frontmatter-shape": "error",
        "nimbus/internal-link": "error",
      },
      markdown: {
        mdastPlugins: [repoLinks()],
        hastPlugins: [tableScroll()],
      },
    }),
  ],
});
