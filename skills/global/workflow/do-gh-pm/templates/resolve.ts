// The resolver: (archetype key, overlay keys) -> a single merged Archetype.
// Pure and deterministic — the same inputs always yield the same board spec.
// This is what the later interview and reconcile engine call.

import type { Archetype, PriorityScheme, SelectOption } from "./types.ts";
import { ARCHETYPES } from "./archetypes.ts";
import { OVERLAYS } from "./overlays.ts";

const PRIORITY_OPTIONS: Record<Exclude<PriorityScheme, "none">, SelectOption[]> = {
  "p0-p3": [
    { name: "P0", color: "RED", description: "critical" },
    { name: "P1", color: "ORANGE", description: "high" },
    { name: "P2", color: "YELLOW", description: "medium" },
    { name: "P3", color: "BLUE", description: "low" },
  ],
  "high-medium-low": [
    { name: "High", color: "RED" },
    { name: "Medium", color: "YELLOW" },
    { name: "Low", color: "BLUE" },
  ],
};

export function resolve(
  archetypeKey: string,
  overlayKeys: string[] = [],
  priorityScheme: PriorityScheme = "p0-p3",
): Archetype {
  const base = ARCHETYPES[archetypeKey];
  if (!base) {
    throw new Error(
      `Unknown archetype "${archetypeKey}". Known: ${Object.keys(ARCHETYPES).join(", ")}`,
    );
  }
  if (!Object.hasOwn(PRIORITY_OPTIONS, priorityScheme) && priorityScheme !== "none") {
    throw new Error(`Unknown priority scheme "${priorityScheme}"`);
  }

  const merged = overlayKeys.reduce<Archetype>((acc, key) => {
    const overlay = OVERLAYS[key];
    if (!overlay) {
      throw new Error(
        `Unknown overlay "${key}". Known: ${Object.keys(OVERLAYS).join(", ")}`,
      );
    }
    return overlay(acc);
  }, base);
  const resolved = structuredClone(merged);
  const priorityIndex = resolved.fields.findIndex((field) => field.name === "Priority");
  if (priorityScheme === "none") {
    if (priorityIndex >= 0) resolved.fields.splice(priorityIndex, 1);
  } else if (priorityIndex >= 0) {
    resolved.fields[priorityIndex] = {
      ...resolved.fields[priorityIndex],
      options: structuredClone(PRIORITY_OPTIONS[priorityScheme]),
    };
  }
  return resolved;
}

export { ARCHETYPES } from "./archetypes.ts";
export { OVERLAYS } from "./overlays.ts";
export type { Archetype, Field, View, Label, FieldType, PriorityScheme } from "./types.ts";
