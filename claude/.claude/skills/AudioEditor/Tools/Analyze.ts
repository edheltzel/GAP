#!/usr/bin/env bun
// @ts-nocheck - Standalone Bun CLI; behavior is covered by native-runtime-skills.test.sh.
/**
 * Analyze.ts
 *
 * Validates model edit classifications supplied by the active agent, adds
 * deterministic long-pause edits, and writes the list consumed by Edit.ts.
 * This tool does not call a model.
 *
 * Usage:
 *   bun Analyze.ts <transcript.json> --model-edits <edits.json> [options]
 *   bun Analyze.ts <transcript.json> --pauses-only [options]
 */

import { existsSync } from "node:fs";

interface Chunk {
  text: string;
  timestamp: [number, number | null];
}

interface EditDecision {
  type: string;
  start: number;
  end: number;
  reason: string;
  context: string;
  confidence: number;
}

const ALLOWED_TYPES = new Set([
  "CUT_FILLER",
  "CUT_FALSE_START",
  "CUT_EDIT_MARKER",
  "CUT_STUTTER",
  "CUT_SELF_CORRECTION",
  "CUT_REDO_TAKE",
  "CUT_DEAD_AIR",
]);

function flagValue(args: string[], name: string): string | undefined {
  const index = args.indexOf(name);
  return index >= 0 ? args[index + 1] : undefined;
}

function fail(message: string): never {
  console.error(message);
  process.exit(1);
}

function validateEdit(value: unknown, index: number): EditDecision {
  if (!value || typeof value !== "object") fail(`Invalid model edit ${index}: expected an object`);
  const edit = value as Record<string, unknown>;
  if (typeof edit.type !== "string" || !ALLOWED_TYPES.has(edit.type)) {
    fail(`Invalid model edit ${index}: unsupported type`);
  }
  if (typeof edit.start !== "number" || typeof edit.end !== "number" || edit.start < 0 || edit.end <= edit.start) {
    fail(`Invalid model edit ${index}: start and end must be an increasing non-negative range`);
  }
  if (typeof edit.reason !== "string" || typeof edit.context !== "string") {
    fail(`Invalid model edit ${index}: reason and context must be strings`);
  }
  if (typeof edit.confidence !== "number" || edit.confidence < 0 || edit.confidence > 1) {
    fail(`Invalid model edit ${index}: confidence must be between 0 and 1`);
  }
  return edit as unknown as EditDecision;
}

const args = process.argv.slice(2);
const inputFile = args[0] && !args[0].startsWith("--") ? args[0] : undefined;
const outputPath = flagValue(args, "--output");
const modelEditsPath = flagValue(args, "--model-edits");
const pausesOnly = args.includes("--pauses-only");
const aggressive = args.includes("--aggressive");

if (!inputFile) {
  fail("Usage: bun Analyze.ts <transcript.json> (--model-edits <edits.json> | --pauses-only) [--output <path>] [--aggressive]");
}
if (!modelEditsPath && !pausesOnly) {
  fail("Analyze requires --model-edits <edits.json> from inline classification, or --pauses-only for deterministic silence cuts.");
}
if (modelEditsPath && pausesOnly) fail("Choose either --model-edits or --pauses-only, not both.");
if (!existsSync(inputFile)) fail(`File not found: ${inputFile}`);
if (modelEditsPath && !existsSync(modelEditsPath)) fail(`Model edits file not found: ${modelEditsPath}`);

const outFile = outputPath || inputFile.replace(/\.transcript\.json$/, ".edits.json").replace(/\.json$/, ".edits.json");
const transcript = await Bun.file(inputFile).json();
const chunks = transcript?.chunks as Chunk[] | undefined;
if (!Array.isArray(chunks) || chunks.length === 0) fail("No word chunks found in transcript");

for (let i = 0; i < chunks.length; i++) {
  const chunk = chunks[i];
  if (!chunk || typeof chunk.text !== "string" || !Array.isArray(chunk.timestamp) || typeof chunk.timestamp[0] !== "number") {
    fail(`Invalid transcript chunk ${i}`);
  }
}

const edits: EditDecision[] = [];
const pauseThreshold = aggressive ? 3 : 5;
const keepPause = 1;
for (let i = 1; i < chunks.length; i++) {
  const previousEnd = chunks[i - 1].timestamp[1] ?? chunks[i - 1].timestamp[0];
  const currentStart = chunks[i].timestamp[0];
  const gap = currentStart - previousEnd;
  const start = previousEnd + keepPause;
  if (gap > pauseThreshold && currentStart - start > 0.5) {
    edits.push({
      type: "CUT_DEAD_AIR",
      start: Math.round(start * 100) / 100,
      end: Math.round(currentStart * 100) / 100,
      reason: `${gap.toFixed(1)}s pause (keeping ${keepPause}s)`,
      context: chunks.slice(Math.max(0, i - 3), i + 3).map((chunk) => chunk.text.trim()).join(" "),
      confidence: 1,
    });
  }
}

if (modelEditsPath) {
  const supplied = await Bun.file(modelEditsPath).json();
  if (!Array.isArray(supplied)) fail("Model edits input must be a JSON array");
  for (let i = 0; i < supplied.length; i++) {
    const edit = validateEdit(supplied[i], i);
    if (edit.confidence >= 0.6 && !edits.some((known) => Math.abs(known.start - edit.start) < 1 && Math.abs(known.end - edit.end) < 1)) {
      edits.push(edit);
    }
  }
}

edits.sort((a, b) => a.start - b.start);
const merged: EditDecision[] = [];
for (const edit of edits) {
  const previous = merged[merged.length - 1];
  if (previous && edit.start < previous.end + 0.3) {
    previous.end = Math.max(previous.end, edit.end);
    if (!previous.type.split("+").includes(edit.type)) previous.type += `+${edit.type}`;
    previous.reason += `; ${edit.reason}`;
  } else {
    merged.push({ ...edit });
  }
}

await Bun.write(outFile, JSON.stringify(merged, null, 2));
console.log(`Validated ${modelEditsPath ? "inline model edits and " : ""}${edits.filter((edit) => edit.type === "CUT_DEAD_AIR").length} long pauses.`);
console.log(`Saved: ${outFile}`);
