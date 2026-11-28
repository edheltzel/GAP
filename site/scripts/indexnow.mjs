import { execFileSync } from "node:child_process";
import { readdirSync } from "node:fs";

const site = new URL(process.env.SITE_URL || "https://gap.edheltzel.codes");
const sha = process.env.DEPLOY_SHA;
const before = process.env.BEFORE_SHA;
const waitMs = Number(process.env.DEPLOY_WAIT_MS || 20 * 60 * 1000);
const dryRun = process.argv.includes("--dry-run");

const keyFile = readdirSync(new URL("../public/", import.meta.url)).find((name) => /^[0-9a-f]{32}\.txt$/.test(name));
if (!keyFile) throw new Error("no IndexNow key file in public/");
const key = keyFile.slice(0, -4);

const git = (...args) => execFileSync("git", args, { cwd: new URL("../..", import.meta.url), encoding: "utf8" });

/** repoPages from site/src/lib/repo-links.ts at a commit: repo file -> page URL. */
function pagesAt(commit) {
  const source = git("show", `${commit}:site/src/lib/repo-links.ts`);
  const block = source.match(/repoPages[^{]*\{([^}]*)\}/)?.[1] ?? "";
  return new Map([...block.matchAll(/"([^"]+)":\s*"([^"]+)"/g)].map(([, file, id]) => [file, new URL(`/${id}/`, site).href]));
}

/**
 * The URLs a deploy changed, or "all" when site code changed. A root page maps
 * to its own URL; a page dropped from repoPages keeps its old URL so IndexNow
 * hears of the removal.
 */
function changedUrls() {
  if (!sha || !before || /^0+$/.test(before)) return "all";
  const now = pagesAt(sha);
  const then = pagesAt(before);
  const urls = new Set();
  for (const [file, url] of then) if (now.get(file) !== url) urls.add(url);
  for (const file of git("diff", "--name-only", "--no-renames", before, sha).split("\n").filter(Boolean)) {
    if (now.has(file)) urls.add(now.get(file));
    else if (then.has(file)) urls.add(then.get(file));
    else if (file.startsWith("site/") && !file.startsWith("site/scripts/") && file !== "site/AGENTS.md") return "all";
  }
  return [...urls];
}

async function text(path, timeoutMs = 15_000) {
  const res = await fetch(new URL(path, site), { cache: "no-store", signal: AbortSignal.timeout(timeoutMs) });
  if (res.status === 404) return { reached: true };
  if (!res.ok) throw new Error(`${path} answered HTTP ${res.status}`);
  return { reached: true, body: (await res.text()).trim() };
}

const deadline = Date.now() + waitMs;
let lastError;
let reached = false;

async function liveSha() {
  try {
    const res = await text("/deploy.txt", Math.max(1_000, Math.min(15_000, deadline - Date.now())));
    reached = true;
    return res.body;
  } catch (error) {
    lastError = error;
    return undefined;
  }
}

const changed = changedUrls();
if (changed !== "all" && !changed.length) {
  console.log("No public page changed; no ping.");
  process.exit(0);
}

if (sha) {
  let live = await liveSha();
  while (live !== sha && Date.now() < deadline) {
    await new Promise((resolve) => setTimeout(resolve, Math.min(20_000, Math.max(0, deadline - Date.now()))));
    live = await liveSha();
  }
  if (live !== sha) {
    if (!reached) {
      console.error(`${site.host} never served deploy.txt: ${lastError?.message}`);
      process.exit(1);
    }
    console.log(`${site.host} still serves ${live ?? "no deploy.txt"}, not ${sha}; no ping.`);
    process.exit(0);
  }
}

if ((await text(`/${keyFile}`)).body !== key) throw new Error(`${site.host}/${keyFile} does not serve the key`);

let urlList = changed;
if (urlList === "all") {
  const sitemap = (await text("/sitemap-0.xml")).body ?? "";
  urlList = [...sitemap.matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1]);
  if (!urlList.length) throw new Error("live sitemap lists no URLs");
}

const body = { host: site.host, key, keyLocation: new URL(`/${keyFile}`, site).href, urlList };
if (dryRun) {
  console.log(JSON.stringify(body, null, 2));
  process.exit(0);
}
const res = await fetch("https://api.indexnow.org/indexnow", {
  method: "POST",
  headers: { "Content-Type": "application/json; charset=utf-8" },
  body: JSON.stringify(body),
});
console.log(`IndexNow ${res.status} for ${urlList.length} URLs`);
if (res.status !== 200 && res.status !== 202) process.exit(1);
