export const THINKING_LEVELS = ["off", "minimal", "low", "medium", "high", "xhigh", "max"] as const;

export type PromptThinkingLevel = (typeof THINKING_LEVELS)[number];

interface PromptTemplateConfigBase {
	readonly description?: string;
	readonly modelSpecs: readonly string[];
	readonly thinking?: PromptThinkingLevel;
	readonly skill?: string;
	readonly restore: boolean;
}

export type PromptTemplateConfig = PromptTemplateConfigBase & (
	| { readonly subagent?: undefined; readonly inheritContext: false }
	| { readonly subagent: true | string; readonly inheritContext: boolean }
);

export type NormalizedPromptFrontmatter =
	| { readonly kind: "native" }
	| { readonly kind: "invalid"; readonly diagnostics: readonly string[] }
	| { readonly kind: "profiled"; readonly config: PromptTemplateConfig };

const RUNTIME_FIELDS = [
	"model",
	"thinking",
	"skill",
	"restore",
	"subagent",
	"inheritContext",
] as const;
const SKILL_NAME_PATTERN = /^[A-Za-z0-9._-]+$/;

export function isValidSkillName(value: string): boolean {
	return value !== "." && value !== ".." && SKILL_NAME_PATTERN.test(value);
}
const INLINE_ARG_SHELL_PATTERN = /\$(?:ARGUMENTS|@(?:\[\d+(?::\d*)?\])?|\d+)|\$\{@:\d+(?::\d+)?\}/;
const INLINE_ARG_TEMPLATE_PATTERN = /\{\{[\s\S]*?(?:\b(?:arguments|ARGUMENTS|args)\b|\barg\s+[^}]+)[\s\S]*?\}\}/;
const PLACEHOLDER_REPLACEMENT_PATTERN = /\$\{@:(\d+)(?::(\d+))?\}|\$@\[(\d+)(?::(\d*))?\]|\$ARGUMENTS|\$@|\$(\d+)/g;

function normalizeModels(value: unknown, diagnostics: string[]): string[] {
	if (value === undefined) return [];

	let rawValues: string[];
	if (typeof value === "string") {
		rawValues = [value];
	} else if (Array.isArray(value) && value.every((item): item is string => typeof item === "string")) {
		rawValues = value;
	} else {
		diagnostics.push("model must be a string or an array of strings");
		return [];
	}

	const specs: string[] = [];
	for (const rawValue of rawValues) {
		for (const valuePart of rawValue.split(",")) {
			const spec = valuePart.trim();
			if (spec) specs.push(spec);
		}
	}
	if (specs.length === 0) diagnostics.push("model must include at least one model selector");
	return specs;
}

function normalizeThinking(value: unknown, diagnostics: string[]): PromptThinkingLevel | undefined {
	if (value === undefined) return undefined;
	if (typeof value === "string" && THINKING_LEVELS.includes(value as PromptThinkingLevel)) {
		return value as PromptThinkingLevel;
	}
	diagnostics.push("thinking must be one of off, minimal, low, medium, high, xhigh, or max");
	return undefined;
}

function normalizeRestore(value: unknown, diagnostics: string[]): boolean {
	if (value === undefined) return true;
	if (typeof value === "boolean") return value;
	diagnostics.push("restore must be a boolean");
	return true;
}

function normalizeSkill(value: unknown, diagnostics: string[]): string | undefined {
	if (value === undefined) return undefined;
	if (typeof value !== "string") {
		diagnostics.push("skill must be a non-empty name containing only letters, numbers, dots, underscores, or hyphens");
		return undefined;
	}

	const skillName = value.trim().replace(/^skill:/, "").trim();
	if (skillName && isValidSkillName(skillName)) return skillName;
	diagnostics.push("skill must be a non-empty name containing only letters, numbers, dots, underscores, or hyphens");
	return undefined;
}

function normalizeDescription(
	frontmatter: Record<string, unknown>,
	diagnostics: string[],
): string | undefined {
	if (!Object.hasOwn(frontmatter, "description")) return undefined;
	const value = frontmatter.description;
	if (typeof value === "string" && value.trim()) return value.trim();
	diagnostics.push("description must be a non-empty string");
	return undefined;
}

function normalizeSubagent(value: unknown, diagnostics: string[]): true | string | undefined {
	if (value === undefined || value === false) return undefined;
	if (value === true) return true;
	if (typeof value === "string" && value.trim()) return value.trim();
	diagnostics.push("subagent must be true, false, or a non-empty agent name");
	return undefined;
}

function normalizeInheritContext(
	value: unknown,
	subagent: true | string | undefined,
	diagnostics: string[],
): boolean {
	if (value === undefined) return false;
	if (typeof value !== "boolean") {
		diagnostics.push("inheritContext must be a boolean");
		return false;
	}
	if (value && !subagent) diagnostics.push("inheritContext requires subagent");
	return value;
}

export function normalizePromptFrontmatter(frontmatter: Record<string, unknown>): NormalizedPromptFrontmatter {
	if (!RUNTIME_FIELDS.some((field) => Object.hasOwn(frontmatter, field))) {
		return { kind: "native" };
	}

	const diagnostics: string[] = [];
	const modelSpecs = normalizeModels(frontmatter.model, diagnostics);
	const thinking = normalizeThinking(frontmatter.thinking, diagnostics);
	const restore = normalizeRestore(frontmatter.restore, diagnostics);
	const skill = normalizeSkill(frontmatter.skill, diagnostics);
	const subagent = normalizeSubagent(frontmatter.subagent, diagnostics);
	const inheritContext = normalizeInheritContext(frontmatter.inheritContext, subagent, diagnostics);
	const description = normalizeDescription(frontmatter, diagnostics);

	if (diagnostics.length > 0) return { kind: "invalid", diagnostics };
	const taskConfig = subagent
		? { subagent, inheritContext }
		: { inheritContext: false as const };
	return {
		kind: "profiled",
		config: {
			...(description ? { description } : {}),
			modelSpecs,
			...(thinking ? { thinking } : {}),
			...(skill ? { skill } : {}),
			restore,
			...taskConfig,
		},
	};
}

export function parseCommandArgs(argsString: string): string[] {
	const args: string[] = [];
	let current = "";
	let quote: string | undefined;

	for (const char of argsString) {
		if (quote) {
			if (char === quote) quote = undefined;
			else current += char;
		} else if (char === '"' || char === "'") {
			quote = char;
		} else if (char === " " || char === "\t") {
			if (current) {
				args.push(current);
				current = "";
			}
		} else {
			current += char;
		}
	}
	if (current) args.push(current);
	return args;
}

function renderSlice(args: string[], startText: string, lengthText?: string): string {
	const start = Number.parseInt(startText, 10);
	if (!Number.isFinite(start) || start < 1 || start > args.length) return "";
	if (lengthText === undefined || lengthText === "") return args.slice(start - 1).join(" ");
	const length = Number.parseInt(lengthText, 10);
	if (!Number.isFinite(length) || length < 1) return "";
	return args.slice(start - 1, start - 1 + length).join(" ");
}

export function templateUsesInlineArgPlaceholders(content: string): boolean {
	return INLINE_ARG_SHELL_PATTERN.test(content) || INLINE_ARG_TEMPLATE_PATTERN.test(content);
}

export function renderPromptBody(content: string, args: string[]): string {
	const allArgs = args.join(" ");
	return content.replace(
		PLACEHOLDER_REPLACEMENT_PATTERN,
		(match, piStart, piLength, ompStart, ompLength, position) => {
			if (piStart) return renderSlice(args, piStart, piLength);
			if (ompStart) return renderSlice(args, ompStart, ompLength);
			if (position) return args[Number.parseInt(position, 10) - 1] ?? "";
			if (match === "$ARGUMENTS" || match === "$@") return allArgs;
			return match;
		},
	);
}

export function appendPromptArgs(
	rendered: string,
	argsText: string,
	usesInlineArgPlaceholders: boolean,
): string {
	if (argsText.length === 0 || usesInlineArgPlaceholders) return rendered;
	if (rendered.length === 0) return argsText;
	return `${rendered}\n\n${argsText}`;
}

export function buildDelegationGuidance(
	templateName: string,
	subagent: true | string,
	inheritContext: boolean,
): string {
	const target = subagent === true
		? "use its configured default agent"
		: `delegate through the native \`task\` tool to the \`${subagent}\` agent`;
	const contextPolicy = inheritContext
		? "include the relevant current-session context in the delegated task"
		: "do not add prior conversation context beyond the rendered prompt";

	return [
		`The user invoked the OMP prompt template \`/${templateName}\` with native task delegation enabled.`,
		`You MUST ${target}. Do not perform the task directly.`,
		`When constructing the task, ${contextPolicy}.`,
		"Return the delegated agent's result to the user after it completes.",
	].join("\n");
}
