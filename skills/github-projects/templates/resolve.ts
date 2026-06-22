// The resolver: (archetype key, overlay keys) -> a single merged Archetype.
// Pure and deterministic — the same inputs always yield the same board spec.
// This is what the later interview and reconcile engine call.

import type { Archetype } from "./types.ts";
import { ARCHETYPES } from "./archetypes.ts";
import { OVERLAYS } from "./overlays.ts";

export function resolve(archetypeKey: string, overlayKeys: string[] = []): Archetype {
  const base = ARCHETYPES[archetypeKey];
  if (!base) {
    throw new Error(
      `Unknown archetype "${archetypeKey}". Known: ${Object.keys(ARCHETYPES).join(", ")}`,
    );
  }
  return overlayKeys.reduce<Archetype>((acc, key) => {
    const overlay = OVERLAYS[key];
    if (!overlay) {
      throw new Error(
        `Unknown overlay "${key}". Known: ${Object.keys(OVERLAYS).join(", ")}`,
      );
    }
    return overlay(acc);
  }, base);
}

export { ARCHETYPES } from "./archetypes.ts";
export { OVERLAYS } from "./overlays.ts";
export type { Archetype, Field, View, Label, FieldType } from "./types.ts";
