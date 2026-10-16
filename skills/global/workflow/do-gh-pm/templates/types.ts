// Typed primitives for the GitHub Projects v2 template layer.
//
// These describe a board's *intended* shape as config-as-code: the source of
// truth a later interview resolves and a reconcile engine converges a live
// board toward. Field types are limited to what the Projects v2 API can
// actually create (see the skill plan §2 "Verified API reality").

// Projects v2 field data types. ITERATION is GraphQL-only; the rest are
// `gh project field-create`-able. No other types exist in the API.
export type FieldType = "TEXT" | "NUMBER" | "DATE" | "SINGLE_SELECT" | "ITERATION";

// The 8-value color enum the GraphQL API accepts for single-select options
// (ProjectV2SingleSelectFieldOptionColor). Not a hex string.
export type SelectColor =
  | "GRAY"
  | "BLUE"
  | "GREEN"
  | "YELLOW"
  | "ORANGE"
  | "RED"
  | "PINK"
  | "PURPLE";

export type PriorityScheme = "p0-p3" | "high-medium-low" | "none";

export interface SelectOption {
  name: string;
  color: SelectColor;
  description?: string;
}

export interface Field {
  name: string;
  type: FieldType;
  // Present only for SINGLE_SELECT fields.
  options?: SelectOption[];
}

export interface View {
  name: string;
  layout: "BOARD" | "TABLE" | "ROADMAP";
  // Projects v2 filter string, e.g. `status:Ready,"In Progress"`.
  filter?: string;
}

export interface Label {
  name: string;
  // 6-digit hex without a leading '#' — the form `gh label create --color`
  // expects. Canon colors are preserved from the live Recall taxonomy.
  color: string;
  description?: string;
}

export interface Archetype {
  key: string;
  title: string;
  // Status is GitHub's built-in single-select field; its vocabulary lives here
  // (reconciled in place, never created). Board order is significant.
  status: SelectOption[];
  // Created board fields. Priority is shared by every archetype; the remainder
  // are archetype-specific extras and overlay additions.
  fields: Field[];
  labels: Label[];
  views: View[];
}

// An overlay is a pure transform that returns a NEW archetype with additive
// status/fields/labels/views merged in. Overlays compose by function application.
export type Overlay = (a: Archetype) => Archetype;
