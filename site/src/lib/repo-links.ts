import type { MdastPluginDefinition } from "satteri";

const blob = "https://github.com/edheltzel/GAP/blob/master/";
const raw = "https://raw.githubusercontent.com/edheltzel/GAP/master/";

/** A link the site can't serve: relative, so a repo file the docs point at. */
export const isRepoRelative = (url: string) => !!url && !/^([a-z][a-z0-9+.-]*:|\/|#|\?)/i.test(url);

/** That repo file on GitHub. Pages sit at the repo root, so paths resolve from there. */
export const githubUrl = (url: string, image = false) =>
  new URL(url.replace(/^\.\//, ""), image ? raw : blob).href;

/**
 * Docs pages are repo-root Markdown written for GitHub. Two fixes make the
 * HTML read right on the site:
 * - The layout already prints the page title, so the file's leading `# H1`
 *   is dropped.
 * - A relative link to a repo file the site doesn't publish (`gap.toml`,
 *   `cmd/gap/AGENTS.md#sync-steps`) goes to that file on GitHub, as it does
 *   in the README there. Anchors and absolute URLs are left alone; images are
 *   left to Astro, which builds them into the site.
 */
export function repoLinks(): MdastPluginDefinition {
  return {
    name: "gap:repo-links",
    heading(node, ctx) {
      if (node.depth !== 1 || ctx.data.gapTitleDropped) return;
      ctx.data.gapTitleDropped = true;
      ctx.removeNode(node);
    },
    link(node, ctx) {
      if (isRepoRelative(node.url)) ctx.setProperty(node, "url", githubUrl(node.url));
    },
  };
}
