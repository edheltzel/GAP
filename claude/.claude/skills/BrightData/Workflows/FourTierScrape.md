# Four-Tier URL Content Scraping

## Voice Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the FourTierScrape workflow in the BrightData skill to scrape URL content"}' \
  > /dev/null 2>&1 &
```

Running **FourTierScrape** in **BrightData**...

---

**Deliverable:** the URL's content in clean markdown, labeled with which tier retrieved it. Start cheap and escalate only on failure. The tier that succeeds first is the answer. Tier table (tool + cost) is in `SKILL.md`.

## Pre-check: Cloudflare markdown negotiation

Before the tier chain, probe for server-side markdown via Cloudflare's [Markdown for Agents](https://blog.cloudflare.com/markdown-for-agents/). Non-Cloudflare sites ignore the header and return HTML, so the probe has no downside.

```bash
curl -sL -H "Accept: text/markdown" "[URL]" | head -5
```

**Markdown detected (any of these) → use the body directly, skip the tiers:**

1. `Content-Type` header contains `text/markdown`
2. `x-markdown-tokens` header present (capture it as token-count metadata)
3. Body starts with YAML frontmatter (`---`) or a markdown heading (`#`) instead of `<!DOCTYPE`/`<html`; Cloudflare's CDN sometimes reports `content-type: text/html` even when the body is markdown

~80% fewer tokens than HTML-to-markdown conversion, ~1-3s, free. HTML or error → proceed to Tier 1.

## Tier 1: Current content fetch

Call `fetch_content` with the URL. For pages, request readable content; for video URLs, request transcript-aware extraction. Success goes to Output. A block, timeout, or empty result advances to Tier 2.

## Tier 2: curl with Chrome headers

The `Sec-Fetch-*` headers are the load-bearing part for bypassing basic detection; `--compressed` handles gzip/br like a real browser.

```bash
curl -L -A "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36" \
  -H "Accept: text/markdown, text/html;q=0.9, application/xhtml+xml;q=0.8, */*;q=0.7" \
  -H "Accept-Language: en-US,en;q=0.9" \
  -H "Accept-Encoding: gzip, deflate, br" \
  -H "DNT: 1" \
  -H "Connection: keep-alive" \
  -H "Upgrade-Insecure-Requests: 1" \
  -H "Sec-Fetch-Dest: document" \
  -H "Sec-Fetch-Mode: navigate" \
  -H "Sec-Fetch-Site: none" \
  -H "Sec-Fetch-User: ?1" \
  -H "Cache-Control: max-age=0" \
  --compressed \
  "[URL]"
```

HTML returned is converted to markdown and sent to Output. Empty, blocked, or JavaScript-required responses advance to Tier 3.

## Tier 3: installed Chrome automation

Use the installed CLI and follow the `chrome-devtools-axi` skill contract. The first command returns an accessibility snapshot. Preserve generation-prefixed refs exactly, refresh the snapshot after rerenders, and confirm every state-changing action with a fresh snapshot.

```bash
BROWSER_CLI=/opt/homebrew/bin/chrome-devtools-axi
"$BROWSER_CLI" open "<url>"
"$BROWSER_CLI" snapshot --full
"$BROWSER_CLI" eval 'document.body.innerText'
"$BROWSER_CLI" stop
```

Rendered text goes to Output. CAPTCHA or advanced bot detection may advance to Tier 4.

## Tier 4: optional credentialed escalation

This is an optional credentialed escalation through the Bright Data HTTP API. It has usage costs. Proceed only after user approval and only when `BRIGHT_DATA_API_KEY` and the API product's dataset or zone identifier are configured. Use the current Bright Data HTTP endpoint documented for that configured product; do not invent an MCP tool name or dataset ID.

If credentials or product configuration are missing, stop and report the exact missing prerequisite. If the API also fails, report the likely access boundary, such as login, paywall, geo restriction, or the site being unavailable.

## Output

Present the markdown content, prefixed with which tier succeeded (and, for Tier 3/4, a one-line note on why escalation was needed). Verify the content is readable, matches the URL, and has no major missing sections.

```markdown
Successfully retrieved content from [URL] using Tier [1/2/3/4]

[Content in markdown format...]
```

## Related

- `Crawl.md`: multi-page site crawling (calls this workflow for its starting URL and as fallback).
