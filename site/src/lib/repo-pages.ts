import { readFileSync } from "node:fs";
import { glob } from "astro/loaders";
import type { DataStore, Loader, LoaderContext } from "astro/loaders";
import { githubUrl, isRepoRelative } from "./repo-links";

/**
 * The public docs are the repo's own Markdown, read in place from the GAP
 * checkout (this site lives in `site/`, so `..` is the repo root). Nothing is
 * copied into the site. Add a file here to publish it; everything else in the
 * repo (AGENTS.md, docs/plans/, docs/solutions/, handoffs) stays unpublished.
 */
export const repoPages: Record<string, string> = {
  "README.md": "docs",
  "CONCEPTS.md": "docs/concepts",
};

/** Each line of `body` outside code fences, through `fn`; fenced lines unchanged. */
function mapProse(body: string, fn: (line: string) => string): string[] {
  let fence: string | null = null;
  return body.split("\n").map((line) => {
    const open = /^\s*(`{3,}|~{3,})/.exec(line);
    if (open) fence = fence === null ? open[1] : null;
    return open || fence !== null ? line : fn(line);
  });
}

/** The first `# Heading` outside code fences: the page title GitHub shows. */
function firstHeading(body: string): string | undefined {
  const titles: string[] = [];
  mapProse(body, (line) => {
    const heading = /^#\s+(.+?)\s*#*\s*$/.exec(line);
    if (heading) titles.push(heading[1].replace(/`([^`]*)`/g, "$1"));
    return line;
  });
  return titles[0];
}

/**
 * The page's Markdown version (`/docs/index.md`, llms-full.txt) is served away
 * from the repo, so its relative links and images point at GitHub instead.
 */
const absoluteLinks = (body: string) =>
  mapProse(body, (line) =>
    line.replace(/(!?)\[([^\]]*)\]\(([^)\s]+)\)/g, (all, bang: string, text: string, url: string) =>
      isRepoRelative(url) ? `${bang}[${text}](${githubUrl(url, bang === "!")})` : all,
    ),
  ).join("\n");

export function repoPagesLoader(): Loader {
  const inner = glob({
    base: "..",
    pattern: Object.keys(repoPages),
    generateId: ({ entry }) => repoPages[entry] ?? entry,
  });
  return {
    name: "gap-repo-pages",
    load: (context: LoaderContext) => {
      const store = new Proxy(context.store, {
        get(target, key) {
          const value = Reflect.get(target, key, target);
          if (key !== "set") return typeof value === "function" ? value.bind(target) : value;
          return (entry: Parameters<DataStore["set"]>[0]) =>
            target.set(entry.body ? { ...entry, body: absoluteLinks(entry.body) } : entry);
        },
      });
      // README.md and CONCEPTS.md carry no frontmatter; their H1 is the title.
      const parseData: LoaderContext["parseData"] = (props) => {
        const file = props.filePath ? new URL(props.filePath, context.config.root) : undefined;
        const title = file ? firstHeading(readFileSync(file, "utf8")) : undefined;
        return context.parseData({ ...props, data: { title: title ?? props.id, ...props.data } });
      };
      return inner.load(
        new Proxy(context, {
          get(target, key, receiver) {
            if (key === "store") return store;
            if (key === "parseData") return parseData;
            return Reflect.get(target, key, receiver);
          },
        }),
      );
    },
  };
}
