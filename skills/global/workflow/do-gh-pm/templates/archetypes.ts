// The five base archetypes (skill plan §5) plus the canonical label taxonomy
// (§4.2). Every archetype shares the canon labels and a Priority field; each
// adds its own Status vocabulary, extra fields, and distinct labels.

import type { Archetype, Field, Label } from "./types.ts";

// --- Canonical `:`-namespaced label taxonomy (§4.2) -------------------------
// Synced across every repo on bootstrap. Colors preserved from the live Recall
// taxonomy (6-hex, no leading '#').
export const CANON_LABELS: Label[] = [
  { name: "type:bug", color: "d4c5f9", description: "work kind: bug" },
  { name: "type:feature", color: "d4c5f9", description: "work kind: feature" },
  { name: "type:docs", color: "d4c5f9", description: "work kind: docs" },
  { name: "type:test", color: "d4c5f9", description: "work kind: test" },
  { name: "type:refactor", color: "d4c5f9", description: "work kind: refactor" },
  { name: "type:chore", color: "d4c5f9", description: "work kind: chore" },
  { name: "needs:triage", color: "FBCA04", description: "maintainer needs to evaluate" },
  { name: "needs:info", color: "c73aa4", description: "waiting on reporter for info" },
  { name: "needs:human", color: "b60205", description: "requires human implementation" },
  { name: "risk:low", color: "edeedd", description: "low ambiguity / judgement" },
  { name: "risk:medium", color: "fef2c0", description: "medium ambiguity / judgement" },
  { name: "risk:high", color: "f9d0c4", description: "high ambiguity / judgement" },
  { name: "agent:ready", color: "bfd4f2", description: "ready for an AFK agent" },
  { name: "agent:blocked", color: "bfd4f2", description: "agent work is blocked" },
  { name: "agent:complete", color: "bfd4f2", description: "agent work complete" },
  { name: "wontfix", color: "ededed", description: "will not be actioned" },
];

// Shared Priority field (§4.2 default scheme P0–P3). The interview may swap the
// scheme; this is the default carried by every archetype.
const PRIORITY: Field = {
  name: "Priority",
  type: "SINGLE_SELECT",
  options: [
    { name: "P0", color: "RED", description: "drop everything" },
    { name: "P1", color: "ORANGE", description: "high" },
    { name: "P2", color: "YELLOW", description: "medium" },
    { name: "P3", color: "GRAY", description: "low" },
  ],
};

export const ARCHETYPES: Record<string, Archetype> = {
  // Default archetype. Estimate lives in the `estimation` overlay (opt-in,
  // §4.1 Q5), so the base extras are Phase + Area.
  "solo-software": {
    key: "solo-software",
    title: "Solo Software",
    status: [
      { name: "Backlog", color: "GRAY" },
      { name: "Ready", color: "BLUE" },
      { name: "In Progress", color: "YELLOW" },
      { name: "In Review", color: "ORANGE" },
      { name: "Done", color: "GREEN" },
    ],
    fields: [
      PRIORITY,
      {
        name: "Phase",
        type: "SINGLE_SELECT",
        options: [
          { name: "Phase 1", color: "BLUE" },
          { name: "Phase 2", color: "YELLOW" },
          { name: "Phase 3", color: "GREEN" },
        ],
      },
      { name: "Area", type: "TEXT" },
    ],
    // type:* is the canon work-kind; "blocked" intentionally omitted — use
    // native issue dependencies (§2) instead of a label.
    labels: [...CANON_LABELS],
    views: [
      { name: "Board by Status", layout: "BOARD" },
      { name: "Ready", layout: "TABLE", filter: 'status:Ready,"In Progress"' },
    ],
  },

  "content-pipeline": {
    key: "content-pipeline",
    title: "Content Pipeline",
    status: [
      { name: "Idea", color: "GRAY" },
      { name: "Drafting", color: "BLUE" },
      { name: "Editing", color: "YELLOW" },
      { name: "Review", color: "ORANGE" },
      { name: "Scheduled", color: "PURPLE" },
      { name: "Published", color: "GREEN" },
    ],
    fields: [
      PRIORITY,
      {
        name: "Channel",
        type: "SINGLE_SELECT",
        options: [
          { name: "blog", color: "BLUE" },
          { name: "newsletter", color: "GREEN" },
          { name: "social", color: "PURPLE" },
        ],
      },
      { name: "Publish Date", type: "DATE" },
    ],
    labels: [
      ...CANON_LABELS,
      { name: "channel:blog", color: "0052cc", description: "blog channel" },
      { name: "channel:newsletter", color: "0052cc", description: "newsletter channel" },
      { name: "channel:social", color: "0052cc", description: "social channel" },
      { name: "needs-art", color: "d93f0b", description: "needs accompanying art" },
    ],
    views: [{ name: "Board by Status", layout: "BOARD" }],
  },

  "research-tracker": {
    key: "research-tracker",
    title: "Research Tracker",
    status: [
      { name: "Open Question", color: "GRAY" },
      { name: "Investigating", color: "BLUE" },
      { name: "Evidence Gathered", color: "YELLOW" },
      { name: "Concluded", color: "GREEN" },
      { name: "Parked", color: "ORANGE" },
    ],
    fields: [
      PRIORITY,
      {
        name: "Confidence",
        type: "SINGLE_SELECT",
        options: [
          { name: "Low", color: "RED" },
          { name: "Medium", color: "YELLOW" },
          { name: "High", color: "GREEN" },
        ],
      },
      { name: "Source", type: "TEXT" },
    ],
    labels: [
      ...CANON_LABELS,
      { name: "hypothesis", color: "fbca04", description: "a hypothesis to test" },
      { name: "finding", color: "0e8a16", description: "a confirmed finding" },
      { name: "dead-end", color: "b60205", description: "investigated, no path" },
      { name: "needs-verification", color: "d93f0b", description: "claim needs verifying" },
    ],
    views: [{ name: "Board by Status", layout: "BOARD" }],
  },

  "life-os-tasks": {
    key: "life-os-tasks",
    title: "Life-OS Tasks",
    status: [
      { name: "Inbox", color: "GRAY" },
      { name: "Next", color: "BLUE" },
      { name: "In Progress", color: "YELLOW" },
      { name: "Waiting", color: "ORANGE" },
      { name: "Done", color: "GREEN" },
    ],
    fields: [
      PRIORITY,
      { name: "Area", type: "TEXT" },
      { name: "Due", type: "DATE" },
      {
        name: "Energy",
        type: "SINGLE_SELECT",
        options: [
          { name: "Low", color: "GREEN" },
          { name: "Medium", color: "YELLOW" },
          { name: "High", color: "RED" },
        ],
      },
    ],
    labels: [
      ...CANON_LABELS,
      { name: "quick-win", color: "0e8a16", description: "fast, high-value" },
      { name: "recurring", color: "5319e7", description: "repeats" },
      { name: "someday", color: "ededed", description: "maybe later" },
    ],
    views: [{ name: "Board by Status", layout: "BOARD" }],
  },

  "open-source-repo": {
    key: "open-source-repo",
    title: "Open-Source Repo",
    status: [
      { name: "Triage", color: "RED" },
      { name: "Accepted", color: "BLUE" },
      { name: "In Progress", color: "YELLOW" },
      { name: "In Review", color: "ORANGE" },
      { name: "Done", color: "GREEN" },
    ],
    fields: [
      PRIORITY,
      {
        name: "Type",
        type: "SINGLE_SELECT",
        options: [
          { name: "Bug", color: "RED" },
          { name: "Feature", color: "GREEN" },
          { name: "Docs", color: "BLUE" },
          { name: "Chore", color: "GRAY" },
        ],
      },
      {
        name: "Difficulty",
        type: "SINGLE_SELECT",
        options: [
          { name: "Easy", color: "GREEN" },
          { name: "Medium", color: "YELLOW" },
          { name: "Hard", color: "RED" },
        ],
      },
    ],
    labels: [
      ...CANON_LABELS,
      { name: "good first issue", color: "7057ff", description: "good for newcomers" },
      { name: "help wanted", color: "008672", description: "extra attention wanted" },
      { name: "needs-repro", color: "d93f0b", description: "needs a reproduction" },
    ],
    views: [{ name: "Board by Status", layout: "BOARD" }],
  },
};
