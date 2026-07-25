import {
  AssistantMessageComponent,
  createAgentSession,
  getMarkdownTheme,
  ToolExecutionComponent,
  UserMessageComponent,
  SessionManager,
  type AgentSession,
  type AgentSessionEvent,
  type ExtensionAPI,
  type ExtensionCommandContext,
} from '@earendil-works/pi-coding-agent';
import {
  Container,
  Input,
  Key,
  matchesKey,
  truncateToWidth,
  visibleWidth,
  type Focusable,
  type KeybindingsManager,
  type OverlayHandle,
  type TUI,
} from '@earendil-works/pi-tui';

const OSC133_PROMPT_MARKER_RE = /\x1b\]133;[ABC]\x07/g;
/** Frame rows that are always drawn: 3 borders, 3 dividers, title, footer, controls. */
const OVERLAY_CHROME_ROWS = 8;
/** The quiet blank row under the title, dropped when the overlay cannot afford it. */
const OVERLAY_SPACER_ROWS = 1;
/** Divider between inline items. The pipe is reserved for the frame edges. */
const SEPARATOR = '•';

/**
 * Geometry that Pi resolves from the `overlayOptions` this extension registers.
 * `resolveOverlayLayout` in pi-tui derives the reserved box from those options
 * and `compositeOverlays` slices anything the component renders beyond it - so
 * a component that sizes itself independently loses its bottom rows, frame
 * included. These constants are the single source of truth: they are passed to
 * Pi as `overlayOptions` and used here to size the frame.
 */
const OVERLAY_MAX_HEIGHT_PERCENT = 60;
const OVERLAY_WIDTH_PERCENT = 100;
const OVERLAY_MARGIN_BOTTOM = 1;
const OVERLAY_MARGIN_LEFT = 0;
const OVERLAY_MARGIN_RIGHT = 0;
const OVERLAY_PREFERRED_HEIGHT_FRACTION = 0.55;

/** Pi's resolved horizontal geometry for the full-width overlay. */
export function resolveOverlayWidth(terminalColumns: number): number {
  return Math.max(1, Math.floor((terminalColumns * OVERLAY_WIDTH_PERCENT) / 100));
}

export function resolveOverlayColumn(_terminalColumns: number): number {
  return OVERLAY_MARGIN_LEFT;
}

/** The tallest overlay Pi will composite without slicing rows off the bottom. */
export function resolveOverlayMaxHeight(terminalRows: number): number {
  const fromPercent = Math.floor((terminalRows * OVERLAY_MAX_HEIGHT_PERCENT) / 100);
  const fromMargin = terminalRows - OVERLAY_MARGIN_BOTTOM;
  return Math.max(1, Math.min(fromPercent, fromMargin));
}

/**
 * Smallest frame that still draws itself completely: the always-on chrome plus
 * one input row and one transcript row. Terminals too short for this cannot
 * show the overlay intact, and Pi will slice the bottom off.
 */
export const OVERLAY_MIN_HEIGHT = OVERLAY_CHROME_ROWS + 2;

/**
 * The height this overlay draws: its preferred size, raised to the smallest
 * frame that fits its own chrome and then capped at what Pi reserves. Asking
 * for less than the frame needs is what silently costs the bottom border.
 */
export function resolveOverlayHeight(terminalRows: number): number {
  const preferred = Math.floor(terminalRows * OVERLAY_PREFERRED_HEIGHT_FRACTION);
  return Math.min(
    resolveOverlayMaxHeight(terminalRows),
    Math.max(OVERLAY_MIN_HEIGHT, preferred)
  );
}

function stripPromptMarkers(lines: string[]): string[] {
  return lines.map(line => line.replace(OSC133_PROMPT_MARKER_RE, ''));
}

const SGR_RE = /\x1b\[([0-9;]*)m/g;

/** True for any SGR that returns the cell to the terminal's default background. */
function resetsBackground(params: string): boolean {
  if (params === '') return true; // `ESC [ m` is `ESC [ 0 m`
  return params.split(';').some(part => part === '' || part === '0' || part === '49');
}

export function formatBackgroundAnsi(rgb: { r: number; g: number; b: number }): string {
  return `\x1b[48;2;${rgb.r};${rgb.g};${rgb.b}m`;
}

/**
 * Make one overlay row opaque.
 *
 * Terminals composite their window backdrop behind every cell left on the
 * DEFAULT background, so with transparency or blur turned on the desktop shows
 * through an overlay that only ever sets foreground colours - which is why
 * clipping and padding rows to the exact width never removed the artifacts.
 * Cells carrying an explicit background are drawn fully opaque instead.
 *
 * Padding alone is not enough: content nested in the row (theme.bg(...) ends
 * with `ESC [ 49 m`, markdown ends with `ESC [ 0 m`) resets the background
 * mid-row and punches the hole straight back open, so the background is
 * re-asserted after every such reset.
 */
export function paintRowBackground(row: string, backgroundAnsi: string): string {
  if (!backgroundAnsi) return row;
  const reasserted = row.replace(SGR_RE, (sequence, params: string) =>
    resetsBackground(params) ? sequence + backgroundAnsi : sequence
  );
  return backgroundAnsi + reasserted + '\x1b[49m';
}

export function fitLineToWidth(line: string, width: number): string {
  const safeWidth = Math.max(0, width);
  const clipped = truncateToWidth(line, safeWidth, '');
  return clipped + ' '.repeat(Math.max(0, safeWidth - visibleWidth(clipped)));
}

export class GhostOverlayComponent implements Focusable {
  private readonly transcriptContainer = new Container();
  private readonly input = new Input();
  private readonly tui: TUI;
  private readonly theme: ExtensionCommandContext['ui']['theme'];
  private readonly sessionCwd: string;
  private readonly getModelStatus: () => string;
  private readonly getBackgroundAnsi: () => string;
  private readonly onSubmitMessage: (text: string) => void;
  private readonly onHideOverlay: () => void;
  private readonly onCloseOverlay: () => void;
  private streamingComponent?: AssistantMessageComponent;
  private pendingTools = new Map<string, ToolExecutionComponent>();
  private statusText = 'Ask a quick side question without interrupting the main conversation';
  private scrollOffset = Number.MAX_SAFE_INTEGER;
  private followMode = true;
  private cachedTranscriptLines?: string[];
  private cachedTranscriptWidth?: number;
  private lastInnerWidth = 0;
  private _focused = false;

  get focused(): boolean {
    return this._focused;
  }

  set focused(value: boolean) {
    this._focused = value;
    this.input.focused = value;
  }

  constructor(
    tui: TUI,
    theme: ExtensionCommandContext['ui']['theme'],
    keybindings: KeybindingsManager,
    sessionCwd: string,
    getModelStatus: () => string,
    getBackgroundAnsi: () => string,
    onSubmitMessage: (text: string) => void,
    onHideOverlay: () => void,
    onCloseOverlay: () => void
  ) {
    void keybindings;
    this.tui = tui;
    this.theme = theme;
    this.sessionCwd = sessionCwd;
    this.getModelStatus = getModelStatus;
    this.getBackgroundAnsi = getBackgroundAnsi;
    this.onSubmitMessage = onSubmitMessage;
    this.onHideOverlay = onHideOverlay;
    this.onCloseOverlay = onCloseOverlay;

    this.input.onSubmit = value => {
      const text = value.trim();
      if (!text) return;
      this.input.setValue('');
      this.scrollToBottom();
      this.onSubmitMessage(text);
    };
    this.input.onEscape = () => {
      this.onCloseOverlay();
    };
  }

  private invalidateTranscriptCache(): void {
    this.cachedTranscriptLines = undefined;
    this.cachedTranscriptWidth = undefined;
  }

  private scrollToBottom(): void {
    this.followMode = true;
    this.scrollOffset = Number.MAX_SAFE_INTEGER;
  }

  private getOverlayHeight(): number {
    return resolveOverlayHeight(this.tui.terminal.rows);
  }

  /**
   * Row budget for one frame. The spacer row is the first thing sacrificed when
   * the terminal is short, so the bottom border always survives inside the box
   * Pi reserved.
   */
  private computeLayout(innerWidth: number): {
    totalHeight: number;
    inputLines: string[];
    showSpacer: boolean;
    viewportRows: number;
  } {
    const totalHeight = this.getOverlayHeight();
    const inputLines = this.input.render(innerWidth);
    const withSpacer = OVERLAY_CHROME_ROWS + OVERLAY_SPACER_ROWS + inputLines.length;
    const showSpacer = totalHeight >= withSpacer + 1;
    const chromeRows = showSpacer ? withSpacer : OVERLAY_CHROME_ROWS + inputLines.length;
    return {
      totalHeight,
      inputLines,
      showSpacer,
      viewportRows: Math.max(1, totalHeight - chromeRows),
    };
  }

  private getFallbackInnerWidth(): number {
    return Math.max(1, Math.floor(this.tui.terminal.columns * 0.85) - 2);
  }

  private getTranscriptLines(width: number): string[] {
    if (this.cachedTranscriptLines && this.cachedTranscriptWidth === width) {
      return this.cachedTranscriptLines;
    }

    const lines = stripPromptMarkers(this.transcriptContainer.render(width));
    this.cachedTranscriptLines = lines;
    this.cachedTranscriptWidth = width;
    return lines;
  }

  private getCurrentMaxScroll(): number {
    const innerWidth = this.lastInnerWidth || this.getFallbackInnerWidth();
    const { viewportRows } = this.computeLayout(innerWidth);
    const transcriptLines = this.getTranscriptLines(innerWidth);
    return Math.max(0, transcriptLines.length - viewportRows);
  }

  setStatus(text: string): void {
    this.statusText = text;
    this.tui.requestRender();
  }

  handleInput(data: string): void {
    if (matchesKey(data, Key.ctrl('h'))) {
      this.onHideOverlay();
      return;
    }

    if (matchesKey(data, Key.up)) {
      const maxScroll = this.getCurrentMaxScroll();
      this.followMode = false;
      this.scrollOffset = Math.max(0, Math.min(this.scrollOffset, maxScroll) - 1);
      if (this.scrollOffset >= maxScroll) this.scrollOffset = Math.max(0, maxScroll - 1);
      this.tui.requestRender();
      return;
    }

    if (matchesKey(data, Key.down)) {
      const maxScroll = this.getCurrentMaxScroll();
      if (this.followMode) {
        this.scrollToBottom();
      } else {
        this.scrollOffset = Math.min(maxScroll, this.scrollOffset + 1);
        if (this.scrollOffset >= maxScroll) this.scrollToBottom();
      }
      this.tui.requestRender();
      return;
    }

    if (matchesKey(data, Key.pageUp)) {
      const maxScroll = this.getCurrentMaxScroll();
      this.followMode = false;
      this.scrollOffset = Math.max(0, Math.min(this.scrollOffset, maxScroll) - 10);
      if (this.scrollOffset >= maxScroll) this.scrollOffset = Math.max(0, maxScroll - 10);
      this.tui.requestRender();
      return;
    }

    if (matchesKey(data, Key.pageDown)) {
      const maxScroll = this.getCurrentMaxScroll();
      if (this.followMode) {
        this.scrollToBottom();
      } else {
        this.scrollOffset = Math.min(maxScroll, this.scrollOffset + 10);
        if (this.scrollOffset >= maxScroll) this.scrollToBottom();
      }
      this.tui.requestRender();
      return;
    }

    if (matchesKey(data, Key.home)) {
      this.followMode = false;
      this.scrollOffset = 0;
      this.tui.requestRender();
      return;
    }

    if (matchesKey(data, Key.end)) {
      this.scrollToBottom();
      this.tui.requestRender();
      return;
    }

    this.input.handleInput(data);
    this.tui.requestRender();
  }

  handleSessionEvent(event: AgentSessionEvent): void {
    switch (event.type) {
      case 'message_start': {
        if (event.message.role === 'user') {
          this.transcriptContainer.addChild(
            new UserMessageComponent(extractMessageText(event.message), getMarkdownTheme())
          );
          this.setStatus('Thinking...');
          break;
        }

        if (event.message.role === 'assistant') {
          this.streamingComponent = new AssistantMessageComponent(
            undefined,
            false,
            getMarkdownTheme()
          );
          this.transcriptContainer.addChild(this.streamingComponent);
          this.streamingComponent.updateContent(event.message);
          this.setStatus('Streaming response...');
        }
        break;
      }

      case 'message_update': {
        if (this.streamingComponent && event.message.role === 'assistant') {
          this.streamingComponent.updateContent(event.message);
        }
        break;
      }

      case 'message_end': {
        if (event.message.role !== 'assistant') break;
        if (this.streamingComponent) {
          this.streamingComponent.updateContent(event.message);

          if (event.message.stopReason === 'aborted' || event.message.stopReason === 'error') {
            for (const component of this.pendingTools.values()) {
              component.updateResult({
                content: [{ type: 'text', text: event.message.errorMessage || 'Error' }],
                isError: true,
              });
            }
            this.pendingTools.clear();
          } else {
            for (const component of this.pendingTools.values()) {
              component.setArgsComplete();
            }
          }
          this.streamingComponent = undefined;
        }
        this.setStatus('Ask a quick side question without interrupting the main conversation');
        break;
      }

      case 'tool_execution_start': {
        let component = this.pendingTools.get(event.toolCallId);
        if (!component) {
          component = new ToolExecutionComponent(
            event.toolName,
            event.toolCallId,
            event.args,
            { showImages: true },
            undefined,
            this.tui,
            this.sessionCwd
          );
          this.transcriptContainer.addChild(component);
          this.pendingTools.set(event.toolCallId, component);
        }
        component.markExecutionStarted();
        this.setStatus(`Running ${event.toolName}...`);
        break;
      }

      case 'tool_execution_update': {
        const component = this.pendingTools.get(event.toolCallId);
        if (component) {
          component.updateResult({ ...event.partialResult, isError: false }, true);
        }
        break;
      }

      case 'tool_execution_end': {
        const component = this.pendingTools.get(event.toolCallId);
        if (component) {
          component.updateResult({ ...event.result, isError: event.isError });
          this.pendingTools.delete(event.toolCallId);
        }
        break;
      }

      case 'agent_end': {
        this.streamingComponent = undefined;
        this.pendingTools.clear();
        this.setStatus('Ask a quick side question without interrupting the main conversation');
        break;
      }
    }

    this.invalidateTranscriptCache();
    if (this.followMode) this.scrollToBottom();
    this.tui.requestRender();
  }

  invalidate(): void {
    this.transcriptContainer.invalidate();
    this.input.invalidate();
    this.invalidateTranscriptCache();
  }

  render(width: number): string[] {
    const th = this.theme;
    const innerW = Math.max(1, width - 2);
    const { inputLines, showSpacer, viewportRows } = this.computeLayout(innerW);
    const transcriptLines = this.getTranscriptLines(innerW);
    const maxScroll = Math.max(0, transcriptLines.length - viewportRows);

    this.lastInnerWidth = innerW;

    if (this.followMode) {
      this.scrollOffset = maxScroll;
    } else {
      this.scrollOffset = Math.max(0, Math.min(this.scrollOffset, maxScroll));
      if (this.scrollOffset >= maxScroll) this.scrollToBottom();
    }

    const visibleTranscript = transcriptLines.slice(
      this.scrollOffset,
      this.scrollOffset + viewportRows
    );
    const scrollInfo =
      transcriptLines.length > viewportRows
        ? `${this.scrollOffset + 1}-${Math.min(this.scrollOffset + viewportRows, transcriptLines.length)}/${transcriptLines.length}`
        : `${transcriptLines.length}L`;
    const followIcon = this.followMode ? th.fg('success', '●') : th.fg('dim', '○');
    const sep = th.fg('border', SEPARATOR);
    const footer = truncateToWidth(
      ` ${th.fg('dim', this.statusText)} ${sep} ${th.fg('dim', formatGhostCwd(this.sessionCwd))} ${sep} ${th.fg('dim', scrollInfo)} ${followIcon}`,
      innerW
    );
    const title =
      ` ${th.fg('accent', th.bold('better pi'))} ${sep} ` +
      `${th.fg('accent', this.getModelStatus())} ${sep} ${th.fg('dim', 'temporary session')}`;
    const controls =
      ` ${th.fg('accent', 'esc')} ${th.fg('dim', 'close')} ` +
      `${sep} ${th.fg('accent', 'up/down')} ${th.fg('dim', 'scroll')} ` +
      `${sep} ${th.fg('accent', 'ctrl+h')} ${th.fg('dim', 'hide')}`;
    const topBorder = th.fg('border', '╭' + '─'.repeat(innerW) + '╮');
    const divider = th.fg('border', '├' + '─'.repeat(innerW) + '┤');
    const bottomBorder = th.fg('border', '╰' + '─'.repeat(innerW) + '╯');
    const framedLine = (line: string) =>
      th.fg('border', '│') + fitLineToWidth(line, innerW) + th.fg('border', '│');
    const lines: string[] = [topBorder, framedLine(title)];
    if (showSpacer) lines.push(framedLine(''));
    lines.push(divider);

    for (const line of visibleTranscript) {
      lines.push(framedLine(line));
    }
    for (let i = visibleTranscript.length; i < viewportRows; i++) {
      lines.push(framedLine(''));
    }

    lines.push(divider, framedLine(footer));
    for (const line of inputLines) {
      lines.push(framedLine(line));
    }
    lines.push(divider, framedLine(controls), bottomBorder);

    const backgroundAnsi = this.getBackgroundAnsi();
    if (backgroundAnsi) {
      return lines.map(line => paintRowBackground(line, backgroundAnsi));
    }

    return lines;
  }
}

function extractMessageText(message: {
  content: string | Array<{ type: string; text?: string }>;
}): string {
  if (typeof message.content === 'string') return message.content.trim();

  return message.content
    .filter(
      (part): part is { type: 'text'; text: string } =>
        part.type === 'text' && typeof part.text === 'string'
    )
    .map(part => part.text)
    .join('\n')
    .trim();
}

export function formatModelStatus(modelId: string, thinkingLevel: string): string {
  return thinkingLevel && thinkingLevel !== 'off' ? `${modelId} (${thinkingLevel})` : modelId;
}

async function queryBackgroundAnsi(tui: TUI): Promise<string> {
  try {
    const rgb = await tui.queryTerminalBackgroundColor({ timeoutMs: 250 });
    return rgb ? formatBackgroundAnsi(rgb) : '';
  } catch {
    return '';
  }
}

function formatGhostCwd(cwd: string): string {
  const home = process.env.HOME;
  if (home && cwd.startsWith(home)) {
    return `~${cwd.slice(home.length)}` || '~';
  }
  return cwd;
}

export default function (pi: ExtensionAPI) {
  let ghostSession: AgentSession | null = null;
  let ghostSessionCwd: string | null = null;
  let ghostModelLabel: string | null = null;
  let overlayHandle: OverlayHandle | null = null;
  let overlayClosed = false;

  const cleanupGhost = (ctx?: ExtensionCommandContext) => {
    overlayClosed = true;
    if (overlayHandle) {
      try {
        overlayHandle.hide();
      } catch {
        // ignore
      }
      overlayHandle = null;
    }
    if (ghostSession) {
      ghostSession.dispose();
      ghostSession = null;
    }
    ghostSessionCwd = null;
    ghostModelLabel = null;
    ctx?.ui.setWidget('pi-better-btw', undefined);
  };

  const setHiddenState = (ctx: ExtensionCommandContext, hidden: boolean) => {
    if (!overlayHandle) return;
    overlayHandle.setHidden(hidden);
    if (hidden) {
      overlayHandle.unfocus();
      ctx.ui.setWidget(
        'pi-better-btw',
        (_tui, theme) => ({
          render: () => [
            theme.fg('accent', '/btw ') + theme.fg('dim', 'is running • run /btw to bring it back'),
          ],
          invalidate: () => {},
        }),
        { placement: 'aboveEditor' }
      );
    } else {
      ctx.ui.setWidget('pi-better-btw', undefined);
      overlayHandle.focus();
    }
  };

  const ensureGhostSession = async (ctx: ExtensionCommandContext): Promise<AgentSession> => {
    if (ghostSession) return ghostSession;
    if (!ctx.model) throw new Error('No model selected');

    const result = await createAgentSession({
      cwd: ctx.cwd,
      model: ctx.model,
      sessionManager: SessionManager.inMemory(ctx.cwd),
    });
    ghostSession = result.session;
    ghostSessionCwd = ctx.cwd;
    ghostModelLabel = ctx.model.id;
    return ghostSession;
  };

  const openGhostOverlay = async (ctx: ExtensionCommandContext, initialPrompt?: string) => {
    const session = await ensureGhostSession(ctx);
    overlayClosed = false;

    void ctx.ui
      .custom<void>(
        (tui, theme, keybindings, done) => {
          // Asked for once per overlay: the terminal's own default background,
          // painted explicitly so the overlay's cells are opaque. Until it
          // answers - or if it never does - rows render exactly as before.
          let backgroundAnsi = '';
          void queryBackgroundAnsi(tui).then(ansi => {
            if (!ansi || overlayClosed) return;
            backgroundAnsi = ansi;
            tui.requestRender();
          });

          const overlay = new GhostOverlayComponent(
            tui,
            theme,
            keybindings,
            ghostSessionCwd ?? ctx.cwd,
            () =>
              formatModelStatus(
                ghostModelLabel ?? ctx.model?.id ?? 'unknown-model',
                pi.getThinkingLevel()
              ),
            () => backgroundAnsi,
            text => {
              void session.prompt(text, { images: [] });
            },
            () => {
              setHiddenState(ctx, true);
            },
            () => {
              done();
            }
          );

          const unsubscribe = session.subscribe(event => {
            overlay.handleSessionEvent(event);
          });

          if (initialPrompt?.trim()) {
            void session.prompt(initialPrompt.trim(), { images: [] });
          }

          return {
            render: (width: number) => overlay.render(width),
            invalidate: () => overlay.invalidate(),
            handleInput: (data: string) => overlay.handleInput(data),
            get focused() {
              return overlay.focused;
            },
            set focused(value: boolean) {
              overlay.focused = value;
            },
            dispose: () => {
              unsubscribe();
            },
          };
        },
        {
          overlay: true,
          overlayOptions: {
            anchor: 'bottom-center',
            width: `${OVERLAY_WIDTH_PERCENT}%`,
            // Must stay in step with resolveOverlayMaxHeight; Pi slices any row
            // the component renders past the box these options reserve.
            maxHeight: `${OVERLAY_MAX_HEIGHT_PERCENT}%`,
            margin: {
              bottom: OVERLAY_MARGIN_BOTTOM,
              left: OVERLAY_MARGIN_LEFT,
              right: OVERLAY_MARGIN_RIGHT,
            },
          },
          onHandle: handle => {
            overlayHandle = handle;
          },
        }
      )
      .finally(() => {
        overlayHandle = null;
        if (!overlayClosed) {
          cleanupGhost(ctx);
        }
      });
  };

  pi.registerCommand('btw', {
    description: 'Open a quick pop-up chat for those btw moments',
    handler: async (args, ctx) => {
      if (!ctx.hasUI) {
        ctx.ui.notify('/btw requires interactive mode', 'error');
        return;
      }

      const prompt = args.trim();

      if (overlayHandle) {
        if (overlayHandle.isHidden()) {
          setHiddenState(ctx, false);
        }
        if (prompt) {
          const session = await ensureGhostSession(ctx);
          void session.prompt(prompt, { images: [] });
        }
        return;
      }

      await openGhostOverlay(ctx, prompt || undefined);
    },
  });

  pi.on('session_shutdown', async (_event, ctx) => {
    cleanupGhost(ctx as ExtensionCommandContext);
  });
}
