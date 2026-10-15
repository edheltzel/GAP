#!/usr/bin/env bun

/**
 * remove-bg - Background Removal CLI
 *
 * Remove backgrounds from images using local rembg.
 * Part of the native Art skill.
 *
 * Usage:
 *   remove-bg input.png                    # Overwrites original
 *   remove-bg input.png output.png         # Saves to new file
 *   remove-bg file1.png file2.png file3.png # Batch process
 */

import { delimiter, extname, resolve } from "node:path";
import { accessSync, constants, existsSync } from "node:fs";
import { unlink, stat, rename } from "node:fs/promises";
import { spawn } from "node:child_process";

function errorCode(error: unknown): string | undefined {
  if (typeof error !== "object" || error === null || !("code" in error)) return undefined;
  const code = Reflect.get(error, "code");
  return typeof code === "string" ? code : undefined;
}

function isExecutable(path: string): boolean {
  try {
    accessSync(path, constants.X_OK);
    return true;
  } catch {
    return false;
  }
}

function resolveRembgBin(): string {
  if (process.env.REMBG_BIN) return resolve(process.env.REMBG_BIN);

  for (const dir of (process.env.PATH || "").split(delimiter)) {
    if (!dir) continue;
    const candidate = resolve(dir, "rembg");
    if (isExecutable(candidate)) return candidate;
  }

  const home = process.env.HOME;
  if (!home) throw new Error("HOME not set and rembg is not on PATH");
  return resolve(home, ".local/bin/rembg");
}

function showHelp(): void {
  const bin = resolveRembgBin();
  console.log(`
remove-bg - Background Removal CLI

Remove backgrounds from images using local rembg (no API).

USAGE:
  remove-bg <input> [output]           Single file
  remove-bg <file1> <file2> ...        Batch process (overwrites originals)

ARGUMENTS:
  input     Path to image file (PNG, JPG, JPEG, WebP)
  output    Optional output path (defaults to overwriting input)

EXAMPLES:
  remove-bg header.png
  remove-bg header.jpg header-transparent.png
  remove-bg diagram1.png diagram2.png diagram3.png

NOTES:
  - rembg always produces PNG output (alpha channel required)
  - If input is .jpg/.jpeg/.webp and you overwrite, the file is renamed to .png
  - Requires rembg at ${bin} (override via REMBG_BIN env var)
    Install: pipx install rembg  (or: uv tool install rembg)
`);
  process.exit(0);
}

async function unlinkIfPresent(path: string): Promise<void> {
  try {
    await unlink(path);
  } catch (error) {
    if (errorCode(error) !== "ENOENT") throw error;
  }
}

function runRembg(bin: string, input: string, output: string): Promise<void> {
  return new Promise((resolveFn, rejectFn) => {
    const proc = spawn(bin, ["i", input, output], { stdio: ["ignore", "ignore", "pipe"] });
    let stderr = "";
    proc.stderr.on("data", (chunk) => { stderr += chunk.toString(); });
    proc.on("error", (err) => rejectFn(new Error(`Failed to launch rembg: ${err.message}`)));
    proc.on("close", (code) => {
      if (code === 0) resolveFn();
      else rejectFn(new Error(`rembg exited ${code}: ${stderr.trim()}`));
    });
  });
}

async function removeBackground(inputPath: string, outputPath?: string): Promise<void> {
  const bin = resolveRembgBin();
  if (!existsSync(bin) || !isExecutable(bin)) {
    console.error(`rembg not found or not executable at ${bin}`);
    console.error("Install: pipx install rembg  (or: uv tool install rembg)");
    console.error("Alternatively put rembg on PATH or set REMBG_BIN=/path/to/rembg");
    process.exit(1);
  }

  if (!existsSync(inputPath)) {
    console.error(`❌ File not found: ${inputPath}`);
    process.exit(1);
  }

  const inputExt = extname(inputPath).toLowerCase();
  let target = outputPath ?? inputPath;
  let renameAfter: { from: string; to: string } | null = null;
  if (!outputPath && inputExt !== ".png") {
    target = inputPath.replace(/\.[^.]+$/, ".png");
    renameAfter = { from: inputPath, to: target };
  }

  console.log(`🔲 Removing background: ${inputPath}`);
  const start = Date.now();
  // Always cut to a temp file first, then move into place. Passing the same
  // path as both input and output makes rembg TRUNCATE its own input to 0 bytes
  // before it reads it; the 2026-06-08 in-place-zeroing bug that silently
  // destroyed the source and failed with "cannot identify image file". A temp
  // file makes every path safe: in-place .png, explicit same-path output, and
  // jpg/webp→png all route through a distinct file rembg can actually read.
  const tmp = `${target}.rembg-tmp-${process.pid}.png`;
  try {
    await runRembg(bin, inputPath, tmp);
  } catch (error) {
    await unlinkIfPresent(tmp);
    console.error(`❌ ${error instanceof Error ? error.message : String(error)}`);
    process.exit(1);
  }

  // Move temp → target (overwrites safely; same dir means an atomic rename).
  try {
    await rename(tmp, target);
  } catch (error) {
    await unlinkIfPresent(tmp);
    const detail = error instanceof Error ? `: ${error.message}` : "";
    console.error(`❌ Failed to move result into place: ${target}${detail}`);
    process.exit(1);
  }

  // If we converted jpg/webp → png at a NEW path, remove the original source.
  if (renameAfter && renameAfter.from !== renameAfter.to) {
    await unlinkIfPresent(renameAfter.from);
  }

  const elapsed = Date.now() - start;
  const s = await stat(target);
  const dims = ` (${(s.size / 1024).toFixed(0)}KB)`;
  console.log(`✅ Saved: ${target}${dims} in ${elapsed}ms`);
}

async function main(): Promise<void> {
  const args = process.argv.slice(2);

  if (args.length === 0 || args.includes("--help") || args.includes("-h")) {
    showHelp();
  }

  if (args.length === 1) {
    await removeBackground(args[0]!);
    return;
  }

  if (args.length === 2) {
    if (existsSync(args[1]!)) {
      for (const file of args) {
        await removeBackground(file);
      }
    } else {
      await removeBackground(args[0]!, args[1]!);
    }
    return;
  }

  console.log(`🔲 Batch processing ${args.length} files...\n`);
  let success = 0;
  let failed = 0;

  for (const file of args) {
    try {
      await removeBackground(file);
      success++;
    } catch (error) {
      failed++;
      console.error(`❌ ${file}: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  console.log(`\n📊 Complete: ${success} succeeded, ${failed} failed`);
}

main();
