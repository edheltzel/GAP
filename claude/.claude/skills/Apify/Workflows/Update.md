# Update Workflow

Review and update the bundled Apify wrapper without making live actor calls during validation.

## Sources

- API changelog: <https://docs.apify.com/api/v2/changelog>
- Actor Store: <https://apify.com/store>
- `apify-client` package: <https://www.npmjs.com/package/apify-client>

## Procedure

1. Read the changelog and actor schemas relevant to the requested update.
2. Update wrappers and types under `~/.claude/skills/Apify`.
3. If the dependency changes, update the exact version in `package.json` and regenerate `bun.lock` with Bun.
4. Install with `bun install --frozen-lockfile --ignore-scripts` to prove the lock is complete.
5. Run the offline smoke test and TypeScript check.
6. Review the diff for actor IDs, input fields, output assumptions, and accidental network execution.
7. Document any actor behavior that still requires an explicitly authorized live check.

## Verification

```bash
cd ~/.claude/skills/Apify
bun install --frozen-lockfile --ignore-scripts
bun examples/smoke-test.ts
bunx tsc --noEmit -p tsconfig.json
```

These checks must not execute an actor or query the Apify API.
