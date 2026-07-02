// Minimal ambient declarations for the Bun/runtime surface smoke.ts uses.
// Kept local (no node_modules) to match the templates/ zero-dependency
// convention while keeping `bunx tsc --noEmit` clean. Only the surface
// actually used is declared.

declare module "node:fs" {
  export function existsSync(path: string): boolean;
}

// Bun exposes the importing module's directory as `import.meta.dir`.
interface ImportMeta {
  dir: string;
}
