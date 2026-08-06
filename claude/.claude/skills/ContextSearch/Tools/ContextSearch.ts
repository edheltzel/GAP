#!/usr/bin/env bun
// @ts-nocheck - Standalone Bun compatibility shim; covered by native-runtime-skills.test.sh.
/**
 * Compatibility shim for the retired local ContextSearch index.
 *
 * Recall is exposed through MCP, not a subprocess API. This command prints the
 * exact current tool and arguments for the active project, then exits without
 * claiming that a search ran.
 */

import { basename, resolve } from "node:path";

function projectName(): string {
  return process.env.RECALL_PROJECT || basename(resolve(process.cwd())).toLowerCase();
}

const args = process.argv.slice(2);
const json = args.includes("--json");
const help = args.includes("--help") || args.includes("-h");
const query = args.filter((arg) => !arg.startsWith("--")).join(" ").trim();

if (help) {
  console.log(`ContextSearch compatibility shim

The retired LifeOS state index is no longer searched. From an agent session,
use Recall directly:

  recall_memory_memory_hybrid_search
  arguments: {"query":"<topic>","project":"<current-project>"}

Set RECALL_PROJECT only when the Recall project name differs from the current
directory name. This executable prints guidance and exits with status 2.`);
  process.exit(0);
}

const request = {
  tool: "recall_memory_memory_hybrid_search",
  arguments: {
    query: query || "recent work",
    project: projectName(),
    limit: 10,
  },
  executed: false,
  reason: "Recall is an MCP tool and has no supported subprocess adapter.",
};

if (json) {
  console.error(JSON.stringify(request));
} else {
  console.error("ContextSearch no longer reads retired LifeOS state.");
  console.error(`Use ${request.tool} with:`);
  console.error(JSON.stringify(request.arguments, null, 2));
  console.error("No search was executed by this compatibility shim.");
}
process.exit(2);
