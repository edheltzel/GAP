import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { glob } from "astro/loaders";
import type { DataStore, Loader, LoaderContext } from "astro/loaders";
import { markdownToMdast } from "satteri";
import type { MdastNode } from "satteri";
import { absoluteLinks, repoPages } from "./repo-links";
import { pageDescriptions } from "../site.js";

/** The text of the first `# Heading`: the page title GitHub shows. */
function firstHeading(node: MdastNode): string | undefined {
  if (node.type === "heading" && node.depth === 1) {
    const text = (n: MdastNode): string =>
      "value" in n ? String(n.value) : "children" in n ? (n.children as MdastNode[]).map(text).join("") : "";
    return text(node);
  }
  if (!("children" in node)) return undefined;
  for (const child of node.children as MdastNode[]) {
    const found = firstHeading(child);
    if (found) return found;
  }
  return undefined;
}

/** The last commit date of a file, for the sitemap lastmod and "Last updated". */
function commitDate(file: URL): Date | undefined {
  try {
    const iso = execFileSync("git", ["log", "-1", "--format=%cI", "--", fileURLToPath(file)], {
      cwd: fileURLToPath(new URL(".", file)),
      encoding: "utf8",
    }).trim();
    return iso ? new Date(iso) : undefined;
  } catch {
    return undefined;
  }
}

/** The pages in `repoPages`, globbed from the repo root (`..` from site/). */
export function repoPagesLoader(): Loader {
  const inner = glob({
    base: "..",
    pattern: Object.keys(repoPages),
    generateId: ({ entry }) => repoPages[entry] ?? entry,
  });
  return {
    name: "gap-repo-pages",
    load: (context: LoaderContext) => {
      // The stored body feeds the Markdown versions; its links must work off GitHub.
      const store = new Proxy(context.store, {
        get(target, key) {
          const value = Reflect.get(target, key, target);
          if (key !== "set") return typeof value === "function" ? value.bind(target) : value;
          return (entry: Parameters<DataStore["set"]>[0]) =>
            target.set(entry.body ? { ...entry, body: absoluteLinks(entry.body) } : entry);
        },
      });
      // README.md and CONCEPTS.md carry no frontmatter; their H1 is the title, site.js holds the description, git gives the date.
      const parseData: LoaderContext["parseData"] = (props) => {
        const file = props.filePath ? new URL(props.filePath, context.config.root) : undefined;
        const title = file ? firstHeading(markdownToMdast(readFileSync(file, "utf8"))) : undefined;
        return context.parseData({ ...props, data: { title: title ?? props.id, description: pageDescriptions[props.id as keyof typeof pageDescriptions], lastUpdated: file && commitDate(file), ...props.data } });
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
