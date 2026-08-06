---
name: BrightData
version: 1.2.19
description: "Progressive web scraping that starts with current content fetching, preserves a public curl path, uses installed Chrome tooling for JavaScript-rendered pages, and offers optional credentialed Bright Data API escalation for blocked targets. USE WHEN Bright Data, scrape URL, web scraping, bot detection, crawl site, CAPTCHA, site blocking, extract page content, or scrape a site. NOT FOR simple public content that fetch_content can retrieve directly or social platform scraping with named actors (use Apify)."
effort: medium
---

## 🚨 MANDATORY: Voice Notification (REQUIRED BEFORE ANY ACTION)

**You MUST send this notification BEFORE doing anything else when this skill is invoked.**

1. **Send voice notification**:

   ```bash
   curl -s -X POST http://localhost:8888/notify \
     -H "Content-Type: application/json" \
     -d '{"message": "Running the WORKFLOWNAME workflow in the BrightData skill to ACTION"}' \
     > /dev/null 2>&1 &
   ```

2. **Output text notification**:

   ```
   Running the **WorkflowName** workflow in the **BrightData** skill to ACTION...
   ```

**This is not optional. Execute this curl command immediately upon skill invocation.**

# BrightData

Scrapes a single URL (FourTierScrape) or crawls a site (Crawl), escalating only as far as each page needs. Output is markdown. Start at Tier 1 and step up only when blocked. A Cloudflare `Accept: text/markdown` pre-check runs before Tier 1.

## The four tiers (tool contract)

| Tier | Tool | Wins on | Cost / latency |
| ------ | ------ | --------- | ---------------- |
| 1 | `fetch_content` | public content and video/page extraction | free, usually seconds |
| 2 | public `curl` with browser headers | user-agent and basic header checks | free, usually seconds |
| 3 | `/opt/homebrew/bin/chrome-devtools-axi` | JavaScript-rendered pages and browser interaction | local browser startup cost |
| 4 | Bright Data HTTP API | CAPTCHA, advanced fingerprinting, or residential-IP needs | paid credits and credentials |

Tier 4 is an optional credentialed escalation. Use it only when the user explicitly approves the paid path and `BRIGHT_DATA_API_KEY` plus the required dataset or zone configuration are present. Do not name or invoke a Bright Data MCP unless the current harness actually exposes one. The exact public curl and Chrome commands live in `Workflows/FourTierScrape.md`.

## Workflows

When routing, output: `Running the **WorkflowName** workflow in the **BrightData** skill to ACTION...`

| Workflow | Trigger | File |
|----------|---------|------|
| FourTierScrape | "scrape/fetch/pull/get/retrieve [URL]", "can't access this site", "site is blocking me", "use Bright Data to fetch" | `Workflows/FourTierScrape.md` |
| Crawl | "crawl this site", "spider this domain", "map this website", "get all pages from", "scrape the whole site", "crawl all pages under /docs" | `Workflows/Crawl.md` |

Crawl picks Light Crawl (per-page tier chain plus link loop, up to 50 pages) for a section, or an explicitly approved Full Crawl through the credentialed Bright Data HTTP API for whole sites.

## Gotchas

- **Escalation order:** `fetch_content`, public `curl`, installed `chrome-devtools-axi`, then optional Bright Data HTTP API.
- **Bright Data has usage costs.** Confirm before Tier 4 and never imply that credentials or an MCP are configured.
- **CAPTCHA solving introduces latency.** Allow extra time for Tier 4 responses.
- **Credential:** `BRIGHT_DATA_API_KEY` in the environment. Dataset or zone identifiers must be supplied for the selected API product.

Replace `WORKFLOW_USED` with the workflow executed, `8_WORD_SUMMARY` with a brief input description, and `SECONDS` with approximate wall-clock time. Log `status: "error"` if the workflow failed.
