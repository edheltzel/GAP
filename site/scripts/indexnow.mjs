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

const notBuilt = new Set(["site/AGENTS.md", "site/README.md"]);
const builtFrom = (file) =>
  (file.startsWith("site/") && !file.startsWith("site/scripts/") && !notBuilt.has(file)) || file.startsWith("docs/images/");

/**
 * The URLs a deploy changed. A root page maps to its own URL; a page dropped
 * from repoPages keeps its old URL so IndexNow hears of the removal; a change to
 * site code or images adds every live sitemap URL (`all`).
 */
function changedUrls() {
  if (!sha || !before || /^0+$/.test(before)) return { urls: [], all: true };
  const now = pagesAt(sha);
  const then = pagesAt(before);
  const urls = new Set();
  let all = false;
  for (const [file, url] of then) if (now.get(file) !== url) urls.add(url);
  for (const file of git("diff", "--name-only", "--no-renames", before, sha).split("\n").filter(Boolean)) {
    if (now.has(file)) urls.add(now.get(file));
    else if (then.has(file)) urls.add(then.get(file));
    else if (builtFrom(file)) all = true;
  }
  return { urls: [...urls], all };
}

/** Whether the live commit is ours or a later one that contains it. */
function includesOurs(live) {
  if (!live || !/^[0-9a-f]{40}$/.test(live)) return false;
  if (live === sha) return true;
  try {
    git("cat-file", "-e", `${live}^{commit}`);
  } catch {
    try {
      git("fetch", "--quiet", "origin", live);
    } catch {
      return false;
    }
  }
  try {
    git("merge-base", "--is-ancestor", sha, live);
    return true;
  } catch {
    return false;
  }
}

async function text(path, timeoutMs = 15_000, onAnswer = () => {}) {
  const res = await fetch(new URL(path, site), { cache: "no-store", signal: AbortSignal.timeout(timeoutMs) });
  if (res.status !== 200 && res.status !== 404) throw new Error(`${path} answered HTTP ${res.status}`);
  onAnswer();
  return res.status === 200 ? (await res.text()).trim() : undefined;
}

const deadline = Date.now() + waitMs;
let lastError;
let reached = false;

async function liveSha() {
  try {
    return await text("/deploy.txt", Math.max(1_000, Math.min(15_000, deadline - Date.now())), () => {
      reached = true;
    });
  } catch (error) {
    lastError = error;
    return undefined;
  }
}

const changed = changedUrls();
if (!changed.all && !changed.urls.length) {
  console.log("No public page changed; no ping.");
  process.exit(0);
}

if (sha) {
  let live = await liveSha();
  while (!includesOurs(live) && Date.now() < deadline) {
    await new Promise((resolve) => setTimeout(resolve, Math.min(20_000, Math.max(0, deadline - Date.now()))));
    live = await liveSha();
  }
  if (!includesOurs(live)) {
    if (!reached) {
      console.error(`${site.host} never served deploy.txt: ${lastError?.message}`);
      process.exit(1);
    }
    console.log(`${site.host} still serves ${live ?? "no deploy.txt"}, not ${sha}; no ping.`);
    process.exit(0);
  }
}

if ((await text(`/${keyFile}`)) !== key) throw new Error(`${site.host}/${keyFile} does not serve the key`);

const urls = new Set(changed.urls);
if (changed.all) {
  const sitemap = (await text("/sitemap-0.xml")) ?? "";
  const live = [...sitemap.matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1]);
  if (!live.length) throw new Error("live sitemap lists no URLs");
  for (const url of live) urls.add(url);
}
const urlList = [...urls];

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
