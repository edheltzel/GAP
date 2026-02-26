---
description: Speak a message using the VoiceServer TTS system
---

Send a voice notification using the VoiceServer. This command speaks the provided message aloud using ElevenLabs TTS (with macOS say fallback).

## Usage

```
/voice <message>
```

## Examples

- `/voice Hello world` - Speak "Hello world"
- `/voice Task completed successfully` - Announce completion
- `/voice Remember to commit your changes` - Reminder

## Implementation

Execute the following curl command to send the notification:

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "$ARGUMENTS", "title": "Voice Command"}'
```

If the VoiceServer is not running, start it first:

```bash
~/.claude/VoiceServer/start-server.sh --quiet
```

Then send the notification.

Report the result to the user: either "Message spoken" or any error that occurred.
