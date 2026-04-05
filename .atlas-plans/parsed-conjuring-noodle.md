# Plan: Replace VoiceMode with VoiceLayer

## Context

VoiceMode (mbailey/voicemode v8.5.1) has lifecycle management gaps — no signal handlers, dead `cleanup_on_shutdown()` code, orphaned MCP server processes consuming ~141MB each, and a Kokoro-FastAPI backend with a PyTorch memory leak growing ~200MB/hour. After surveying 80+ projects across all MCP registries, VoiceLayer (EtanHey/voicelayer) scored 13/21 against voicemode's feature set and is the only TypeScript alternative with a real `converse` tool, Silero VAD, session mutex, and proper signal handlers. It runs as a singleton daemon on a Unix socket — all Claude sessions connect through lightweight `socat` shims, eliminating the orphan problem by design.

## Phase 1: Kill Stale Session & Cleanup Prep

1. **Kill stale Thursday Claude session** (PID 88492) — frees ~755MB
   ```bash
   kill 88492
   ```

2. **Verify child processes died** (chrome-devtools-mcp, recall, context7, pencil MCP servers)
   ```bash
   pgrep -P 88492 || echo "All children dead"
   ```

## Phase 2: Install VoiceLayer Prerequisites

3. **Install system dependencies**
   ```bash
   brew install sox socat whisper-cpp
   pip3 install edge-tts
   ```

4. **Download Whisper model** (large-v3-turbo recommended by VoiceLayer)
   ```bash
   mkdir -p ~/.cache/whisper
   curl -L -o ~/.cache/whisper/ggml-large-v3-turbo.bin \
     https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin
   ```

## Phase 3: Install VoiceLayer

5. **Install VoiceLayer globally via bun**
   ```bash
   bun add -g voicelayer-mcp
   ```

6. **Start VoiceLayer daemon** (install as LaunchAgent for auto-start)
   ```bash
   # Clone repo for launchd scripts, or create plist manually
   # The daemon runs at /tmp/voicelayer-mcp.sock
   ```

7. **Register VoiceLayer as MCP server in Claude Code**
   ```bash
   claude mcp add --scope user voicelayer -- socat STDIO UNIX-CONNECT:/tmp/voicelayer-mcp.sock
   ```

## Phase 4: Uninstall VoiceMode

8. **Remove VoiceMode MCP from Claude Code**
   ```bash
   claude mcp remove voicemode
   ```

9. **Stop and unload Kokoro launchd service**
   ```bash
   launchctl unload ~/Library/LaunchAgents/com.voicemode.kokoro.plist
   kill 1302  # Kokoro process
   ```

10. **Stop and unload Whisper launchd service** (replaced by brew whisper-cpp)
    ```bash
    launchctl unload ~/Library/LaunchAgents/com.voicemode.whisper.plist
    kill 1061  # Whisper server process
    kill 1038  # Kokoro start script
    ```

11. **Uninstall VoiceMode uv tool**
    ```bash
    uv tool uninstall voice-mode
    ```

12. **Remove LaunchAgent plist files**
    ```bash
    rm ~/Library/LaunchAgents/com.voicemode.kokoro.plist
    rm ~/Library/LaunchAgents/com.voicemode.whisper.plist
    ```

13. **Optionally archive ~/.voicemode/ directory** (keep for rollback, remove later)
    ```bash
    mv ~/.voicemode ~/.voicemode.bak
    ```

## Phase 5: Verify

14. **Confirm VoiceLayer is connected**
    ```bash
    claude mcp list  # Should show voicelayer: connected
    ```

15. **Confirm VoiceMode is gone**
    ```bash
    claude mcp list  # Should NOT show voicemode
    ```

16. **Confirm no Python voice processes remain**
    ```bash
    ps aux | grep -iE "kokoro|whisper-server|voicemode" | grep -v grep
    # Should return empty
    ```

17. **Test VoiceLayer converse** — start a new Claude Code session and invoke the converse tool

18. **Confirm no other coding agents load voicemode** — start OpenCode, Codex, Pi sessions and verify no voicemode process spawns (they weren't configured for voicemode based on our check, but confirm)

## Phase 6: Future — Local TTS Fallback (Separate Session)

19. **Configure local TTS backend** in VoiceLayer (XTTS-v2 or F5-TTS MLX) as fallback for offline use. This is a follow-up task, not part of today's work.

## Rollback

If VoiceLayer doesn't work out:
```bash
# Re-register voicemode
claude mcp add --scope user voicemode -- uvx voice-mode
# Restore services
mv ~/.voicemode.bak ~/.voicemode
launchctl load ~/Library/LaunchAgents/com.voicemode.kokoro.plist
launchctl load ~/Library/LaunchAgents/com.voicemode.whisper.plist
# Remove voicelayer
claude mcp remove voicelayer
bun remove -g voicelayer-mcp
```

## Memory Savings Expected

| Before | After | Savings |
|--------|-------|---------|
| Kokoro (702MB RSS) | Removed | -702MB |
| Whisper server (varies) | brew whisper-cpp CLI (on-demand) | -50MB idle |
| VoiceMode MCP (~141MB per session) | VoiceLayer daemon (~50MB total) | -91MB per session |
| Stale session (755MB) | Killed | -755MB |
| **Total** | | **~1.6GB freed** |
