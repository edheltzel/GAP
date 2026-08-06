#!/usr/bin/env bun
/// <reference types="node" />

/**
 * Load one self-contained role context and optionally append a complete task.
 * This utility prepares prompt text. It does not select a model or dispatch a worker.
 */

import { existsSync, readFileSync, readdirSync } from "fs";
import { homedir } from "os";
import { join } from "path";
import { pathToFileURL } from "url";

interface AgentContext {
  agentType: string;
  contextContent: string;
}

export class AgentContextLoader {
  private agentsDir: string;

  constructor() {
    this.agentsDir = join(homedir(), ".claude", "skills", "Agents");
  }

  loadContext(agentType: string): AgentContext {
    const contextPath = join(this.agentsDir, `${agentType}Context.md`);

    if (!existsSync(contextPath)) {
      throw new Error(
        `Context file not found for agent type: ${agentType}\nExpected at: ${contextPath}`
      );
    }

    return {
      agentType,
      contextContent: readFileSync(contextPath, "utf-8"),
    };
  }

  getAvailableAgents(): string[] {
    if (!existsSync(this.agentsDir)) {
      return [];
    }

    return readdirSync(this.agentsDir)
      .filter((file) => file.endsWith("Context.md"))
      .map((file) => file.replace("Context.md", ""))
      .sort();
  }

  generateEnrichedPrompt(agentType: string, taskDescription: string): string {
    const context = this.loadContext(agentType);

    return `${context.contextContent}

---

## Current Task

${taskDescription}`;
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const args = process.argv.slice(2);

  if (args.length === 0) {
    console.log("Usage: LoadAgentContext.ts <agentType> [taskDescription]");
    console.log("\nAvailable role contexts:");
    const loader = new AgentContextLoader();
    loader.getAvailableAgents().forEach((agent) => console.log(`  - ${agent}`));
    process.exit(1);
  }

  const [agentType, ...taskParts] = args;
  const taskDescription = taskParts.join(" ");

  try {
    const loader = new AgentContextLoader();

    if (taskDescription) {
      console.log(`\n=== Enriched Prompt for ${agentType} Role ===\n`);
      console.log(loader.generateEnrichedPrompt(agentType, taskDescription));
    } else {
      const context = loader.loadContext(agentType);
      console.log(`\n=== Context for ${context.agentType} Role ===\n`);
      console.log(context.contextContent);
    }
  } catch (error) {
    console.error(`Error: ${error}`);
    process.exit(1);
  }
}

export default AgentContextLoader;
