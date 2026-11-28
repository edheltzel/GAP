/** Site name and description. `gap` alone is ambiguous in search. */
export const siteName = 'GAP: Global Agent Policies';

export const siteDescription = 'One set of rules for every coding agent. gap sync copies global/AGENTS.md into the file each agent reads, installs pinned skills, and registers MCP servers.';

/** Search description for each docs page, keyed by page id (see repoPages). */
export const pageDescriptions = {
  docs: 'Install gap, write the global agent policy once, sync it to Claude, Codex, Cursor, Pi, Empryo and more, and manage pinned skills and utilities.',
  'docs/concepts': 'The words GAP uses, defined once: harness, skill, policy, publish, utility, component, pin, store, drift, load level, override and profile.',
};

/** Install line from the GAP README. The hero shows this and the copy button reads it. */
export const installCommand =
  'curl -fsSL https://raw.githubusercontent.com/edheltzel/GAP/master/scripts/install.sh | sh';

/**
 * Social preview image for every page. Landing.astro and the docs route resolve
 * `path` against the `site` origin. File: public/og.png, 1200x630.
 */
export const ogImage = {
  path: '/og.png',
  width: 1200,
  height: 630,
  alt: 'GAP: Global Agent Policies. One set of rules for every coding agent.',
};
