#!/usr/bin/env bun
/**
 * Compatibility guard for the retired personal TELOS updater.
 *
 * Personal TELOS is private state. This command is intentionally read-only and
 * never creates files, backups, or changelog entries.
 */

declare const process: {
  argv: string[];
  exit(code?: number): never;
};

const VALID_FILES = [
  'BELIEFS.md', 'BOOKS.md', 'CHALLENGES.md', 'FRAMES.md', 'GOALS.md',
  'LESSONS.md', 'MISSION.md', 'MODELS.md', 'MOVIES.md', 'NARRATIVES.md',
  'PREDICTIONS.md', 'PROBLEMS.md', 'PROJECTS.md', 'STRATEGIES.md',
  'TELOS.md', 'TRAUMAS.md', 'WISDOM.md', 'WRONG.md',
];

const args = process.argv.slice(2);
if (args.includes('--help') || args.includes('-h') || args.length === 0) {
  console.log(`update-telos compatibility guard

Personal TELOS mutation is disabled. This tool is read-only and makes no changes.
Read current files with the harness file tools, draft a proposed patch, and ask
for explicit user approval before any separate manual edit.

Known filenames: ${VALID_FILES.join(', ')}`);
  process.exit(0);
}

const filename = args[0] || '';
if (!VALID_FILES.includes(filename)) {
  console.error(`Invalid TELOS filename: ${filename}`);
  console.error(`Known filenames: ${VALID_FILES.join(', ')}`);
  process.exit(2);
}

console.error('Personal TELOS updates are read-only in the current native skill.');
console.error('No file was created, modified, backed up, or logged.');
console.error('Read the current file, prepare a proposed patch, and request explicit approval.');
process.exit(2);
