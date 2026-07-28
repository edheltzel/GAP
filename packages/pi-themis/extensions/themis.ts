import type { AgentMessage } from "@earendil-works/pi-agent-core";
import type { AssistantMessage, TextContent } from "@earendil-works/pi-ai";
import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

const STATE_ENTRY = "themis-state";
const VOICE_URL = process.env.THEMIS_VOICE_URL ?? "http://localhost:8888/notify";
const VOICE_ID = process.env.THEMIS_VOICE_ID ?? "themis";

interface ThemisState {
	active: boolean;
	item?: string;
	startedAt?: number;
}

function isAssistantMessage(message: AgentMessage): message is AssistantMessage {
	return message.role === "assistant" && Array.isArray(message.content);
}

function textFromAssistant(message: AssistantMessage): string {
	return message.content
		.filter((block): block is TextContent => block.type === "text")
		.map((block) => block.text)
		.join("\n")
		.trim();
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

function guardedPathFromInput(toolName: string, input: Record<string, unknown>): string | undefined {
	if (toolName === "write" || toolName === "read") return input.path as string | undefined;
	if (toolName === "edit") return input.path as string | undefined;
	return undefined;
}

function shouldBlockBash(command: string): string | undefined {
	const blocked: Array<[RegExp, string]> = [
		[/\bgit\s+(commit|push|reset|clean|merge|rebase|checkout|switch)\b/i, "Themis may inspect git state but may not mutate repository history or branches."],
		[/\b(?:npm|pnpm|yarn|bun)\s+(?:install|add|remove|update|upgrade|dedupe)\b/i, "Themis may not change dependencies."],
		[/\brm\s+-(?:r|f|rf|fr)\b/i, "Themis may not perform destructive filesystem operations."],
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
## Themis Mode — Pi-native Arbiter of Done

You are Themis, a Pi-native delegating project manager for this session. You coordinate, sequence, validate, and document. You do not implement code yourself.

### Operating contract
- Report directly to Ed. If a decision belongs to Ed, ask with \`better_ask_user\` when available; otherwise ask one concise question in chat.
- Never program directly. You may read/explore freely and may author markdown or artifacts under \`.agents/**\`. Code, dependency, branch, and commit changes must be delegated.
- Use Pi's strengths instead of Claude Code hooks: extension commands, tool guards, \`subagent\` delegation with acceptance contracts, loaded skills, Recall/memory tools, codegraph, and package-installed tools.
- Before delegating with \`subagent\`, inspect available agents with the subagent list action. Give implementation agents explicit acceptance criteria and verification commands.
- Use RedTeam/adversarial validation for material claims before accepting completion. A worker's assertion is not evidence.
- Ground codebase claims in current files, commands, codegraph, tests, or cited tool output. Do not guess when a tool can answer.
- Own project-management documentation. Plans and handoffs are repository knowledge: they live committed under \`docs/plans/\` and \`docs/handoffs/\` (archives under \`docs/plans/archive/\` and \`docs/handoffs/archive/\`, supporting evidence under \`docs/plans/support/<topic>/\`). \`.agents/atlas/\` holds disposable local artifacts only — briefs, scratch reports, worktrees — and is typically gitignored; promote anything durable into \`docs/\`. A project's own \`AGENTS.md\` wins if it says otherwise. Keep stale docs from accumulating.
- Use GitHub Projects / issue-tracker workflows when converting plans or handoffs into trackable work. Search first to avoid duplicates.
- Prefer herdr when already running inside HERDR_ENV for visible worker/reviewer panes, but Pi subagents are the portable default delegation surface.
- Host harness is the default: dispatched workers, reviewers, and explorers run on Pi (subagents or Pi-driven herdr panes) unless Ed approves otherwise.
- Cross-vendor bridge check-in (mandatory, once per session, before the first dispatch): determine whether this Pi session has a cross-vendor plugin/extension for reaching other model vendors. If it does, ask Ed whether to use it or a native external harness (claude, omp, codex, …); if it does not, ask Ed which external harness serves as the session's confirmed bridge. Re-ask only if Ed changes direction.
- Reviewer dispatches must include a cross-vendor review pass through the confirmed bridge. When dispatching multiple explorers, mix model families: at least one Pi-native explorer and at least one routed through the confirmed bridge.
- Never launch an agent on a non-host harness (claude, omp, codex CLI, …) without Ed's confirmation via the bridge check-in.
- When dispatching through herdr, label every tab that hosts an agent. Issue-linked work uses \`GH#<issue>-<role>\` (e.g. \`GH#21-worker\`) — set via \`herdr tab create --label\` or \`herdr tab rename\` right after a split lands the agent. No issue number → short descriptive label; never leave an agent on a default numbered tab.
- Use the \`themis_notify\` tool for important spoken status updates. The extension also attempts to speak your final status through the local voice server on port 8888.

### Required per-turn status shape
End every substantive Themis turn with a concise status block:
Phase: <phase>
Completed: <what changed or was verified>
In flight: <active workers/reviews/docs, or none>
Blocked: <decision/waiting item, or none>
Next: <the next concrete action>

### First action when activated
Read the nearest AGENTS.md and CLAUDE.md if present, confirm the relevant Pi tools/skills available for this repo, then either perform the activation handshake or proceed with Ed's supplied item after surfacing the dispatch plan.${itemText}
`;
}

function buildKickoff(item?: string): string {
	if (item?.trim()) {
		return `Activate Themis mode for Pi and coordinate this item:\n\n${item.trim()}\n\nFirst read the nearest AGENTS.md and CLAUDE.md if present, confirm the relevant Pi tools/skills, then surface the dispatch/validation plan before delegating.`;
	}
	return "Activate Themis mode for Pi. Read the nearest AGENTS.md and CLAUDE.md if present, confirm relevant Pi tools/skills, introduce yourself as Themis, and wait for Ed's next item.";
}

function extractSpokenStatus(messages: AgentMessage[]): string | undefined {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const message = messages[i];
		if (!isAssistantMessage(message)) continue;
		const text = textFromAssistant(message);
		if (!text) continue;

		const statusLine = text
			.split("\n")
			.map((line) => line.replace(/^[-*#>\s]+/, "").trim())
			.reverse()
			.find((line) => /^(Phase|Completed|Blocked|Next):/i.test(line));

		const chosen = statusLine ?? text.split("\n").find((line) => line.trim().length > 0);
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
		ctx.ui.notify("Themis mode enabled. Markdown/.agents authoring is allowed; code edits are guarded.", "info");
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
		description: "Speak a short Themis status line through the local Pi/Atlas voice server.",
		parameters: Type.Object({
			message: Type.String({ description: "Short spoken status line, ideally under 16 words." }),
			title: Type.Optional(Type.String({ description: "Optional notification title. Defaults to Themis." })),
		}),
		async execute(_toolCallId, params) {
			if (!state.active) {
				return { content: [{ type: "text", text: "Themis mode is not active." }], details: { active: false } };
			}
			const ok = await speak(params.message, params.title ?? "Themis");
			return {
				content: [{ type: "text", text: ok ? "Spoken." : "Voice server unavailable or rejected the notification." }],
				details: { ok, url: VOICE_URL, voiceId: VOICE_ID },
			};
		},
	});

	pi.registerCommand("themis", {
		description: "Activate/deactivate the Pi-native Themis project-manager persona",
		getArgumentCompletions: (prefix) => {
			const commands = ["status", "off", "stop", "disable"];
			const matches = commands.filter((command) => command.startsWith(prefix));
			return matches.length ? matches.map((value) => ({ value, label: value })) : null;
		},
		handler: async (args, ctx) => {
			const trimmed = args.trim();
			const lowered = trimmed.toLowerCase();

			if (["off", "stop", "disable"].includes(lowered)) {
				deactivate(ctx);
				return;
			}

			if (lowered === "status") {
				ctx.ui.notify(
					state.active
						? `Themis active${state.item ? ` for: ${state.item}` : ""}`
						: "Themis mode is not active.",
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
		const entries = ctx.sessionManager.getEntries();
		const lastState = entries
			.filter((entry: { type: string; customType?: string }) => entry.type === "custom" && entry.customType === STATE_ENTRY)
			.pop() as { data?: ThemisState } | undefined;
		if (lastState?.data) state = lastState.data;
		setStatus(ctx);
	});

	pi.on("before_agent_start", async (event) => {
		if (!state.active) return undefined;
		return { systemPrompt: `${event.systemPrompt}\n\n${buildSystemPrompt(state.item)}` };
	});

	pi.on("tool_call", async (event) => {
		if (!state.active) return undefined;

		if (event.toolName === "write" || event.toolName === "edit") {
			const path = guardedPathFromInput(event.toolName, event.input as Record<string, unknown>);
			if (!isMarkdownOrAgentsPath(path)) {
				return {
					block: true,
					reason: `Themis mode blocks direct code edits. Delegate implementation; only markdown and .agents/** artifacts may be changed. Path: ${path ?? "unknown"}`,
				};
			}
		}

		if (event.toolName === "bash") {
			const command = String((event.input as Record<string, unknown>).command ?? "");
			const reason = shouldBlockBash(command);
			if (reason) return { block: true, reason: `Themis mode blocked this shell command: ${reason}` };
		}

		return undefined;
	});

	pi.on("agent_end", async (event) => {
		if (!state.active) return;
		const status = extractSpokenStatus(event.messages as AgentMessage[]);
		if (status) void speak(status);
	});
}
