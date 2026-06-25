import type { ExtensionAPI, ExtensionContext } from "@oh-my-pi/pi-coding-agent";

declare const process: { env: Record<string, string | undefined> };

const STATE_ENTRY = "themis-state";
const VOICE_URL = process.env.THEMIS_VOICE_URL ?? "http://localhost:8888/notify";
const VOICE_ID = process.env.THEMIS_VOICE_ID ?? "themis";
const DEACTIVATE_ALIASES = ["off", "stop", "disable"];

interface ThemisState {
  active: boolean;
  item?: string;
  startedAt?: number;
}

type SessionEntry = {
  type?: string;
  customType?: string;
  data?: unknown;
};

type AssistantLike = {
  role?: string;
  content?: unknown;
};

type TextBlock = {
  type?: string;
  text?: string;
};

function isThemisState(value: unknown): value is ThemisState {
  if (!value || typeof value !== "object") return false;
  const candidate = value as Partial<ThemisState>;
  return typeof candidate.active === "boolean";
}

function isMarkdownOrAgentsPath(path: string | undefined): boolean {
  if (!path) return false;
  const normalized = path.replace(/\\/g, "/");
  return (
    normalized.endsWith(".md") ||
    normalized.endsWith(".mdx") ||
    normalized.includes("/.agents/") ||
    normalized.startsWith(".agents/") ||
    normalized === ".agents"
  );
}

function pathsFromEditInput(input: Record<string, unknown>): string[] {
  const raw = typeof input.input === "string" ? input.input : "";
  const paths: string[] = [];
  const headerPattern = /^\[([^#\]\n]+)#[0-9A-F]{4}\]/gm;
  for (const match of raw.matchAll(headerPattern)) {
    paths.push(match[1]);
  }
  return paths;
}

function shouldBlockDirectMutation(toolName: string, input: Record<string, unknown>): string | undefined {
  if (toolName === "write") {
    const path = typeof input.path === "string" ? input.path : undefined;
    return isMarkdownOrAgentsPath(path)
      ? undefined
      : `Themis mode blocks direct non-markdown writes. Delegate implementation. Path: ${path ?? "unknown"}`;
  }

  if (toolName === "edit") {
    const paths = pathsFromEditInput(input);
    if (paths.length === 0) return "Themis mode blocks edit calls whose target path cannot be proven markdown or .agents/**.";
    const blocked = paths.find((path) => !isMarkdownOrAgentsPath(path));
    return blocked ? `Themis mode blocks direct code edits. Delegate implementation. Path: ${blocked}` : undefined;
  }

  if (toolName === "ast_edit") {
    return "Themis mode blocks structural code edits. Delegate implementation to a worker.";
  }

  if (toolName === "lsp") {
    const action = typeof input.action === "string" ? input.action : "";
    const applies = input.apply === true;
    if (action === "rename" || action === "rename_file" || (action === "code_actions" && applies)) {
      return `Themis mode blocks mutating LSP action '${action}'. Delegate implementation.`;
    }
  }

  return undefined;
}

function shouldBlockBash(command: string): string | undefined {
  const blocked: Array<[RegExp, string]> = [
    [/\bgit\s+(commit|push|reset|clean|merge|rebase|checkout|switch|branch\s+-D)\b/i, "Themis may inspect git state but may not mutate repository history or branches."],
    [/\b(?:npm|pnpm|yarn|bun)\s+(?:install|add|remove|update|upgrade|dedupe)\b/i, "Themis may not change dependencies."],
    [/\brm\s+-(?:[a-z]*r[a-z]*f|[a-z]*f[a-z]*r)\b/i, "Themis may not perform destructive filesystem operations."],
    [/\b(?:sed|perl|ruby|python|node)\b[^\n;|&]*\s-i\b/i, "Themis may not mutate files through shell one-liners."],
    [/(^|\s)(?:>|>>)\s*(?![^\s;|&]*(?:\.mdx?|\.agents\/))/i, "Themis shell redirection is limited to markdown or .agents artifacts."],
  ];

  for (const [pattern, reason] of blocked) {
    if (pattern.test(command)) return reason;
  }
  return undefined;
}

function buildSystemPrompt(item?: string): string {
  const itemText = item?.trim() ? `\n\nCurrent Themis item/context:\n${item.trim()}` : "";
  return `
## Themis Mode — OMP-native Arbiter of Done

You are Themis, an OMP-native delegating project manager for this session. You coordinate, sequence, validate, and document. You do not implement code yourself.

### Operating contract
- Report directly to Ed. If a decision belongs to Ed, ask one concise question and wait.
- Never program directly. You may read/explore freely and may author markdown or artifacts under \`.agents/**\`. Code, dependency, branch, and commit changes must be delegated.
- Use OMP's strengths instead of Claude Code mechanics: the \`task\` tool for subagents, \`job\` for waits, \`irc\` for agent coordination, \`todo\` for orchestration state, \`lsp\`/\`debug\` for code intelligence, \`github\`/\`issue://\`/\`pr://\` for tracker work, Recall/memory tools, and package-installed skills/tools.
- Before delegating, create explicit worker/reviewer/explorer briefs with target files, acceptance criteria, and verification commands. Subagents do not run project-wide gates; you run the final gate after merging their work.
- Use adversarial validation before accepting material claims. A worker's assertion is not evidence.
- Ground codebase claims in current files, OMP tool output, codegraph when available, tests, or cited command output. Do not guess when a tool can answer.
- Own project-management documentation under the Documentation Ownership rules below.
- Prefer herdr only when already running inside \`HERDR_ENV\` for visible panes. OMP \`task\` subagents are the portable default delegation surface.
- Use the \`themis_notify\` tool for important spoken status updates. The extension also attempts to speak your final status through the local voice server on port 8888.

### Documentation Ownership (DOX Framework)
- Plans, specs, and handoffs live under \`.agents/atlas/\` (\`plans/\`, \`plans/archive/\`, \`handoffs/\`, \`handoffs/archive/\`, \`artifacts/\`), never in \`docs/\`.
- After the start of a handoff, immediately move it to \`handoffs/archive/\`.
- Keep stale docs from accumulating.
- Use GitHub Projects / issue-tracker workflows when converting plans or handoffs into trackable work. Search first to avoid duplicates.

### Required per-turn status shape
End every substantive Themis turn with a concise status block:
Phase: <phase>
Completed: <what changed or was verified>
In flight: <active workers/reviews/docs, or none>
Blocked: <decision/waiting item, or none>
Next: <the next concrete action>

### First action when activated
Read the already-loaded project context first. If extra project context is needed, read the nearest AGENTS.md/CLAUDE.md-style files that OMP reports in /extensions. Confirm relevant OMP tools/skills for this repo, then either perform the activation handshake or proceed with Ed's supplied item after surfacing the dispatch/validation plan.${itemText}
`;
}


function buildKickoff(item?: string): string {
  if (item?.trim()) {
    return `Activate Themis mode for OMP and coordinate this item:\n\n${item.trim()}\n\nConfirm the relevant OMP tools/skills, then surface the dispatch and validation plan before delegating.`;
  }
  return "Activate Themis mode for OMP. Confirm relevant OMP tools/skills, introduce yourself as Themis, and wait for Ed's next item.";
}

function textFromAssistant(message: AssistantLike): string {
  if (message.role !== "assistant" || !Array.isArray(message.content)) return "";
  return message.content
    .filter((block): block is TextBlock => typeof block === "object" && block !== null && (block as TextBlock).type === "text")
    .map((block) => block.text ?? "")
    .join("\n")
    .trim();
}

function extractSpokenStatus(messages: AssistantLike[] | undefined): string | undefined {
  if (!messages) return undefined;
  for (let i = messages.length - 1; i >= 0; i -= 1) {
    const text = textFromAssistant(messages[i]);
    if (!text) continue;

    const lines = text.split("\n");
    const statusLine = lines
      .map((line) => line.replace(/^[-*#>\s]+/, "").trim())
      .reverse()
      .find((line) => /^(Phase|Completed|Blocked|Next):/i.test(line));

    const chosen = statusLine ?? lines.find((line) => line.trim().length > 0);
    if (!chosen) continue;
    return chosen.replace(/[*_`#]/g, "").slice(0, 180);
  }
  return undefined;
}

async function speak(message: string, title = "Themis"): Promise<boolean> {
  try {
    const response = await fetch(VOICE_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        message,
        voice_id: VOICE_ID,
        title,
        voice_enabled: true,
      }),
    });
    return response.ok;
  } catch {
    return false;
  }
}

export default function themisExtension(pi: ExtensionAPI): void {
  const { Type } = pi.typebox;
  let state: ThemisState = { active: false };

  function persist(): void {
    pi.appendEntry(STATE_ENTRY, state);
  }

  function setStatus(ctx: ExtensionContext): void {
    ctx.ui.setStatus("themis", state.active ? ctx.ui.theme.fg("accent", "⚖ Themis") : undefined);
  }

  function activate(ctx: ExtensionContext, item?: string): void {
    state = { active: true, item: item?.trim() || undefined, startedAt: Date.now() };
    persist();
    setStatus(ctx);
    ctx.ui.notify("Themis mode enabled. Markdown/.agents authoring is allowed; implementation edits are guarded.", "info");
    void speak("Themis is active and ready to arbitrate done.");
  }

  function deactivate(ctx: ExtensionContext): void {
    state = { ...state, active: false };
    persist();
    setStatus(ctx);
    ctx.ui.notify("Themis mode disabled.", "info");
    void speak("Themis mode is disabled.");
  }

  pi.registerTool({
    name: "themis_notify",
    label: "Themis Voice",
    description: "Speak a short Themis status line through the local Atlas voice server.",
    parameters: Type.Object({
      message: Type.String({ description: "Short spoken status line, ideally under 16 words." }),
      title: Type.Optional(Type.String({ description: "Optional notification title. Defaults to Themis." })),
    }),
    async execute(_toolCallId, params) {
      if (!state.active) {
        return {
          content: [{ type: "text", text: "Themis mode is not active." }],
          details: { active: false, ok: false, url: VOICE_URL, voiceId: VOICE_ID },
        };
      }
      const ok = await speak(params.message, params.title ?? "Themis");
      return {
        content: [{ type: "text", text: ok ? "Spoken." : "Voice server unavailable or rejected the notification." }],
        details: { active: true, ok, url: VOICE_URL, voiceId: VOICE_ID },
      };
    },
  });

  pi.registerCommand("themis", {
    description: "Activate/deactivate the OMP-native Themis project-manager persona",
    getArgumentCompletions: (prefix) => {
      const commands = ["status", ...DEACTIVATE_ALIASES];
      const matches = commands.filter((command) => command.startsWith(prefix));
      return matches.length ? matches.map((value) => ({ value, label: value })) : null;
    },
    handler: async (args, ctx) => {
      const trimmed = args.trim();
      const lowered = trimmed.toLowerCase();

      if (DEACTIVATE_ALIASES.includes(lowered)) {
        deactivate(ctx);
        return;
      }

      if (lowered === "status") {
        ctx.ui.notify(
          state.active ? `Themis active${state.item ? ` for: ${state.item}` : ""}` : "Themis mode is not active.",
          "info",
        );
        setStatus(ctx);
        return;
      }

      activate(ctx, trimmed);
      pi.sendUserMessage(buildKickoff(trimmed), ctx.isIdle() ? undefined : { deliverAs: "followUp" });
    },
  });

  pi.on("session_start", async (_event, ctx) => {
    const entries = ctx.sessionManager.getEntries() as SessionEntry[];
    const lastState = entries
      .filter((entry) => entry.type === "custom" && entry.customType === STATE_ENTRY)
      .pop();
    if (isThemisState(lastState?.data)) state = lastState.data;
    setStatus(ctx);
  });

  pi.on("before_agent_start", async (event) => {
    if (!state.active) return undefined;
    return { systemPrompt: [...event.systemPrompt, buildSystemPrompt(state.item)] };
  });

  pi.on("tool_call", async (event) => {
    if (!state.active) return undefined;

    const input = event.input as Record<string, unknown>;
    const mutationReason = shouldBlockDirectMutation(event.toolName, input);
    if (mutationReason) return { block: true, reason: mutationReason };

    if (event.toolName === "bash") {
      const command = String(input.command ?? "");
      const reason = shouldBlockBash(command);
      if (reason) return { block: true, reason: `Themis mode blocked this shell command: ${reason}` };
    }

    return undefined;
  });

  pi.on("agent_end", async (event) => {
    if (!state.active) return;
    const status = extractSpokenStatus(event.messages as AssistantLike[] | undefined);
    if (status) void speak(status);
  });
}
