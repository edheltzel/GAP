// Pure merge helpers shared by overlays and the resolver. Deterministic:
// first occurrence of a given name wins, input order is otherwise preserved.

import type { Archetype, SelectOption, Field, Label, View } from "./types.ts";

export function dedupeByName<T extends { name: string }>(items: T[]): T[] {
  const seen = new Set<string>();
  const out: T[] = [];
  for (const item of items) {
    if (seen.has(item.name)) continue;
    seen.add(item.name);
    out.push(item);
  }
  return out;
}

interface Delta {
  // Prepended to the existing status vocabulary (e.g. `triage` adds "Triage"
  // as the first column). Deduped by name.
  status?: SelectOption[];
  fields?: Field[];
  labels?: Label[];
  views?: View[];
}

// Additively merge a delta into an archetype, returning a new object. The
// source archetype is never mutated.
export function extend(a: Archetype, delta: Delta): Archetype {
  return {
    ...a,
    status: dedupeByName([...(delta.status ?? []), ...a.status]),
    fields: dedupeByName([...a.fields, ...(delta.fields ?? [])]),
    labels: dedupeByName([...a.labels, ...(delta.labels ?? [])]),
    views: dedupeByName([...a.views, ...(delta.views ?? [])]),
  };
}
