import { markdownToMdast } from "satteri";
import type { MdastNode, MdastPluginDefinition } from "satteri";

/**
 * The public docs are the repo's own Markdown, read in place from the GAP
 * checkout (this site lives in `site/`, so `..` is the repo root). Nothing is
 * copied into the site. Add a file here to publish it; everything else in the
 * repo (AGENTS.md, docs/plans/, docs/solutions/, handoffs) stays unpublished.
 * Keys are repo paths, values the page ids (`docs/concepts` is /docs/concepts/).
 */
export const repoPages: Record<string, string> = {
  "README.md": "docs",
  "CONCEPTS.md": "docs/concepts",
};

const repo = "https://repo.invalid/";

/**
 * Where a link written for GitHub goes from the site, or undefined to leave it
 * alone (absolute URLs, site paths, anchors). The pages sit at the repo root, so
 * relative paths resolve from there: a published page becomes its site URL, an
 * image its raw file on GitHub, any other repo file its GitHub page.
 */
export function repoUrl(url: string, image = false): string | undefined {
  if (!url || /^([a-z][a-z0-9+.-]*:|\/|#|\?)/i.test(url)) return undefined;
  const resolved = new URL(url, repo);
  const path = decodeURIComponent(resolved.pathname.slice(1));
  image ||= /\.(avif|gif|jpe?g|png|svg|webp)$/i.test(path);
  const page = image ? undefined : repoPages[path];
  if (page) return `/${page}/${resolved.hash}`;
  const base = image
    ? "https://raw.githubusercontent.com/edheltzel/GAP/master/"
    : "https://github.com/edheltzel/GAP/blob/master/";
  return base + resolved.href.slice(repo.length);
}

/**
 * The source span of a link destination: the token after `](` (link, image)
 * or `]:` (definition), where `open` is the index of the label's `[`. A
 * `<…>` destination or a bare one with escapes and balanced parentheses, per
 * CommonMark. Undefined when the source doesn't have that shape.
 */
function destinationSpan(src: string, open: number, definition: boolean): [number, number] | undefined {
  let i = open + 1;
  for (let depth = 1; i < src.length && depth > 0; i++) {
    if (src[i] === "\\") i++;
    else if (src[i] === "[") depth++;
    else if (src[i] === "]") depth--;
  }
  if (src[i] !== (definition ? ":" : "(")) return undefined;
  i++;
  while (/\s/.test(src[i] ?? "")) i++;
  const from = i;
  if (src[i] === "<") {
    for (i++; i < src.length && src[i] !== ">"; i++) if (src[i] === "\\") i++;
    return src[i] === ">" ? [from, i + 1] : undefined;
  }
  for (let depth = 0; i < src.length && !/\s/.test(src[i]); i++) {
    if (src[i] === "\\") i++;
    else if (src[i] === "(") depth++;
    else if (src[i] === ")" && depth-- === 0) break;
  }
  return i > from ? [from, i] : undefined;
}

/**
 * The Markdown version of a page (`/docs/index.md`, llms-full.txt) is served
 * away from the repo, so its links, images and link definitions go through
 * `repoUrl` too, images included. Nodes come from the parsed tree, so code
 * spans and fences are never touched; each destination token is replaced in
 * the source, so titles and the rest of the text stay as written.
 */
export function absoluteLinks(body: string): string {
  const edits: { from: number; to: number; url: string }[] = [];
  const walk = (node: MdastNode) => {
    if ((node.type === "link" || node.type === "image" || node.type === "definition") && node.position) {
      const target = repoUrl(node.url, node.type === "image");
      const start = node.position.start.offset;
      const open = start === undefined ? -1 : body.indexOf("[", start);
      const span = target && open >= 0 ? destinationSpan(body, open, node.type === "definition") : undefined;
      if (target && span) edits.push({ from: span[0], to: span[1], url: target });
    }
    if ("children" in node) for (const child of node.children as MdastNode[]) walk(child);
  };
  walk(markdownToMdast(body, { position: true }));
  let out = body;
  for (const edit of edits.sort((a, b) => b.from - a.from)) out = out.slice(0, edit.from) + edit.url + out.slice(edit.to);
  return out;
}

/**
 * The same pages as HTML. The layout already prints the page title, so the
 * file's leading `# H1` is dropped. Links go through `repoUrl`; images are
 * left to Astro, which builds them into the site.
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
      const target = repoUrl(node.url);
      if (target) ctx.setProperty(node, "url", target);
    },
    definition(node, ctx) {
      const target = repoUrl(node.url);
      if (target) ctx.setProperty(node, "url", target);
    },
  };
}
