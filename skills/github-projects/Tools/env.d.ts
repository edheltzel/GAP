// Minimal ambient declarations for the Bun/runtime globals Reconcile.ts uses.
// Kept local (no node_modules) to match the templates/ zero-dependency
// convention while keeping `bunx tsc --noEmit` clean. Only the surface actually
// used is declared.

declare const process: {
  argv: string[];
  exitCode?: number;
  exit(code?: number): never;
};

declare namespace Bun {
  function spawnSync(
    cmd: string[],
    options?: { stdin?: Uint8Array | string },
  ): { stdout: Uint8Array; stderr: Uint8Array; exitCode: number; success: boolean };
  function write(path: string, data: string): Promise<number>;
  function file(path: string): { text(): Promise<string>; exists(): Promise<boolean> };
}
