#!/usr/bin/env bun
// Expand a literal $HOME a harness may inject into PROJECTS_DIR.
for (const __k of ["PROJECTS_DIR"]) {
  const __v = process.env[__k];
  if (__v && /^\$\{?HOME\}?(\/|$)/.test(__v)) process.env[__k] = __v.replace(/^\$\{?HOME\}?/, process.env.HOME ?? "~");
}


/**
 * generate - image generation CLI
 *
 * Generate images with xAI Grok Imagine, OpenAI gpt-image-2, or Google
 * Gemini (Nano Banana, Nano Banana Pro). Picks a provider that has an API key,
 * falls back to another one when the requested provider has none, and stops
 * with setup instructions when no provider has a key.
 *
 * Usage:
 *   bun Generate.ts --workflow=<name> --model nano-banana-pro --prompt "..." --aspect-ratio 16:9
 *   bun Generate.ts --check-keys
 *
 * @see <skill-dir>/SKILL.md
 */

import OpenAI from "openai";
import { GoogleGenAI } from "@google/genai";
import { accessSync, constants, existsSync, mkdirSync } from "node:fs";
import { writeFile, readFile } from "node:fs/promises";
import { homedir } from "node:os";
import { delimiter, extname, join, resolve } from "node:path";

// ============================================================================
// Environment Loading
// ============================================================================

/**
 * Optional env files. Process environment wins. Never required.
 * Order: ./.env, then ${XDG_CONFIG_HOME:-~/.config}/do-art/.env
 */
async function loadEnv(): Promise<void> {
  const xdg = process.env.XDG_CONFIG_HOME || join(homedir(), ".config");
  const candidates = [resolve(".env"), join(xdg, "do-art", ".env")];
  for (const envPath of candidates) {
    let envContent: string;
    try {
      envContent = await readFile(envPath, "utf-8");
    } catch {
      continue;
    }
    for (const line of envContent.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#")) continue;
      const eqIndex = trimmed.indexOf("=");
      if (eqIndex === -1) continue;
      const key = trimmed.slice(0, eqIndex).trim();
      let value = trimmed.slice(eqIndex + 1).trim();
      if ((value.startsWith('"') && value.endsWith('"')) ||
          (value.startsWith("'") && value.endsWith("'"))) {
        value = value.slice(1, -1);
      }
      if (!process.env[key]) process.env[key] = value;
    }
  }

  // Canonical key aliases — the user's .env may use _OPTIN suffix variants for some
  // providers (data-usage opt-in keys). Tools that look up the bare name
  // must transparently get the OPTIN value when no bare key is set.
  // Add new aliases here when a provider has a suffix variant in the env.
  const aliases: Record<string, string> = {
    OPENAI_API_KEY: "OPENAI_API_KEY_OPTIN",
  };
  for (const [bare, suffixed] of Object.entries(aliases)) {
    if (!process.env[bare] && process.env[suffixed]) {
      process.env[bare] = process.env[suffixed];
    }
  }
}

// ============================================================================
// Providers, Models, and API Keys
// ============================================================================

type Provider = "xai" | "openai" | "google" | "openrouter";
type ImageModel = "grok" | "gpt-image-2" | "nano-banana" | "nano-banana-pro";
type Model = ImageModel | "auto" | "compare";
type Route = "direct" | "openrouter";
type AspectRatio = "1:1" | "16:9" | "3:2" | "2:3" | "3:4" | "4:3" | "4:5" | "5:4" | "9:16" | "21:9";
type OpenAISize2 = "1024x1024" | "1536x1024" | "1024x1536" | "2048x2048" | "auto";
type Resolution = "1K" | "2K" | "4K";
type Quality = "low" | "medium" | "high" | "auto";

const PROVIDERS: Record<Provider, { label: string; envVars: string[]; keyUrl: string }> = {
  xai: { label: "xAI (Grok Imagine)", envVars: ["XAI_API_KEY"], keyUrl: "https://console.x.ai" },
  openai: { label: "OpenAI (ChatGPT images)", envVars: ["OPENAI_API_KEY", "OPENAI_API_KEY_OPTIN"], keyUrl: "https://platform.openai.com/api-keys" },
  google: { label: "Google Gemini (Nano Banana)", envVars: ["GEMINI_API_KEY", "GOOGLE_API_KEY"], keyUrl: "https://aistudio.google.com/apikey" },
  openrouter: { label: "OpenRouter (all four models)", envVars: ["OPENROUTER_API_KEY"], keyUrl: "https://openrouter.ai/keys" },
};

// openRouterModel: the same model's slug on OpenRouter's /images API, used when the direct key is missing.
const MODELS: Record<ImageModel, { provider: Exclude<Provider, "openrouter">; apiModel: string; openRouterModel: string; label: string }> = {
  "grok": { provider: "xai", apiModel: "grok-imagine-image-2.0", openRouterModel: "x-ai/grok-imagine-image-2.0", label: "Grok Imagine" },
  "gpt-image-2": { provider: "openai", apiModel: "gpt-image-2", openRouterModel: "openai/gpt-image-2", label: "gpt-image-2" },
  "nano-banana": { provider: "google", apiModel: "gemini-nano-banana-2.1", openRouterModel: "google/gemini-nano-banana-2.1", label: "Nano Banana 2.1" },
  "nano-banana-pro": { provider: "google", apiModel: "gemini-3-pro-image", openRouterModel: "google/gemini-3-pro-image", label: "Nano Banana Pro" },
};

// Order tried by --model auto and by fallback when the requested model has no route.
const FALLBACK_ORDER: ImageModel[] = ["grok", "gpt-image-2", "nano-banana-pro"];
const IMAGE_MODELS = Object.keys(MODELS) as ImageModel[];

function apiKey(provider: Provider): string | undefined {
  for (const name of PROVIDERS[provider].envVars) {
    const value = process.env[name]?.trim();
    if (value) return value;
  }
  return undefined;
}

/** How a model can run: its own provider's key first, then OpenRouter, else not at all. */
function routeFor(model: ImageModel): Route | undefined {
  if (apiKey(MODELS[model].provider)) return "direct";
  if (apiKey("openrouter")) return "openrouter";
  return undefined;
}

function keySetupHelp(): string {
  const lines = Object.values(PROVIDERS).map(
    (p) => `  ${p.label.padEnd(30)} ${p.envVars.join(" or ").padEnd(40)} ${p.keyUrl}`
  );
  return [
    "Set at least one of these API keys (OPENROUTER_API_KEY alone covers every model):",
    ...lines,
    "Keys load from the process environment, then ./.env, then ${XDG_CONFIG_HOME:-~/.config}/do-art/.env",
  ].join("\n");
}

/**
 * Pick the model and route that will actually run. A requested model without
 * its own key runs through OpenRouter when that key is set; otherwise it falls
 * back, with a notice, to the first routable model in FALLBACK_ORDER.
 * Reference images go to Gemini models only.
 */
function resolveModel(requested: ImageModel | "auto", needsGemini: boolean): { model: ImageModel; route: Route } {
  if (requested !== "auto") {
    const route = routeFor(requested);
    if (route) {
      if (route === "openrouter") {
        console.warn(`⚠️ ${PROVIDERS[MODELS[requested].provider].envVars.join(" or ")} is not set; running ${requested} through OpenRouter.`);
      }
      return { model: requested, route };
    }
  }

  const candidates = needsGemini ? (["nano-banana-pro", "nano-banana"] as ImageModel[]) : FALLBACK_ORDER;
  const fallback = candidates.find((m) => routeFor(m));

  if (!fallback) {
    const why = needsGemini
      ? "--reference-image needs a Gemini model, and neither a Gemini key nor OPENROUTER_API_KEY is set."
      : requested === "auto"
        ? "No image provider has an API key."
        : `${MODELS[requested].label} needs ${PROVIDERS[MODELS[requested].provider].envVars.join(" or ")} or OPENROUTER_API_KEY, and no other provider has a key to fall back to.`;
    throw new MissingKeyError(`${why}\n\n${keySetupHelp()}`);
  }

  const route = routeFor(fallback)!;
  if (requested !== "auto") {
    const missing = PROVIDERS[MODELS[requested].provider].envVars.join(" or ");
    console.warn(`⚠️ ${missing} is not set, so ${requested} cannot run. Falling back to ${fallback} (${route === "openrouter" ? "via OpenRouter" : PROVIDERS[MODELS[fallback].provider].label}).`);
  }
  return { model: fallback, route };
}

function printKeyStatus(): void {
  for (const p of Object.values(PROVIDERS)) {
    const found = p.envVars.find((name) => process.env[name]?.trim());
    console.log(`${found ? "✅" : "❌"} ${p.label.padEnd(30)} ${found ? `${found} set` : `missing ${p.envVars.join(" or ")}`}`);
  }
  console.log("");
  for (const m of IMAGE_MODELS) {
    const route = routeFor(m);
    console.log(`  ${m.padEnd(16)} ${route === "direct" ? "direct" : route === "openrouter" ? "via OpenRouter" : "unavailable"}`);
  }
  const auto = FALLBACK_ORDER.find((m) => routeFor(m));
  if (!auto) {
    console.log(`\n${keySetupHelp()}`);
    process.exit(2);
  }
  console.log(`\n--model auto would use: ${auto}`);
}

interface CLIArgs {
  model: Model;
  prompt: string;
  aspectRatio: AspectRatio;
  resolution: Resolution;
  openaiSize?: OpenAISize2; // explicit gpt-image-2 pixel size from --size
  output: string;
  creativeVariations?: number;
  quality?: Quality; // gpt-image-2 only
  transparent?: boolean; // Enable transparent background
  referenceImages?: string[]; // Reference image paths (Gemini only) - up to 14 total
  removeBg?: boolean; // Remove background after generation using local rembg
  addBg?: string; // Add background color (hex) to transparent image
  thumbnail?: boolean; // Generate additional thumbnail with #EAE9DF background for social previews
  workflow?: string; // Name of the Art workflow that constructed this call (REQUIRED unless freeformConfirmed)
  freeformConfirmed?: boolean; // Explicit opt-out of workflow discipline — logged to stderr
  signature?: boolean; // true forces a log if ART_SIGNATURE is unset; false skips the stamp
}

// ============================================================================
// Configuration
// ============================================================================

function defaultModel(): Model {
  const fromEnv = process.env.ART_MODEL?.trim();
  if (!fromEnv) return "auto";
  if (fromEnv === "auto" || fromEnv === "compare" || (IMAGE_MODELS as string[]).includes(fromEnv)) return fromEnv as Model;
  throw new CLIError(`Invalid ART_MODEL: ${fromEnv}. Must be: auto, compare, ${IMAGE_MODELS.join(", ")}`);
}

function previewDir(): string {
  const fromEnv = process.env.ART_OUTPUT_DIR;
  if (fromEnv) {
    mkdirSync(fromEnv, { recursive: true });
    return fromEnv;
  }
  const home = process.env.HOME;
  if (home) {
    const downloads = join(home, "Downloads");
    if (existsSync(downloads)) return downloads;
  }
  const local = resolve("art-output");
  mkdirSync(local, { recursive: true });
  return local;
}

const ASPECT_RATIOS: AspectRatio[] = ["1:1", "16:9", "3:2", "2:3", "3:4", "4:3", "4:5", "5:4", "9:16", "21:9"];
// Grok Imagine does not offer 4:5 or 5:4; those fall back to its "auto" ratio.
const GROK_ASPECT_RATIOS: AspectRatio[] = ["1:1", "16:9", "3:2", "2:3", "3:4", "4:3", "9:16", "21:9"];
const OPENAI_V2_SIZES: OpenAISize2[] = ["1024x1024", "1536x1024", "1024x1536", "2048x2048", "auto"];
const RESOLUTIONS: Resolution[] = ["1K", "2K", "4K"];
const QUALITY_VALUES: Quality[] = ["low", "medium", "high", "auto"];

/** gpt-image-2 takes pixel sizes; derive one from the shared aspect ratio and resolution. */
function openaiSizeFor(aspect: AspectRatio, resolution: Resolution): OpenAISize2 {
  const [w, h] = aspect.split(":").map(Number) as [number, number];
  if (w === h) return resolution === "1K" ? "1024x1024" : "2048x2048";
  return w > h ? "1536x1024" : "1024x1536";
}

// ============================================================================
// Error Handling
// ============================================================================

class CLIError extends Error {
  constructor(message: string, public exitCode: number = 1) {
    super(message);
    this.name = "CLIError";
  }
}

/** Exit code 2: no provider can run this request because no usable API key is set. */
class MissingKeyError extends CLIError {
  constructor(message: string) {
    super(message, 2);
    this.name = "MissingKeyError";
  }
}

function handleError(error: unknown): never {
  if (error instanceof CLIError) {
    console.error(`❌ Error: ${error.message}`);
    process.exit(error.exitCode);
  }

  if (error instanceof Error) {
    console.error(`❌ Unexpected error: ${error.message}`);
    console.error(error.stack);
    process.exit(1);
  }

  console.error(`❌ Unknown error:`, error);
  process.exit(1);
}

// ============================================================================
// Image Format Detection
// ============================================================================

/**
 * Detect actual image format from magic bytes.
 * Prevents MIME type mismatch when API returns different format than requested.
 */
function detectImageFormat(data: Buffer | Uint8Array): { format: string; ext: string; mime: string } | null {
  if (data.length < 12) return null;
  if (data[0] === 0x89 && data[1] === 0x50 && data[2] === 0x4e && data[3] === 0x47)
    return { format: "png", ext: ".png", mime: "image/png" };
  if (data[0] === 0xff && data[1] === 0xd8 && data[2] === 0xff)
    return { format: "jpeg", ext: ".jpg", mime: "image/jpeg" };
  if (data[0] === 0x52 && data[1] === 0x49 && data[2] === 0x46 && data[3] === 0x46 &&
      data[8] === 0x57 && data[9] === 0x45 && data[10] === 0x42 && data[11] === 0x50)
    return { format: "webp", ext: ".webp", mime: "image/webp" };
  if (data[0] === 0x47 && data[1] === 0x49 && data[2] === 0x46)
    return { format: "gif", ext: ".gif", mime: "image/gif" };
  return null;
}

/**
 * Save image data with correct file extension based on actual content format.
 * Returns the final path (may differ from requested if format mismatch detected).
 */
async function saveImage(data: Buffer | Uint8Array | any, requestedPath: string): Promise<string> {
  const buffer = data instanceof Buffer ? data : Buffer.from(data as any);
  const detected = detectImageFormat(buffer);
  if (detected) {
    const requestedExt = extname(requestedPath).toLowerCase();
    if (requestedExt && requestedExt !== detected.ext) {
      const correctedPath = requestedPath.replace(/\.[^.]+$/, detected.ext);
      console.warn(`⚠️ API returned ${detected.format.toUpperCase()} data (requested ${requestedExt.slice(1).toUpperCase()}). Saving as ${correctedPath}`);
      await writeFile(correctedPath, buffer);
      return correctedPath;
    }
  }
  await writeFile(requestedPath, buffer);
  return requestedPath;
}

/**
 * Detect MIME type from image file content (magic bytes), falling back to extension.
 */
async function detectMimeType(filePath: string): Promise<string> {
  try {
    const data = await readFile(filePath);
    const detected = detectImageFormat(data);
    if (detected) return detected.mime;
  } catch {
    // Fall through to extension-based detection
  }
  const ext = extname(filePath).toLowerCase();
  switch (ext) {
    case ".png": return "image/png";
    case ".jpg": case ".jpeg": return "image/jpeg";
    case ".webp": return "image/webp";
    default: throw new CLIError(`Unsupported image format: ${ext}. Supported: .png, .jpg, .jpeg, .webp`);
  }
}

// ============================================================================
// Help Text
// ============================================================================


function showHelp(): void {
  console.log(`
generate - image generation CLI

Generate images with xAI Grok Imagine, OpenAI gpt-image-2, or Google Gemini
(Nano Banana, Nano Banana Pro). Uses whichever provider has an API key; when the
requested provider has none, falls back to another and says so.

USAGE:
  generate --workflow=<name> --prompt "<prompt>" [--model <model>] [OPTIONS]
  generate --check-keys        Show which providers have keys, then exit

REQUIRED:
  --prompt <text>      Image generation prompt (quote if contains spaces)
  --workflow=<name>    The Art workflow that built this call (or --freeform-confirmed)

MODELS (--model, default: auto, or ART_MODEL):
  auto                 First model that can run, in this order: grok, gpt-image-2, nano-banana-pro
  grok                 xAI grok-imagine-image-2.0                     needs XAI_API_KEY
  gpt-image-2          OpenAI gpt-image-2                             needs OPENAI_API_KEY
  nano-banana          Google gemini-nano-banana-2.1 (fast, cheap)    needs GEMINI_API_KEY or GOOGLE_API_KEY
  nano-banana-pro      Google gemini-3-pro-image (best text, refs)    needs GEMINI_API_KEY or GOOGLE_API_KEY
  compare              Same prompt on every model that can run, side by side

  OPENROUTER_API_KEY runs any of the four models through OpenRouter when that
  model's own key is missing. One OpenRouter key covers everything.

OPTIONS:
  --aspect-ratio <ratio>     1:1, 16:9, 3:2, 2:3, 3:4, 4:3, 4:5, 5:4, 9:16, 21:9 (default 16:9)
                             Grok has no 4:5 or 5:4 and uses its own "auto" ratio for those
  --size <size>              Resolution 1K, 2K, 4K (default 2K), an aspect ratio, or an exact
                             gpt-image-2 size (1024x1024, 1536x1024, 1024x1536, 2048x2048, auto).
                             Grok tops out at 2k; gpt-image-2 maps resolution to its nearest size.
  --quality <level>          gpt-image-2 only: low, medium, high, auto (default: high)
  --output <path>            Output file (default: <preview>/art.png)
  --reference-image <path>   Style/character reference (Gemini models only; repeatable, max 14)
  --transparent              Add transparency instructions to the prompt
  --remove-bg                Remove the background with local rembg (true alpha)
  --add-bg <hex>             Flatten a transparent image onto a color (e.g. "#EAE9DF")
  --thumbnail                Write output.png (transparent) + output-thumb.png (#EAE9DF); implies --remove-bg
  --no-signature             Skip the ART_SIGNATURE stamp
  --signature                Stamp ART_SIGNATURE if set. No default text when unset
  --creative-variations <n>  Generate N images from the same prompt (-v1, -v2, ...)
  --freeform-confirmed       Skip the workflow requirement (logged)
  --help, -h                 Show this help message

EXAMPLES:
  generate --workflow=Essay --model nano-banana-pro --prompt "Abstract editorial illustration..." --size 2K --aspect-ratio 16:9
  generate --workflow=Stats --model gpt-image-2 --prompt "Stat card with crisp serif numerals..." --size 1024x1536 --quality high
  generate --workflow=Comics --model grok --prompt "Three-panel ink comic..." --aspect-ratio 3:2
  generate --workflow=Essay --model compare --prompt "..." --creative-variations 2 --output <preview>/shootout.png
  generate --workflow=Essay --model nano-banana-pro --prompt "Person from references at a party..." \\
    --reference-image face1.jpg --reference-image face2.jpg --size 2K --aspect-ratio 16:9

ENVIRONMENT VARIABLES:
  XAI_API_KEY                     xAI key for grok (https://console.x.ai)
  OPENAI_API_KEY                  OpenAI key for gpt-image-2 (OPENAI_API_KEY_OPTIN also accepted)
  GEMINI_API_KEY / GOOGLE_API_KEY Google key for nano-banana and nano-banana-pro (https://aistudio.google.com/apikey)
  OPENROUTER_API_KEY              Runs any model via OpenRouter when its own key is missing (https://openrouter.ai/keys)
  ART_MODEL            Default --model (auto when unset)
  REMBG_BIN            Optional rembg path. Else PATH, else ~/.local/bin/rembg
  ART_OUTPUT_DIR       Preview folder. Else ~/Downloads if it exists, else ./art-output
  ART_SIGNATURE        Optional signature text. Unset means no stamp
  ART_SIGNATURE_FONT   Optional ImageMagick font for the stamp
  Keys load from the process environment, then ./.env, then \${XDG_CONFIG_HOME:-~/.config}/do-art/.env

EXIT CODES:
  0  Success
  1  General error (invalid arguments, API error, file write error)
  2  No usable API key: nothing ran. The message lists which keys to set.

MORE INFO:
  Documentation: <skill-dir>/SKILL.md
  Source: <skill-dir>/Tools/Generate.ts
`);
  process.exit(0);
}

// ============================================================================
// Workflow Discipline Gate
// ============================================================================

/**
 * Enforce that callers either name the Art workflow that produced this
 * invocation OR explicitly opt out via --freeform-confirmed.
 *
 * Background: the Art skill's "ALWAYS RUN A NAMED WORKFLOW" doctrine used
 * to live in markdown only and was silently ignored — see ISA
 * 20260430-180000_art-skill-freeform-enforcement. This gate is the code
 * substitute for that doctrine.
 *
 * Behavior:
 *   --workflow=<name>       → validate <name>.md exists, allow.
 *   --workflow <name>       → same.
 *   --freeform-confirmed    → log explicit opt-out to stderr, allow.
 *   neither                 → exit 1 with the workflow lookup table.
 *   --workflow=<bad-name>   → exit 1 listing valid workflow names.
 */
function enforceWorkflowDiscipline(parsed: Partial<CLIArgs>): void {
  const workflowsDir = join(import.meta.dir, "..", "Workflows");
  let availableWorkflows: string[] = [];
  try {
    // readdirSync via Bun.readdirSync isn't a thing; use Node fs sync via dynamic
    // require to avoid adding a top-level import that would pull in fs at load.
    const fs = require("node:fs") as typeof import("node:fs");
    availableWorkflows = fs
      .readdirSync(workflowsDir)
      .filter((f: string) => f.endsWith(".md"))
      .map((f: string) => f.replace(/\.md$/, ""))
      .sort();
  } catch (err) {
    // If the workflows dir is gone, the skill is broken in a bigger way;
    // don't try to fix that here, but don't block on it either.
    process.stderr.write(
      `[Generate] WARNING: workflows dir not readable at ${workflowsDir} — skipping workflow validation\n`
    );
    return;
  }

  if (parsed.freeformConfirmed) {
    process.stderr.write(
      `[Generate] FREEFORM mode confirmed by caller (--freeform-confirmed). ` +
        `This is logged for audit. The Art skill's named workflows ` +
        `(${availableWorkflows.join(", ")}) are the recommended path; ` +
        `freeform output quality has historically been rejected.\n`
    );
    return;
  }

  if (!parsed.workflow) {
    const lines = [
      "",
      "═══════════════════════════════════════════════════════════════════════════",
      "  Generate.ts REFUSED — caller did not name a workflow.",
      "═══════════════════════════════════════════════════════════════════════════",
      "",
      "  The Art skill requires every image generation to run through a named",
      "  workflow. Freeform prompts have a documented rejection rate of ~100%.",
      "",
      "  Pass ONE of:",
      "    --workflow=<name>          (recommended)",
      "    --freeform-confirmed       (explicit opt-out, logged for audit)",
      "",
      "  Available workflows:",
      ...availableWorkflows.map(
        (w) => `    --workflow=${w.padEnd(28)} → ${workflowsDir}/${w}.md`
      ),
      "",
      "  Read the matching workflow file FIRST. Each one encodes the",
      "  prompt template, palette, composition rules, and validation gate",
      "  that the bare model fails to honor without them.",
      "",
      "═══════════════════════════════════════════════════════════════════════════",
      "",
    ];
    process.stderr.write(lines.join("\n"));
    process.exit(1);
  }

  // Validate the named workflow actually exists on disk.
  if (!availableWorkflows.includes(parsed.workflow)) {
    const lines = [
      "",
      `[Generate] REFUSED — workflow "${parsed.workflow}" not found.`,
      "",
      "Available workflows:",
      ...availableWorkflows.map((w) => `  --workflow=${w}`),
      "",
    ];
    process.stderr.write(lines.join("\n"));
    process.exit(1);
  }
}

// ============================================================================
// Argument Parsing
// ============================================================================

function parseArgs(argv: string[]): CLIArgs {
  const args = argv.slice(2);

  // Check for help flag
  if (args.includes("--help") || args.includes("-h") || args.length === 0) {
    showHelp();
  }

  const parsed: Partial<CLIArgs> = {
    model: defaultModel(),
    output: join(previewDir(), "art.png"),
  };
  let sizeArg: string | undefined;

  // Collect reference images into array
  const referenceImages: string[] = [];

  // Parse arguments
  for (let i = 0; i < args.length; i++) {
    const flag = args[i];

    if (!flag || !flag.startsWith("--")) {
      throw new CLIError(`Invalid flag: ${flag}. Flags must start with --`);
    }

    const key = flag.slice(2);

    // Handle boolean flags (no value)
    if (key === "transparent") {
      parsed.transparent = true;
      continue;
    }
    if (key === "remove-bg") {
      parsed.removeBg = true;
      continue;
    }
    if (key === "thumbnail") {
      parsed.thumbnail = true;
      parsed.removeBg = true; // Thumbnail mode requires remove-bg
      continue;
    }
    if (key === "signature") {
      parsed.signature = true;
      continue;
    }
    if (key === "no-signature") {
      parsed.signature = false;
      continue;
    }
    if (key === "freeform-confirmed") {
      parsed.freeformConfirmed = true;
      continue;
    }

    // --workflow=<name> (one-token form) — split here so it doesn't fall into the
    // value-required branch below.
    if (key.startsWith("workflow=")) {
      parsed.workflow = key.slice("workflow=".length);
      continue;
    }

    // Handle flags with values
    const value = args[i + 1];
    if (!value || value.startsWith("--")) {
      throw new CLIError(`Missing value for flag: ${flag}`);
    }

    switch (key) {
      case "model":
        if (value !== "auto" && value !== "compare" && !(IMAGE_MODELS as string[]).includes(value)) {
          if (value === "gpt-image-1") {
            throw new CLIError("gpt-image-1 is deprecated. Use --model gpt-image-2.");
          }
          if (value === "flux" || value === "midjourney") {
            throw new CLIError(`${value} was removed. Use grok, gpt-image-2, nano-banana, or nano-banana-pro.`);
          }
          throw new CLIError(`Invalid model: ${value}. Must be: auto, compare, ${IMAGE_MODELS.join(", ")}`);
        }
        parsed.model = value as Model;
        i++;
        break;
      case "quality":
        if (!QUALITY_VALUES.includes(value as Quality)) {
          throw new CLIError(`Invalid quality: ${value}. Must be: ${QUALITY_VALUES.join(", ")}`);
        }
        parsed.quality = value as Quality;
        i++;
        break;
      case "prompt":
        parsed.prompt = value;
        i++; // Skip next arg (value)
        break;
      case "size":
        sizeArg = value;
        i++;
        break;
      case "aspect-ratio":
        if (!ASPECT_RATIOS.includes(value as AspectRatio)) {
          throw new CLIError(`Invalid aspect-ratio: ${value}. Must be: ${ASPECT_RATIOS.join(", ")}`);
        }
        parsed.aspectRatio = value as AspectRatio;
        i++;
        break;
      case "output":
        parsed.output = value;
        i++; // Skip next arg (value)
        break;
      case "reference-image":
        // Collect multiple reference images into array
        referenceImages.push(value);
        i++; // Skip next arg (value)
        break;
      case "creative-variations":
        const variations = parseInt(value, 10);
        if (isNaN(variations) || variations < 1 || variations > 10) {
          throw new CLIError(`Invalid creative-variations: ${value}. Must be 1-10`);
        }
        parsed.creativeVariations = variations;
        i++; // Skip next arg (value)
        break;
      case "add-bg":
        // Validate hex color format
        if (!/^#[0-9A-Fa-f]{6}$/.test(value)) {
          throw new CLIError(`Invalid hex color: ${value}. Must be in format #RRGGBB (e.g., #EAE9DF)`);
        }
        parsed.addBg = value;
        i++; // Skip next arg (value)
        break;
      case "workflow":
        // --workflow <name> two-token form (the --workflow=<name> one-token
        // form is handled earlier, before this value-required branch)
        parsed.workflow = value;
        i++;
        break;
      default:
        throw new CLIError(`Unknown flag: ${flag}`);
    }
  }

  // Assign collected reference images if any
  if (referenceImages.length > 0) {
    parsed.referenceImages = referenceImages;
  }

  // Validate required arguments
  if (!parsed.prompt) {
    throw new CLIError("Missing required argument: --prompt");
  }

  if (!parsed.model) {
    throw new CLIError("Missing required argument: --model");
  }

  // --size is overloaded for backward compatibility: a resolution tier, an
  // aspect ratio, or an exact gpt-image-2 pixel size.
  if (sizeArg !== undefined) {
    const upper = sizeArg.toUpperCase();
    if (RESOLUTIONS.includes(upper as Resolution)) parsed.resolution = upper as Resolution;
    else if (OPENAI_V2_SIZES.includes(sizeArg as OpenAISize2)) parsed.openaiSize = sizeArg as OpenAISize2;
    else if (ASPECT_RATIOS.includes(sizeArg as AspectRatio)) parsed.aspectRatio ??= sizeArg as AspectRatio;
    else {
      throw new CLIError(
        `Invalid size: ${sizeArg}. Use a resolution (${RESOLUTIONS.join(", ")}), an aspect ratio (${ASPECT_RATIOS.join(", ")}), or a gpt-image-2 size (${OPENAI_V2_SIZES.join(", ")})`
      );
    }
  }
  parsed.aspectRatio ??= "16:9";
  parsed.resolution ??= "2K";

  // ──────────────────────────────────────────────────────────────────────
  // WORKFLOW DISCIPLINE GATE (the load-bearing line — see ISA
  // 20260430-180000_art-skill-freeform-enforcement)
  //
  // The Art skill's "ALWAYS RUN A NAMED WORKFLOW" doctrine used to live
  // in markdown only and was silently ignored. This gate moves it into
  // code: callers must either name the workflow that produced this call
  // OR explicitly opt out with --freeform-confirmed.
  // ──────────────────────────────────────────────────────────────────────
  enforceWorkflowDiscipline(parsed);

  const refs = parsed.referenceImages?.length ?? 0;
  if (refs > 0 && (parsed.model === "grok" || parsed.model === "gpt-image-2" || parsed.model === "compare")) {
    throw new CLIError("--reference-image only works with the Gemini models: nano-banana, nano-banana-pro, or auto");
  }

  // Validate reference image count (API limits: 5 human, 6 object, 14 total max)
  if (refs > 14) {
    throw new CLIError(`Too many reference images: ${refs}. Maximum is 14 total (5 human, 6 object)`);
  }

  return parsed as CLIArgs;
}

// ============================================================================
// Prompt Enhancement
// ============================================================================

function enhancePromptForTransparency(prompt: string): string {
  const transparencyPrefix = "CRITICAL: Transparent background (PNG with alpha channel) - NO background color, pure transparency. Object floating in transparent space. ";
  return transparencyPrefix + prompt;
}

// ============================================================================
// Background Removal
// ============================================================================

import { execFile } from "node:child_process";
import { promisify } from "node:util";

// Expand a literal $HOME a harness may leave in PROJECTS_DIR
for (const k of ["PROJECTS_DIR"]) {
  const v = process.env[k];
  if (v && /^\$\{?HOME\}?(\/|$)/.test(v)) process.env[k] = v.replace(/^\$\{?HOME\}?/, process.env.HOME ?? "~");
}


// Argument arrays, never a shell string: paths and env values reach ImageMagick verbatim.
const execFileAsync = promisify(execFile);

// ============================================================================
// Background Operations
// ============================================================================

/**
 * Add a solid background color to a transparent PNG image
 * Uses ImageMagick to composite the transparent image onto a colored background
 */
async function addBackgroundColor(inputPath: string, outputPath: string, hexColor: string): Promise<void> {
  console.log(`🎨 Adding background color ${hexColor} to image...`);

  // Use ImageMagick to composite the transparent image onto a colored background
  // -background sets the fill color, -flatten composites onto that background
  try {
    await execFileAsync("magick", [inputPath, "-background", hexColor, "-flatten", outputPath]);
    console.log(`✅ Thumbnail saved to ${outputPath}`);
  } catch (error) {
    throw new CLIError(`Failed to add background color: ${error instanceof Error ? error.message : String(error)}`);
  }
}

/** Stamp ART_SIGNATURE bottom-right. Caller must pass a non-empty name. */
async function stampSignature(imagePath: string, name: string): Promise<void> {
  let pointsize = 31;
  try {
    const { stdout } = await execFileAsync("identify", ["-format", "%w", imagePath]);
    const width = parseInt(stdout.trim(), 10);
    if (Number.isFinite(width) && width > 0) {
      pointsize = Math.max(20, Math.round(width * 0.03));
    }
  } catch {
    // fall back to 31pt
  }

  const font = process.env.ART_SIGNATURE_FONT;
  // ImageMagick reads `-annotate @file` as a file path; escape a leading @ so the text stays literal.
  const text = name.startsWith("@") ? `\\${name}` : name;
  const args = [
    imagePath,
    "-gravity", "SouthEast",
    ...(font ? ["-font", font] : []),
    "-pointsize", String(pointsize),
    "-fill", "rgba(55,45,38,0.55)",
    "-annotate", "352x352+44+30", text,
    imagePath,
  ];

  try {
    await execFileAsync("magick", args);
    console.log("Signature stamped");
  } catch (error) {
    throw new CLIError(
      `Failed to stamp signature: ${error instanceof Error ? error.message : String(error)}.`
    );
  }
}

function resolveRembgBin(): string {
  if (process.env.REMBG_BIN) return resolve(process.env.REMBG_BIN);
  for (const dir of (process.env.PATH || "").split(delimiter)) {
    if (!dir) continue;
    const candidate = resolve(dir, "rembg");
    try {
      accessSync(candidate, constants.X_OK);
      return candidate;
    } catch {
      // keep looking
    }
  }
  const home = process.env.HOME;
  if (home) {
    const local = resolve(home, ".local/bin/rembg");
    if (existsSync(local)) return local;
  }
  throw new CLIError(
    "rembg not found on PATH or at ~/.local/bin/rembg. Install: pipx install rembg (or set REMBG_BIN)."
  );
}

async function removeBackground(imagePath: string): Promise<string> {
  const rembgBin = resolveRembgBin();
  if (!existsSync(rembgBin)) {
    throw new CLIError(
      `rembg not found at ${rembgBin}. Install: pipx install rembg (or set REMBG_BIN).`
    );
  }

  console.log("🔲 Removing background with local rembg...");

  // rembg always emits PNG. Force the output path to .png so we don't end up
  // with PNG bytes inside a .jpg extension.
  const currentExt = extname(imagePath).toLowerCase();
  const finalPath = currentExt === ".png" ? imagePath : imagePath.replace(/\.[^.]+$/, ".png");

  // rembg truncates output before reading input, so input == output corrupts
  // the file. Always write to a temp path, then rename.
  const tempPath = finalPath.replace(/\.png$/, `.rembg-tmp.png`);

  const { spawn } = await import("node:child_process");
  await new Promise<void>((resolveFn, rejectFn) => {
    const proc = spawn(rembgBin, ["i", imagePath, tempPath], { stdio: ["ignore", "ignore", "pipe"] });
    let stderr = "";
    proc.stderr.on("data", (chunk) => { stderr += chunk.toString(); });
    proc.on("error", (err) => rejectFn(new CLIError(`Failed to launch rembg: ${err.message}`)));
    proc.on("close", (code) => {
      if (code === 0) resolveFn();
      else rejectFn(new CLIError(`rembg exited ${code}: ${stderr.trim()}`));
    });
  });

  const { unlink, rename } = await import("node:fs/promises");
  // Drop the original (whether .jpg or the .png we're about to overwrite)
  try { await unlink(imagePath); } catch {}
  await rename(tempPath, finalPath);

  if (finalPath !== imagePath) {
    console.log(`   ↪ renamed ${currentExt} → .png (transparency requires PNG): ${finalPath}`);
  }

  // Validate output is actually PNG with alpha
  const result = await readFile(finalPath);
  const detected = detectImageFormat(result);
  if (!detected || detected.format !== "png") {
    throw new CLIError(
      `rembg produced non-PNG output (got ${detected?.format ?? "unknown"}). Transparency requires PNG.`
    );
  }

  console.log("✅ Background removed successfully");
  return finalPath;
}

// ============================================================================
// Image Generation
// ============================================================================

/** Output path for image i of n: the base path for one image, base-v<i> for several. */
function variantPath(outputBase: string, i: number, n: number): string {
  return n === 1 ? outputBase : outputBase.replace(/(\.[^.]+)?$/, (ext) => `-v${i}${ext || ".png"}`);
}

/** Request body for gpt-image-2 and Grok Imagine. Both speak the OpenAI images API. */
interface ImageRequest {
  model: string;
  prompt: string;
  n: number;
  size?: OpenAISize2;
  quality?: Quality;
  aspect_ratio?: AspectRatio | "auto"; // xAI only
  resolution?: "1k" | "2k"; // xAI only
  response_format?: "b64_json";
}

async function generateOpenAICompatible(
  model: "grok" | "gpt-image-2",
  prompt: string,
  args: CLIArgs,
  n: number,
  outputBase: string
): Promise<string[]> {
  const { provider, apiModel, label } = MODELS[model];
  const client = new OpenAI({
    apiKey: apiKey(provider)!,
    baseURL: provider === "xai" ? process.env.XAI_BASE_URL || "https://api.x.ai/v1" : process.env.OPENAI_BASE_URL,
  });

  const body: ImageRequest = { model: apiModel, prompt, n };
  if (model === "gpt-image-2") {
    body.size = args.openaiSize ?? openaiSizeFor(args.aspectRatio, args.resolution);
    body.quality = args.quality ?? "high";
  } else {
    const supported = GROK_ASPECT_RATIOS.includes(args.aspectRatio);
    if (!supported) console.warn(`⚠️ Grok has no ${args.aspectRatio}; using its "auto" ratio.`);
    body.aspect_ratio = supported ? args.aspectRatio : "auto";
    body.resolution = args.resolution === "1K" ? "1k" : "2k";
    body.response_format = "b64_json";
  }

  const detail = [body.size, body.quality, body.aspect_ratio, body.resolution].filter(Boolean).join(" ");
  console.log(`🎨 Generating ${n} image(s) with ${label} (${apiModel}) ${detail}...`);

  // The SDK's types predate gpt-image-2's 2048x2048 size and xAI's aspect_ratio/resolution;
  // the API accepts them, so widen once here.
  const response = await client.images.generate(body as unknown as OpenAI.ImageGenerateParamsNonStreaming);
  const data = response.data ?? [];
  if (data.length === 0) {
    throw new CLIError(`No image data returned from ${label}`);
  }

  const paths: string[] = [];
  for (const [i, item] of data.entries()) {
    let buffer: Buffer;
    if (item.b64_json) {
      buffer = Buffer.from(item.b64_json, "base64");
    } else if (item.url) {
      const resp = await fetch(item.url);
      if (!resp.ok) throw new CLIError(`${label} image ${i + 1} download failed: HTTP ${resp.status}`);
      buffer = Buffer.from(await resp.arrayBuffer());
    } else {
      throw new CLIError(`${label} returned image ${i + 1} with neither b64_json nor url`);
    }
    const finalPath = await saveImage(buffer, variantPath(outputBase, i + 1, data.length));
    console.log(`✅ ${label} image saved to ${finalPath}`);
    paths.push(finalPath);
  }
  return paths;
}

async function generateGeminiImage(
  model: "nano-banana" | "nano-banana-pro",
  prompt: string,
  args: CLIArgs,
  output: string
): Promise<string> {
  const { apiModel, label } = MODELS[model];
  const baseUrl = process.env.GEMINI_BASE_URL;
  const ai = new GoogleGenAI({ apiKey: apiKey("google")!, ...(baseUrl ? { httpOptions: { baseUrl } } : {}) });
  const refs = args.referenceImages ?? [];
  console.log(`🍌 Generating with ${label} (${apiModel}) at ${args.resolution} ${args.aspectRatio}${refs.length ? ` with ${refs.length} reference image(s)` : ""}...`);

  const parts: Array<{ text?: string; inlineData?: { mimeType: string; data: string } }> = [];
  for (const ref of refs) {
    const imageBuffer = await readFile(ref);
    parts.push({ inlineData: { mimeType: await detectMimeType(ref), data: imageBuffer.toString("base64") } });
  }
  parts.push({ text: prompt });

  const response = await ai.models.generateContent({
    model: apiModel,
    contents: [{ parts }],
    config: {
      responseModalities: ["TEXT", "IMAGE"],
      imageConfig: { aspectRatio: args.aspectRatio, imageSize: args.resolution },
    },
  });

  const imageData = response.candidates?.[0]?.content?.parts?.find((p) => p.inlineData?.data)?.inlineData?.data;
  if (!imageData) {
    throw new CLIError(`No image data returned from ${label}`);
  }

  const finalPath = await saveImage(Buffer.from(imageData, "base64"), output);
  console.log(`✅ ${label} image saved to ${finalPath}`);
  return finalPath;
}

/** One OpenRouter /images call for any of the four models. References go as data URLs. */
async function generateOpenRouter(model: ImageModel, prompt: string, args: CLIArgs, n: number, outputBase: string): Promise<string[]> {
  const { openRouterModel, label } = MODELS[model];
  const body: Record<string, unknown> = { model: openRouterModel, prompt, n };
  if (model === "gpt-image-2") {
    // OpenRouter rejects a pixel size combined with resolution or aspect_ratio; gpt-image-2 takes the size alone.
    body.size = args.openaiSize ?? openaiSizeFor(args.aspectRatio, args.resolution);
    body.quality = args.quality ?? "high";
  } else {
    body.aspect_ratio = args.aspectRatio;
    body.resolution = args.resolution;
  }
  const refs = args.referenceImages ?? [];
  if (refs.length > 0) {
    body.input_references = await Promise.all(
      refs.map(async (ref) => ({
        type: "image_url",
        image_url: { url: `data:${await detectMimeType(ref)};base64,${(await readFile(ref)).toString("base64")}` },
      }))
    );
  }

  console.log(`🔀 Generating ${n} image(s) with ${label} via OpenRouter (${openRouterModel})...`);
  const base = process.env.OPENROUTER_BASE_URL || "https://openrouter.ai/api/v1";
  const resp = await fetch(`${base}/images`, {
    method: "POST",
    headers: { Authorization: `Bearer ${apiKey("openrouter")}`, "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!resp.ok) {
    throw new CLIError(`OpenRouter ${openRouterModel} failed: HTTP ${resp.status} ${(await resp.text()).slice(0, 300)}`);
  }
  const json: unknown = await resp.json();
  const data = json && typeof json === "object" && "data" in json && Array.isArray(json.data) ? json.data : [];
  const images = data.flatMap((item: unknown) =>
    item && typeof item === "object" && "b64_json" in item && typeof item.b64_json === "string" ? [item.b64_json] : []
  );
  if (images.length === 0) throw new CLIError(`No image data returned from OpenRouter ${openRouterModel}`);

  const paths: string[] = [];
  for (const [i, b64] of images.entries()) {
    const finalPath = await saveImage(Buffer.from(b64, "base64"), variantPath(outputBase, i + 1, images.length));
    console.log(`✅ ${label} (OpenRouter) image saved to ${finalPath}`);
    paths.push(finalPath);
  }
  return paths;
}

/** Run one model for n images. OpenRouter, Grok, and gpt-image-2 batch natively; direct Gemini fans out. */
async function generateImages(model: ImageModel, route: Route, prompt: string, args: CLIArgs, n: number, outputBase: string): Promise<string[]> {
  if (route === "openrouter") return generateOpenRouter(model, prompt, args, n, outputBase);
  if (model === "grok" || model === "gpt-image-2") {
    return generateOpenAICompatible(model, prompt, args, n, outputBase);
  }
  return Promise.all(
    Array.from({ length: n }, (_, i) => generateGeminiImage(model, prompt, args, variantPath(outputBase, i + 1, n)))
  );
}

// ============================================================================
// Main
// ============================================================================

async function main(): Promise<void> {
  try {
    // Load optional env files
    await loadEnv();

    if (process.argv.includes("--check-keys")) {
      printKeyStatus();
      return;
    }

    const args = parseArgs(process.argv);

    // Enhance prompt for transparency if requested
    const finalPrompt = args.transparent
      ? enhancePromptForTransparency(args.prompt)
      : args.prompt;

    if (args.transparent) {
      console.log("🔲 Transparent background mode enabled");
      console.log("💡 Note: Not all models support transparency natively; may require post-processing\n");
    }

    const n = args.creativeVariations && args.creativeVariations > 1 ? args.creativeVariations : 1;

    // Compare mode: the same prompt on every model that has a route (own key or OpenRouter), one flagship each.
    if (args.model === "compare") {
      const runnable = FALLBACK_ORDER.flatMap((m) => {
        const route = routeFor(m);
        return route ? [{ model: m, route }] : [];
      });
      if (runnable.length === 0) {
        throw new MissingKeyError(`Compare mode needs at least one image provider key.\n\n${keySetupHelp()}`);
      }
      for (const m of FALLBACK_ORDER.filter((m) => !routeFor(m))) {
        console.warn(`⚠️ Skipping ${m}: ${PROVIDERS[MODELS[m].provider].envVars.join(" or ")} is not set (nor OPENROUTER_API_KEY).`);
      }
      if (runnable.length === 1) console.warn(`⚠️ Only ${runnable[0]!.model} has a key, so compare runs a single provider.`);

      console.log(`⚖️  Compare Mode: ${n} image(s) each from ${runnable.map((r) => (r.route === "openrouter" ? `${r.model} (OpenRouter)` : r.model)).join(", ")}`);
      const basePath = args.output.replace(/\.[^.]+$/, "");
      const results = await Promise.all(
        runnable.map(({ model, route }) =>
          generateImages(model, route, finalPrompt, args, n, `${basePath}-${model}.png`).catch((err) => {
            console.error(`❌ ${model} failed: ${err instanceof Error ? err.message : err}`);
            return [] as string[];
          })
        )
      );
      console.log(`\n✅ Compare complete`);
      runnable.forEach(({ model }, i) => console.log(`   ${model}: ${results[i]!.length}/${n} ${results[i]!.join(", ")}`));
      return;
    }

    const { model, route } = resolveModel(args.model, (args.referenceImages?.length ?? 0) > 0);

    // Creative variations: N images from the same prompt.
    if (n > 1) {
      console.log(`🎨 Creative Mode: Generating ${n} variations with ${model}...`);
      const paths = await generateImages(model, route, finalPrompt, args, n, args.output);
      console.log(`\n✅ Generated ${paths.length} variation(s)`);
      console.log(`   Files: ${paths.join(", ")}`);
      return;
    }

    // Single image. The saved path may differ from --output if the format was corrected.
    let actualOutput = (await generateImages(model, route, finalPrompt, args, 1, args.output))[0]!;

    // Remove background if requested (use actual output path)
    // May return a renamed path (e.g., .jpg → .png) since rembg returns PNG.
    if (args.removeBg) {
      actualOutput = await removeBackground(actualOutput);
    }

    // Add background color if requested (standalone mode)
    if (args.addBg && !args.thumbnail) {
      // For standalone --add-bg, modify the image in place
      const tempPath = actualOutput.replace(/\.[^.]+$/, "-temp.png");
      await addBackgroundColor(actualOutput, tempPath, args.addBg);
      // Replace original with the one with background
      const { rename } = await import("node:fs/promises");
      await rename(tempPath, actualOutput);
    }

    // Optional stamp. ART_SIGNATURE unset means no text. Runs before the thumb
    // so a requested signature lands on both files. --no-signature skips it.
    if (args.signature !== false) {
      const signature = process.env.ART_SIGNATURE?.trim();
      if (signature) await stampSignature(actualOutput, signature);
      else if (args.signature === true) {
        console.error("ART_SIGNATURE is unset; no signature stamped.");
      }
    }

    // Generate thumbnail with background color if requested (blog header mode)
    if (args.thumbnail) {
      const thumbPath = actualOutput.replace(/\.[^.]+$/, "-thumb.png");
      const THUMBNAIL_BG_COLOR = "#EAE9DF"; // sepia preview background for social thumbs
      await addBackgroundColor(actualOutput, thumbPath, THUMBNAIL_BG_COLOR);
      console.log(`\n📸 Blog header mode: Created both versions`);
      console.log(`   Transparent: ${actualOutput}`);
      console.log(`   Thumbnail:   ${thumbPath}`);
    }
  } catch (error) {
    handleError(error);
  }
}

main();
