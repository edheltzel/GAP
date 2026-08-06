# Extract Workflow

Extract dynamic, content-adaptive wisdom from any content source.

## Input Sources

| Source | Method |
| -------- | -------- |
| YouTube or video URL | `fetch_content` with transcript extraction enabled |
| Article URL | `fetch_content` in readable mode |
| File path | Read the file directly |
| Pasted text | Use directly |

For long content, save the transcript to a temporary working file and chunk it. Do not extract from a three-hour transcript in one pass.

## Execution

1. Fetch URL or video content with `fetch_content`; use the returned transcript when the source is a video.
2. Read local files directly. For pasted text, use the text as supplied.
3. Extract wisdom locally according to the depth and voice contracts in `../SKILL.md`.
4. Return the result in the conversation by default.
5. Offer optional Recall persistence only when the user asks to save the extraction. Use the live Recall tools and schemas; do not assume a local memory path.

## The Deliverable

Get the full content via the method above, then produce the extraction defined in SKILL.md:

- Sections named specifically for THIS content (the depth-level table sets the count), each with at least 3 Level-3 bullets.
- Bullets that sound spoken, carry the specific detail or quote, and preserve any contrarian take undiluted.
- Closing sections per the depth-level table.

Output in the format from SKILL.md, and pass its Quality Check before delivering.
