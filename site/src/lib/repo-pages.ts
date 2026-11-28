import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
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

/**
 * The last commit date of a file, for the sitemap lastmod and "Last updated".
 * In a shallow clone the boundary commit seems to add every file, so a match
 * there is no date at all.
 */
function commitDate(file: URL): Date | undefined {
  const git = (...args: string[]) =>
    execFileSync("git", args, { cwd: fileURLToPath(new URL(".", file)), encoding: "utf8" }).trim();
  try {
    const [hash, iso] = git("log", "-1", "--format=%H %cI", "--", fileURLToPath(file)).split(" ");
    if (!hash || !iso) return undefined;
    const shallow = git("rev-parse", "--path-format=absolute", "--git-path", "shallow");
    if (existsSync(shallow) && readFileSync(shallow, "utf8").split("\n").includes(hash)) return undefined;
    return new Date(iso);
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
      const dates = new Map<string, Date | undefined>();
      const meta = (id: string, filePath: string | undefined) => {
        const file = filePath ? new URL(filePath, context.config.root) : undefined;
        if (file && !dates.has(file.href)) dates.set(file.href, commitDate(file));
        const description = pageDescriptions[id as keyof typeof pageDescriptions];
        return { description, lastUpdated: file && dates.get(file.href) };
      };
      const stamp = (id: string, filePath: string | undefined) => {
        const { description, lastUpdated } = meta(id, filePath);
        return `|${description ?? ""}|${lastUpdated?.toISOString() ?? ""}`;
      };
      // The stored body feeds the Markdown versions; its links must work off GitHub.
      // The glob loader reuses an entry whose Markdown is unchanged, so the stored
      // digest also carries the description and date: a change to either reloads it.
      const store = new Proxy(context.store, {
        get(target, key) {
          const value = Reflect.get(target, key, target);
          if (key === "get") {
            return (id: string) => {
              const entry = target.get(id);
              if (!entry?.digest) return entry;
              const tail = stamp(id, entry.filePath);
              const digest = String(entry.digest);
              return { ...entry, digest: digest.endsWith(tail) ? digest.slice(0, -tail.length) : undefined };
            };
          }
          if (key !== "set") return typeof value === "function" ? value.bind(target) : value;
          return (entry: Parameters<DataStore["set"]>[0]) =>
            target.set({
              ...entry,
              ...(entry.body ? { body: absoluteLinks(entry.body) } : {}),
              ...(entry.digest ? { digest: `${entry.digest}${stamp(entry.id, entry.filePath)}` } : {}),
            });
        },
      });
      // README.md and CONCEPTS.md carry no frontmatter; their H1 is the title, site.js holds the description, git gives the date.
      const parseData: LoaderContext["parseData"] = (props) => {
        const file = props.filePath ? new URL(props.filePath, context.config.root) : undefined;
        const title = file ? firstHeading(markdownToMdast(readFileSync(file, "utf8"))) : undefined;
        return context.parseData({ ...props, data: { title: title ?? props.id, ...meta(props.id, props.filePath), ...props.data } });
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
