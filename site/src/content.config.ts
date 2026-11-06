import { defineCollection } from "astro:content";
import { docsCollection, withNimbusMarkdown } from "@cloudflare/nimbus-docs/content";
import { repoPagesLoader } from "./lib/repo-pages";

export const collections = {
  // README.md and CONCEPTS.md, read from the repo root: /docs/ and /docs/concepts/.
  docs: defineCollection({
    loader: withNimbusMarkdown(repoPagesLoader()),
    schema: docsCollection().schema,
  }),
};
