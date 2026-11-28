import { readdirSync } from "node:fs";

const site = new URL(process.env.SITE_URL || "https://gap.edheltzel.codes");
const sha = process.env.DEPLOY_SHA;
const waitMs = Number(process.env.DEPLOY_WAIT_MS || 20 * 60 * 1000);
const dryRun = process.argv.includes("--dry-run");

const keyFile = readdirSync(new URL("../public/", import.meta.url)).find((name) => /^[0-9a-f]{32}\.txt$/.test(name));
if (!keyFile) throw new Error("no IndexNow key file in public/");
const key = keyFile.slice(0, -4);

async function text(path, timeoutMs = 15_000) {
  const res = await fetch(new URL(path, site), { cache: "no-store", signal: AbortSignal.timeout(timeoutMs) });
  return res.ok ? (await res.text()).trim() : undefined;
}

async function liveSha() {
  try {
    return await text("/deploy.txt", Math.max(1_000, Math.min(15_000, deadline - Date.now())));
  } catch (error) {
    lastError = error;
    return undefined;
  }
}

const deadline = Date.now() + waitMs;
let lastError;

if (sha) {
  let live = await liveSha();
  while (live !== sha && Date.now() < deadline) {
    await new Promise((resolve) => setTimeout(resolve, Math.min(20_000, Math.max(0, deadline - Date.now()))));
    live = await liveSha();
  }
  if (live !== sha) {
    if (live === undefined && lastError) {
      console.error(`${site.host} unreachable: ${lastError.message}`);
      process.exit(1);
    }
    console.log(`${site.host} still serves ${live ?? "no deploy.txt"}, not ${sha}; no ping.`);
    process.exit(0);
  }
}

if ((await text(`/${keyFile}`)) !== key) throw new Error(`${site.host}/${keyFile} does not serve the key`);

const sitemap = await text("/sitemap-0.xml");
const urlList = [...(sitemap ?? "").matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1]);
if (!urlList.length) throw new Error("live sitemap lists no URLs");

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
