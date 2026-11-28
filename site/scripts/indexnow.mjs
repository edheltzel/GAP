import { execFileSync } from "node:child_process";
import { readdirSync } from "node:fs";

const site = new URL(process.env.SITE_URL || "https://gap.edheltzel.codes");
const sha = process.env.DEPLOY_SHA;
const before = process.env.BEFORE_SHA;
const waitMs = Number(process.env.DEPLOY_WAIT_MS || 20 * 60 * 1000);
const dryRun = process.argv.includes("--dry-run");

const git = (...args) =>
  execFileSync("git", args, { cwd: new URL("../..", import.meta.url), encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
const isKeyFile = (name) => /^[0-9a-f]{32}\.txt$/.test(name);

/** The IndexNow key file in site/public/, on disk or at a commit. */
function keyFileAt(commit) {
  const names = commit
    ? git("ls-tree", "-z", "--name-only", `${commit}:site/public/`).split("\0")
    : readdirSync(new URL("../public/", import.meta.url));
  const name = names.find(isKeyFile);
  if (!name) throw new Error(`no IndexNow key file in site/public/${commit ? ` at ${commit}` : ""}`);
  return name;
}

/**
 * The published pages at a commit (site/src/repo-pages.json): repo file -> page
 * URL. Commits before that file existed kept the map in repo-links.ts.
 */
function pagesAt(commit) {
  let pages;
  try {
    pages = JSON.parse(git("show", `${commit}:site/src/repo-pages.json`));
  } catch {
    const source = git("show", `${commit}:site/src/lib/repo-links.ts`);
    const block = source.match(/repoPages[^{]*\{([^}]*)\}/)?.[1] ?? "";
    pages = Object.fromEntries([...block.matchAll(/["']([^"']+)["']:\s*["']([^"']+)["']/g)].map(([, file, id]) => [file, id]));
  }
  if (!Object.keys(pages).length) throw new Error(`no published pages found at ${commit}`);
  return new Map(Object.entries(pages).map(([file, id]) => [file, new URL(`/${id}/`, site).href]));
}

const builtFrom = (file) =>
  /^site\/(src|public)\//.test(file) ||
  /^site\/(astro\.config\.[cm]?[jt]s|package(-lock)?\.json|tsconfig\.json)$/.test(file) ||
  file.startsWith("docs/images/");

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
  for (const file of git("diff", "-z", "--name-only", "--no-renames", before, sha).split("\0").filter(Boolean)) {
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
  lastError = undefined;
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

let live;
let keyFile;
let key;
let keyError;
const servesKey = async () => {
  keyError = undefined;
  try {
    const left = sha ? deadline - Date.now() : 15_000;
    if (left <= 0) {
      keyError = new Error("wait deadline passed");
      return false;
    }
    const served = await text(`/${keyFile}`, Math.min(15_000, left));
    if (served === key) {
      if (!sha || Date.now() < deadline) return true;
      keyError = new Error("key arrived after the wait deadline");
      return false;
    }
    keyError = new Error(served === undefined ? "HTTP 404" : "wrong key");
  } catch (error) {
    keyError = error;
  }
  return false;
};

async function sitemapUrls() {
  const urls = new Set(changed.urls);
  if (changed.all) {
    const sitemap = (await text("/sitemap-0.xml")) ?? "";
    const listed = [...sitemap.matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1]);
    if (!listed.length) throw new Error("live sitemap lists no URLs");
    for (const url of listed) urls.add(url);
  }
  return [...urls];
}

let urlList;
let failure;
for (;;) {
  if (sha) live = await liveSha();
  const ours = !sha || includesOurs(live);
  if (!ours) {
    if (!reached) failure = `${site.host} never served deploy.txt: ${lastError?.message}`;
    else if (lastError) failure = `${site.host} deploy.txt failed: ${lastError.message}`;
    else failure = `${site.host} still serves ${live ?? "no deploy.txt"}, which does not contain ${sha}; no ping. Rerun this job once it does.`;
  }
  if (ours) {
    keyFile = keyFileAt(live);
    key = keyFile.slice(0, -4);
    if (await servesKey()) {
      let sitemapError;
      try {
        urlList = await sitemapUrls();
      } catch (error) {
        sitemapError = error;
      }
      const still = sha ? await liveSha() : live;
      if (still === live) {
        if (!sitemapError && (!sha || Date.now() < deadline)) break;
        failure = sitemapError
          ? `${site.host} sitemap: ${sitemapError.message}`
          : `${site.host} confirmed ${sha} only after the wait deadline; no ping. Rerun this job.`;
        if (!sha) {
          console.error(failure);
          process.exit(1);
        }
      } else if (still === undefined && lastError) {
        failure = `${site.host} deploy.txt failed while confirming ${sha}: ${lastError.message}`;
      } else {
        failure = `${site.host} moved to ${still ?? "no deploy.txt"} while checking the key; no ping. Rerun this job once it contains ${sha}.`;
        if (Date.now() < deadline) continue;
      }
    } else if (!(failure && keyError?.message === "wait deadline passed")) {
      failure = `${site.host}/${keyFile} does not serve the key: ${keyError?.message}`;
    }
  }
  const left = sha ? deadline - Date.now() : 0;
  if (left <= 0) {
    console.error(failure);
    process.exit(1);
  }
  await new Promise((resolve) => setTimeout(resolve, Math.min(20_000, left)));
  if (Date.now() >= deadline) {
    console.error(failure);
    process.exit(1);
  }
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
