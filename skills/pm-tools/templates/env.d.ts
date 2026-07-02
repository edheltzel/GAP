// Minimal ambient declarations for the Bun/runtime surface smoke.ts uses.
// Kept local (no node_modules) to match the templates/ zero-dependency
// convention while keeping `bunx tsc --noEmit` clean. Only the surface
// actually used is declared.

declare module "node:fs" {
  export function existsSync(path: string): boolean;
  export function readFileSync(path: string, encoding: "utf8"): string;
}

// Bun exposes the importing module's directory as `import.meta.dir`.
interface ImportMeta {
  dir: string;
}

// Bun ships a built-in YAML parser (used to validate the scaffold templates).
declare namespace Bun {
  const YAML: { parse(input: string): any };
}
