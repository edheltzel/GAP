package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type agentEnv struct {
	Home, ConfigHome, Cwd string
	Getenv                func(string) string
}

type agentDef struct {
	ID, Display, Dir                                                                 string
	GlobalDir                                                                        func(agentEnv) string
	Detect                                                                           func(agentEnv) bool
	HideInUniversalList, HideInUniversalPrompt, GlobalSkipsCanonical, MakeProjectDir bool
}

func currentAgentEnv() agentEnv {
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = filepath.Join(home, ".config")
	}
	return agentEnv{Home: home, ConfigHome: config, Cwd: cwd, Getenv: os.Getenv}
}

func (e agentEnv) envHome(key, fallback string) string {
	v := strings.TrimSpace(e.Getenv(key))
	if v == "" {
		return filepath.Join(e.Home, fallback)
	}
	if abs, err := filepath.Abs(v); err == nil {
		return abs
	}
	return v
}

func pathExists(path string) func(agentEnv) bool {
	return func(agentEnv) bool { return fileExists(path) }
}

func existsUnder(base func(agentEnv) string, rels []string) func(agentEnv) bool {
	return func(e agentEnv) bool {
		return slices.ContainsFunc(rels, func(rel string) bool { return fileExists(filepath.Join(base(e), rel)) })
	}
}

func homeExists(rels ...string) func(agentEnv) bool {
	return existsUnder(func(e agentEnv) string { return e.Home }, rels)
}

func configExists(rels ...string) func(agentEnv) bool {
	return existsUnder(func(e agentEnv) string { return e.ConfigHome }, rels)
}

func cwdExists(rels ...string) func(agentEnv) bool {
	return existsUnder(func(e agentEnv) string { return e.Cwd }, rels)
}

func envHomeExists(key, fallback string) func(agentEnv) bool {
	return func(e agentEnv) bool { return fileExists(e.envHome(key, fallback)) }
}

func anyExists(checks ...func(agentEnv) bool) func(agentEnv) bool {
	return func(e agentEnv) bool {
		return slices.ContainsFunc(checks, func(check func(agentEnv) bool) bool { return check(e) })
	}
}

func inHome(rel string) func(agentEnv) string {
	return func(e agentEnv) string { return filepath.Join(e.Home, rel) }
}

func inConfig(rel string) func(agentEnv) string {
	return func(e agentEnv) string { return filepath.Join(e.ConfigHome, rel) }
}

func inEnvHome(key, fallback, rel string) func(agentEnv) string {
	return func(e agentEnv) string { return filepath.Join(e.envHome(key, fallback), rel) }
}

func openClawGlobalDir(e agentEnv) string {
	for _, dir := range []string{".openclaw", ".clawdbot", ".moltbot"} {
		if fileExists(filepath.Join(e.Home, dir)) {
			return filepath.Join(e.Home, dir, "skills")
		}
	}
	return filepath.Join(e.Home, ".openclaw", "skills")
}

func eveInstalled(e agentEnv) bool {
	if !fileExists(filepath.Join(e.Cwd, "agent")) {
		return false
	}
	b, err := os.ReadFile(filepath.Join(e.Cwd, "package.json"))
	if err != nil {
		return false
	}
	var pkg struct {
		Dependencies    map[string]any `json:"dependencies"`
		DevDependencies map[string]any `json:"devDependencies"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return false
	}
	return jsTruthy(pkg.Dependencies["eve"]) || jsTruthy(pkg.DevDependencies["eve"])
}

func zedInstalled(e agentEnv) bool {
	if fileExists(filepath.Join(e.ConfigHome, "zed")) {
		return true
	}
	if appData := strings.TrimSpace(e.Getenv("APPDATA")); appData != "" && fileExists(filepath.Join(appData, "Zed")) {
		return true
	}
	flatpak := strings.TrimSpace(e.Getenv("FLATPAK_XDG_CONFIG_HOME"))
	return flatpak != "" && fileExists(filepath.Join(flatpak, "zed"))
}

func neverInstalled(agentEnv) bool { return false }

var agentTable = []agentDef{
	{ID: "aider-desk", Display: "AiderDesk", Dir: ".aider-desk/skills", GlobalDir: inHome(".aider-desk/skills"), Detect: homeExists(".aider-desk")},
	{ID: "amp", Display: "Amp", Dir: ".agents/skills", GlobalDir: inConfig("agents/skills"), Detect: configExists("amp")},
	{ID: "antigravity", Display: "Antigravity", Dir: ".agents/skills", GlobalDir: inHome(".gemini/config/skills"), Detect: homeExists(".gemini/antigravity"), HideInUniversalPrompt: true, GlobalSkipsCanonical: true},
	{ID: "antigravity-cli", Display: "Antigravity CLI", Dir: ".agents/skills", GlobalDir: inHome(".gemini/config/skills"), Detect: homeExists(".gemini/antigravity-cli"), HideInUniversalPrompt: true, GlobalSkipsCanonical: true},
	{ID: "astrbot", Display: "AstrBot", Dir: "data/skills", GlobalDir: inHome(".astrbot/data/skills"), Detect: anyExists(cwdExists("data/skills"), homeExists(".astrbot"))},
	{ID: "autohand-code", Display: "Autohand Code CLI", Dir: ".autohand/skills", GlobalDir: inEnvHome("AUTOHAND_HOME", ".autohand", "skills"), Detect: envHomeExists("AUTOHAND_HOME", ".autohand")},
	{ID: "augment", Display: "Augment", Dir: ".augment/skills", GlobalDir: inHome(".augment/skills"), Detect: homeExists(".augment")},
	{ID: "bob", Display: "IBM Bob", Dir: ".bob/skills", GlobalDir: inHome(".bob/skills"), Detect: homeExists(".bob")},
	{ID: "claude-code", Display: "Claude Code", Dir: ".claude/skills", GlobalDir: inEnvHome("CLAUDE_CONFIG_DIR", ".claude", "skills"), Detect: envHomeExists("CLAUDE_CONFIG_DIR", ".claude"), MakeProjectDir: true},
	{ID: "openclaw", Display: "OpenClaw", Dir: "skills", GlobalDir: openClawGlobalDir, Detect: homeExists(".openclaw", ".clawdbot", ".moltbot")},
	{ID: "cline", Display: "Cline", Dir: ".agents/skills", GlobalDir: inHome(".agents/skills"), Detect: homeExists(".cline")},
	{ID: "codearts-agent", Display: "CodeArts Agent", Dir: ".codeartsdoer/skills", GlobalDir: inHome(".codeartsdoer/skills"), Detect: homeExists(".codeartsdoer")},
	{ID: "codebuddy", Display: "CodeBuddy", Dir: ".codebuddy/skills", GlobalDir: inHome(".codebuddy/skills"), Detect: anyExists(cwdExists(".codebuddy"), homeExists(".codebuddy"))},
	{ID: "codemaker", Display: "Codemaker", Dir: ".codemaker/skills", GlobalDir: inHome(".codemaker/skills"), Detect: homeExists(".codemaker")},
	{ID: "codestudio", Display: "Code Studio", Dir: ".codestudio/skills", GlobalDir: inHome(".codestudio/skills"), Detect: homeExists(".codestudio")},
	{ID: "codex", Display: "Codex", Dir: ".agents/skills", GlobalDir: inHome(".agents/skills"), Detect: anyExists(envHomeExists("CODEX_HOME", ".codex"), pathExists("/etc/codex"))},
	{ID: "command-code", Display: "Command Code", Dir: ".commandcode/skills", GlobalDir: inHome(".commandcode/skills"), Detect: homeExists(".commandcode")},
	{ID: "continue", Display: "Continue", Dir: ".continue/skills", GlobalDir: inHome(".continue/skills"), Detect: anyExists(cwdExists(".continue"), homeExists(".continue"))},
	{ID: "cortex", Display: "Cortex Code", Dir: ".cortex/skills", GlobalDir: inHome(".snowflake/cortex/skills"), Detect: homeExists(".snowflake/cortex")},
	{ID: "crush", Display: "Crush", Dir: ".crush/skills", GlobalDir: inHome(".config/crush/skills"), Detect: homeExists(".config/crush")},
	{ID: "cursor", Display: "Cursor", Dir: ".agents/skills", GlobalDir: inHome(".cursor/skills"), Detect: homeExists(".cursor")},
	{ID: "deepagents", Display: "Deep Agents", Dir: ".agents/skills", GlobalDir: inHome(".deepagents/agent/skills"), Detect: homeExists(".deepagents"), HideInUniversalPrompt: true},
	{ID: "devin", Display: "Devin for Terminal", Dir: ".devin/skills", GlobalDir: inConfig("devin/skills"), Detect: configExists("devin")},
	{ID: "dexto", Display: "Dexto", Dir: ".agents/skills", GlobalDir: inHome(".agents/skills"), Detect: homeExists(".dexto"), HideInUniversalPrompt: true},
	{ID: "droid", Display: "Droid", Dir: ".agents/skills", GlobalDir: inHome(".factory/skills"), Detect: homeExists(".factory")},
	{ID: "eve", Display: "Eve", Dir: "agent/skills", Detect: eveInstalled},
	{ID: "firebender", Display: "Firebender", Dir: ".agents/skills", GlobalDir: inHome(".firebender/skills"), Detect: homeExists(".firebender"), HideInUniversalPrompt: true},
	{ID: "forgecode", Display: "ForgeCode", Dir: ".forge/skills", GlobalDir: inHome(".forge/skills"), Detect: homeExists(".forge")},
	{ID: "fx", Display: "fx", Dir: ".agents/skills", GlobalDir: inHome(".agents/skills"), Detect: homeExists(".fx")},
	{ID: "gemini-cli", Display: "Gemini CLI", Dir: ".agents/skills", GlobalDir: inHome(".gemini/skills"), Detect: homeExists(".gemini")},
	{ID: "github-copilot", Display: "GitHub Copilot", Dir: ".agents/skills", GlobalDir: inHome(".copilot/skills"), Detect: homeExists(".copilot")},
	{ID: "goose", Display: "Goose", Dir: ".goose/skills", GlobalDir: inConfig("goose/skills"), Detect: configExists("goose")},
	{ID: "grok", Display: "Grok Build", Dir: ".grok/skills", GlobalDir: inEnvHome("GROK_HOME", ".grok", "skills"), Detect: envHomeExists("GROK_HOME", ".grok")},
	{ID: "hermes-agent", Display: "Hermes Agent", Dir: ".hermes/skills", GlobalDir: inEnvHome("HERMES_HOME", ".hermes", "skills"), Detect: envHomeExists("HERMES_HOME", ".hermes")},
	{ID: "inference-sh", Display: "inference.sh", Dir: ".inferencesh/skills", GlobalDir: inHome(".inferencesh/skills"), Detect: homeExists(".inferencesh")},
	{ID: "jazz", Display: "Jazz", Dir: ".jazz/skills", GlobalDir: inHome(".jazz/skills"), Detect: anyExists(homeExists(".jazz"), cwdExists(".jazz"))},
	{ID: "junie", Display: "Junie", Dir: ".junie/skills", GlobalDir: inHome(".junie/skills"), Detect: homeExists(".junie")},
	{ID: "iflow-cli", Display: "iFlow CLI", Dir: ".iflow/skills", GlobalDir: inHome(".iflow/skills"), Detect: homeExists(".iflow")},
	{ID: "kilo", Display: "Kilo Code", Dir: ".agents/skills", GlobalDir: inHome(".kilo/skills"), Detect: homeExists(".kilo", ".kilocode")},
	{ID: "kimchi", Display: "Kimchi", Dir: ".kimchi/skills", GlobalDir: inHome(".config/kimchi/harness/skills"), Detect: homeExists(".config/kimchi")},
	{ID: "kimi-code-cli", Display: "Kimi Code CLI", Dir: ".agents/skills", GlobalDir: inHome(".agents/skills"), Detect: homeExists(".kimi-code", ".kimi")},
	{ID: "kiro-cli", Display: "Kiro CLI", Dir: ".kiro/skills", GlobalDir: inHome(".kiro/skills"), Detect: homeExists(".kiro")},
	{ID: "kode", Display: "Kode", Dir: ".kode/skills", GlobalDir: inHome(".kode/skills"), Detect: homeExists(".kode")},
	{ID: "lingma", Display: "Lingma", Dir: ".lingma/skills", GlobalDir: inHome(".lingma/skills"), Detect: homeExists(".lingma")},
	{ID: "loaf", Display: "Loaf", Dir: ".agents/skills", GlobalDir: inHome(".agents/skills"), Detect: homeExists(".loaf"), HideInUniversalPrompt: true},
	{ID: "mcpjam", Display: "MCPJam", Dir: ".mcpjam/skills", GlobalDir: inHome(".mcpjam/skills"), Detect: homeExists(".mcpjam")},
	{ID: "minimax-code", Display: "MiniMax Code", Dir: ".minimax/skills", GlobalDir: inHome(".minimax/skills"), Detect: anyExists(homeExists(".minimax"), pathExists("/Applications/MiniMax Code.app"))},
	{ID: "mistral-vibe", Display: "Mistral Vibe", Dir: ".vibe/skills", GlobalDir: inEnvHome("VIBE_HOME", ".vibe", "skills"), Detect: envHomeExists("VIBE_HOME", ".vibe")},
	{ID: "moxby", Display: "Moxby", Dir: ".moxby/skills", GlobalDir: inHome(".moxby/skills"), Detect: homeExists(".moxby")},
	{ID: "mux", Display: "Mux", Dir: ".mux/skills", GlobalDir: inHome(".mux/skills"), Detect: homeExists(".mux")},
	{ID: "opencode", Display: "OpenCode", Dir: ".agents/skills", GlobalDir: inConfig("opencode/skills"), Detect: configExists("opencode")},
	{ID: "openhands", Display: "OpenHands", Dir: ".openhands/skills", GlobalDir: inHome(".openhands/skills"), Detect: homeExists(".openhands")},
	{ID: "ona", Display: "Ona", Dir: ".ona/skills", GlobalDir: inHome(".ona/skills"), Detect: homeExists(".ona")},
	{ID: "pi", Display: "Pi", Dir: ".agents/skills", GlobalDir: inHome(".agents/skills"), Detect: homeExists(".pi/agent")},
	{ID: "posit-assistant", Display: "Posit Assistant", Dir: ".posit/assistant/skills", GlobalDir: inHome(".posit/assistant/skills"), Detect: homeExists(".posit/assistant", ".positai")},
	{ID: "qoder", Display: "Qoder", Dir: ".qoder/skills", GlobalDir: inHome(".qoder/skills"), Detect: homeExists(".qoder")},
	{ID: "qoder-cn", Display: "Qoder CN", Dir: ".qoder/skills", GlobalDir: inHome(".qoder-cn/skills"), Detect: homeExists(".qoder-cn")},
	{ID: "qwen-code", Display: "Qwen Code", Dir: ".qwen/skills", GlobalDir: inHome(".qwen/skills"), Detect: homeExists(".qwen")},
	{ID: "replit", Display: "Replit", Dir: ".agents/skills", GlobalDir: inConfig("agents/skills"), Detect: cwdExists(".replit"), HideInUniversalList: true},
	{ID: "reasonix", Display: "Reasonix", Dir: ".reasonix/skills", GlobalDir: inEnvHome("REASONIX_HOME", ".reasonix", "skills"), Detect: envHomeExists("REASONIX_HOME", ".reasonix")},
	{ID: "rovodev", Display: "Rovo Dev", Dir: ".rovodev/skills", GlobalDir: inHome(".rovodev/skills"), Detect: homeExists(".rovodev")},
	{ID: "roo", Display: "Roo Code", Dir: ".roo/skills", GlobalDir: inHome(".roo/skills"), Detect: homeExists(".roo")},
	{ID: "sarvam-code", Display: "Sarvam Code", Dir: ".agents/skills", GlobalDir: inHome(".agents/skills"), Detect: envHomeExists("SARVAM_HOME", ".sarvam"), HideInUniversalPrompt: true},
	{ID: "tabnine-cli", Display: "Tabnine CLI", Dir: ".tabnine/agent/skills", GlobalDir: inHome(".tabnine/agent/skills"), Detect: homeExists(".tabnine")},
	{ID: "terramind", Display: "Terramind", Dir: ".terramind/skills", GlobalDir: inHome(".terramind/skills"), Detect: homeExists(".terramind")},
	{ID: "tinycloud", Display: "Tinycloud", Dir: ".tinycloud/skills", GlobalDir: inHome(".tinycloud/skills"), Detect: homeExists(".tinycloud")},
	{ID: "trae", Display: "Trae", Dir: ".trae/skills", GlobalDir: inHome(".trae/skills"), Detect: homeExists(".trae")},
	{ID: "trae-cn", Display: "Trae CN", Dir: ".trae/skills", GlobalDir: inHome(".trae-cn/skills"), Detect: homeExists(".trae-cn")},
	{ID: "warp", Display: "Warp", Dir: ".agents/skills", GlobalDir: inHome(".agents/skills"), Detect: homeExists(".warp")},
	{ID: "windsurf", Display: "Windsurf", Dir: ".windsurf/skills", GlobalDir: inHome(".codeium/windsurf/skills"), Detect: homeExists(".codeium/windsurf")},
	{ID: "zed", Display: "Zed", Dir: ".agents/skills", GlobalDir: inHome(".agents/skills"), Detect: zedInstalled},
	{ID: "zcode", Display: "ZCode", Dir: ".zcode/skills", GlobalDir: inHome(".zcode/skills"), Detect: anyExists(homeExists(".zcode"), pathExists("/Applications/ZCode.app"))},
	{ID: "zencoder", Display: "Zencoder", Dir: ".zencoder/skills", GlobalDir: inHome(".zencoder/skills"), Detect: homeExists(".zencoder")},
	{ID: "zenflow", Display: "Zenflow", Dir: ".zencoder/skills", GlobalDir: inHome(".zencoder/skills"), Detect: homeExists(".zencoder")},
	{ID: "neovate", Display: "Neovate", Dir: ".neovate/skills", GlobalDir: inHome(".neovate/skills"), Detect: homeExists(".neovate")},
	{ID: "pochi", Display: "Pochi", Dir: ".pochi/skills", GlobalDir: inHome(".pochi/skills"), Detect: homeExists(".pochi")},
	{ID: "promptscript", Display: "PromptScript", Dir: ".agents/skills", Detect: cwdExists(".promptscript", "promptscript.yaml"), HideInUniversalPrompt: true},
	{ID: "adal", Display: "AdaL", Dir: ".adal/skills", GlobalDir: inHome(".adal/skills"), Detect: homeExists(".adal")},
	{ID: "universal", Display: "Universal", Dir: ".agents/skills", GlobalDir: inConfig("agents/skills"), Detect: neverInstalled, HideInUniversalList: true},
}

func agentByID(id string) (agentDef, bool) {
	i := slices.IndexFunc(agentTable, func(a agentDef) bool { return a.ID == id })
	if i < 0 {
		return agentDef{}, false
	}
	return agentTable[i], true
}

func agentIDs() []string {
	ids := make([]string, len(agentTable))
	for i, a := range agentTable {
		ids[i] = a.ID
	}
	return ids
}

func agentDisplay(id string) string {
	if a, ok := agentByID(id); ok {
		return a.Display
	}
	return id
}

func isUniversal(id string) bool {
	a, _ := agentByID(id)
	return a.Dir == ".agents/skills"
}

func readsCanonicalGlobally(id string) bool {
	a, _ := agentByID(id)
	return isUniversal(id) && !a.GlobalSkipsCanonical
}

func usesCanonical(id string, global bool) bool {
	if global {
		return readsCanonicalGlobally(id)
	}
	return isUniversal(id)
}

func agentBaseDir(id string, global bool, root, subagent string, env agentEnv) string {
	a, _ := agentByID(id)
	if global && a.GlobalDir == nil {
		return ""
	}
	if usesCanonical(id, global) {
		return filepath.Join(root, ".agents", "skills")
	}
	if id == "eve" && subagent != "" {
		return filepath.Join(root, "agent", "subagents", sanitizeName(subagent), "skills")
	}
	if global {
		return a.GlobalDir(env)
	}
	return filepath.Join(root, filepath.FromSlash(a.Dir))
}

func detectInstalledAgents(env agentEnv) []string {
	var ids []string
	for _, a := range agentTable {
		if a.Detect(env) {
			ids = append(ids, a.ID)
		}
	}
	return ids
}

func ensureUniversalAgents(ids []string) []string {
	out := slices.Clone(ids)
	for _, a := range agentTable {
		if !a.HideInUniversalList && readsCanonicalGlobally(a.ID) {
			out = appendMissing(out, a.ID)
		}
	}
	return out
}

func defaultCopyMode(agents, subagents []string) bool {
	keys := map[string]bool{}
	allEve := true
	for _, id := range agents {
		if id != "eve" {
			allEve = false
			a, _ := agentByID(id)
			keys[a.Dir] = true
			continue
		}
		for _, sub := range eveSubagents(subagents) {
			if sub == "" {
				keys["agent/skills"] = true
			} else {
				keys["eve:subagent:"+sub] = true
			}
		}
	}
	return len(keys) <= 1 || allEve
}

func eveSubagents(subagents []string) []string {
	if len(subagents) == 0 {
		return []string{""}
	}
	return subagents
}

var devinMarker = "/opt/.devin"

var hostAgentTypes = map[string]string{
	"cursor":         "cursor",
	"cursor-cli":     "cursor",
	"claude":         "claude-code",
	"cowork":         "claude-code",
	"devin":          "universal",
	"replit":         "replit",
	"gemini":         "gemini-cli",
	"codex":          "codex",
	"antigravity":    "antigravity",
	"augment-cli":    "augment",
	"opencode":       "opencode",
	"github-copilot": "github-copilot",
}

func hostAgent(getenv func(string) string) (name, id string) {
	set := func(keys ...string) bool {
		return slices.ContainsFunc(keys, func(k string) bool { return getenv(k) != "" })
	}
	switch {
	case strings.TrimSpace(getenv("AI_AGENT")) != "":
		name = strings.TrimSpace(getenv("AI_AGENT"))
		if name == "github-copilot-cli" {
			name = "github-copilot"
		}
	case set("CURSOR_TRACE_ID"):
		name = "cursor"
	case set("CURSOR_AGENT") || getenv("CURSOR_EXTENSION_HOST_ROLE") == "agent-exec":
		name = "cursor-cli"
	case set("GEMINI_CLI"):
		name = "gemini"
	case set("CODEX_SANDBOX", "CODEX_CI", "CODEX_THREAD_ID"):
		name = "codex"
	case set("ANTIGRAVITY_AGENT"):
		name = "antigravity"
	case set("AUGMENT_AGENT"):
		name = "augment-cli"
	case getenv("OPENCODE") == "1" || set("OPENCODE_CLIENT"):
		name = "opencode"
	case set("CLAUDECODE", "CLAUDE_CODE"):
		name = "claude"
		if set("CLAUDE_CODE_IS_COWORK") {
			name = "cowork"
		}
	case set("REPL_ID"):
		name = "replit"
	case set("COPILOT_MODEL", "COPILOT_ALLOW_ALL", "COPILOT_GITHUB_TOKEN"):
		name = "github-copilot"
	case fileExists(devinMarker):
		name = "devin"
	}
	if name == "cursor" || name == "cursor-cli" {
		if strings.TrimSpace(getenv("CURSOR_AGENT")) == "" && getenv("CURSOR_EXTENSION_HOST_ROLE") != "agent-exec" {
			return "", ""
		}
		name = "cursor-cli"
	}
	return name, hostAgentTypes[name]
}
