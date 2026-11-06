/** Wording from docs-copy.md. `gap` alone is ambiguous in search. */
export const siteName = 'GAP: Global Agent Policies';

export const siteDescription =
  'One set of rules for every coding agent. Write the rules once in global/AGENTS.md. gap sync copies them into the file each agent already reads.';

/** Install line from the GAP README. The hero shows this and the copy button reads it. */
export const installCommand =
  'curl -fsSL https://raw.githubusercontent.com/edheltzel/GAP/master/scripts/install.sh | sh';

/**
 * Social preview image. The landing page (Base.astro) resolves `path` against
 * the `site` origin. File: public/og.png, 1200x630.
 */
export const ogImage = {
  path: '/og.png',
  width: 1200,
  height: 630,
  alt: 'GAP: Global Agent Policies. One set of rules for every coding agent.',
};
