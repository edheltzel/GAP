# Crawl Workflow

**Purpose:** Crawl multiple pages from a website by following links, extracting content, and returning structured results. Two modes are supported: **Light Crawl** uses the per-page tier chain, while **Full Crawl** is an optional credentialed Bright Data HTTP API path for entire sites.

---

## Prerequisites

- Starting URL (required)
- Crawl scope: specific section, depth limit, or full site
- Output format preference (markdown, HTML, JSON)
- For Full Crawl: Bright Data API key (set in environment)

---

## Mode Selection

**Choose based on scale:**

| Mode | Pages | Method | Cost | Best For |
|------|-------|--------|------|----------|
| **Light Crawl** | 1-50 pages | FourTierScrape per page + link extraction loop | local and public-fetch costs only | Section of a site, specific content |
| **Full Crawl** | 50-unlimited | Bright Data Crawl API (HTTP POST) | $1.50/1K pages | Entire site maps, comprehensive extraction |

**Decision logic:**

- User says "crawl this section" or "get all pages under /docs" → **Light Crawl**
- User says "crawl the entire site" or "map this whole site" → **Full Crawl**
- User says "crawl" without scope → **Ask** how many pages or what section, then choose

---

## Light Crawl (Agent-Driven)

Uses iterative link discovery and applies FourTierScrape to each page. Public fetch and curl stay free; optional paid escalation is handled per page only after approval.

### Step 1: Scrape the Starting URL

Use the FourTierScrape workflow to fetch the starting URL. Extract all internal links from the page content.

```
Scrape starting URL → Extract all <a href="..."> links → Filter to same-domain only
```

**Link filtering rules:**

- Same domain only (no external links unless user requests it)
- Respect URL path scope if user specified one (e.g., only links under `/docs/`)
- Deduplicate URLs (normalize trailing slashes, query params)
- Skip: anchors (#), mailto:, tel:, javascript:, static assets (.css, .js, .png, .jpg, .svg, .pdf)

### Step 2: Batch Scrape Discovered Links

Fetch unvisited URLs in bounded groups of up to 10, running FourTierScrape for each URL. Do not assume a batch MCP exists.

```
For each bounded group of unvisited URLs:
  1. Run FourTierScrape for each URL
  2. Extract new internal links from successful results
  3. Add unseen links to the queue
  4. Store page content in the results collection
  5. Repeat until the queue is empty or a page/depth limit is reached
```

**Depth tracking:**

- Starting URL = depth 0
- Links found on starting URL = depth 1
- Links found on depth 1 pages = depth 2
- Default max depth: 3 (override with user preference)

**Page limit:**

- Default: 30 pages
- User can specify: "crawl up to 100 pages"
- Hard cap: 50 pages per Light Crawl (for cost/time control)

### Step 3: Compile Results

Assemble all crawled pages into a structured output:

```markdown
## Crawl Results: [domain]
**Pages crawled:** [N]
**Depth reached:** [N]
**Starting URL:** [URL]

### Site Map
- [URL 1] (depth 0)
  - [URL 2] (depth 1)
  - [URL 3] (depth 1)
    - [URL 4] (depth 2)

### Page Contents

#### [URL 1]
[markdown content]

#### [URL 2]
[markdown content]
...
```

### Light Crawl Error Handling

- If a page fails all 4 tiers → log it as failed, continue crawling other pages
- If >50% of pages fail → warn user, suggest Full Crawl instead
- If rate limited → add 2-second delay between batches

---

## Full Crawl (optional Bright Data HTTP API)

For large-scale site crawling. This is a paid, credentialed path. Confirm cost and scope with the user before making requests. Require `BRIGHT_DATA_API_KEY` and a real `BRIGHT_DATA_DATASET_ID`; never leave a placeholder dataset ID in an executable request.

### Step 1: Configure the Crawl

Build the API request based on user requirements:

```bash
test -n "${BRIGHT_DATA_API_KEY:-}" || { echo "BRIGHT_DATA_API_KEY is required" >&2; exit 1; }
test -n "${BRIGHT_DATA_DATASET_ID:-}" || { echo "BRIGHT_DATA_DATASET_ID is required" >&2; exit 1; }
curl -X POST "https://api.brightdata.com/datasets/v3/trigger?dataset_id=${BRIGHT_DATA_DATASET_ID}&format=json" \
  -H "Authorization: Bearer ${BRIGHT_DATA_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '[{
    "url": "[STARTING_URL]",
    "crawl_depth": [DEPTH],
    "url_filter": "[REGEX_PATTERN]",
    "format": "markdown"
  }]'
```

**Key parameters:**

- `url`: Starting URL for the crawl
- `crawl_depth`: How many link hops to follow (default: 3)
- `url_filter`: Regex to restrict which URLs to crawl (e.g., `"https://example\\.com/docs/.*"`)
- `format`: Output format: `markdown`, `html`, `json`, `ld_json`
- `include_errors`: Set to `true` for detailed error logs

### Step 2: Monitor Progress

The API returns a `snapshot_id`. Poll for completion:

```bash
curl -X GET "https://api.brightdata.com/datasets/v3/progress/${SNAPSHOT_ID}" \
  -H "Authorization: Bearer ${BRIGHT_DATA_API_KEY}"
```

**Status values:** `running`, `ready`, `failed`

- Poll every 10 seconds
- Timeout after 5 minutes for small sites, 15 minutes for large
- Report progress to user: "Crawling in progress... X pages collected so far"

### Step 3: Retrieve Results

Once status is `ready`:

```bash
curl -X GET "https://api.brightdata.com/datasets/v3/snapshot/${SNAPSHOT_ID}?format=json" \
  -H "Authorization: Bearer ${BRIGHT_DATA_API_KEY}"
```

Results come as an array of page objects, each with URL and content.

### Step 4: Present Results

Same structured output format as Light Crawl, plus:

- Total pages crawled
- Total cost estimate ($1.50/1K pages)
- Any errors encountered
- Site map derived from crawled URLs

### Full Crawl Error Handling

- API auth failure → check `BRIGHT_DATA_API_KEY` environment variable
- Timeout → offer to increase wait time or try Light Crawl for smaller scope
- Partial failure → deliver what was crawled, list failed URLs

---

## Output & Verification

For both modes:

1. **Present site map:** hierarchical URL tree showing crawl structure
2. **Present content:** each page's content in the requested format
3. **Summary stats:** pages crawled, depth, time taken, cost
4. **Quality check:** flag any pages that returned empty or error content
5. **Offer export:** if the result set is large, offer to write it to a file

---

## Cost Considerations

| Mode | Cost | Time |
| ------ | ------ | ------ |
| Light Crawl (10 pages) | ~$0.06 | 30-60 seconds |
| Light Crawl (30 pages) | ~$0.18 | 1-3 minutes |
| Light Crawl (50 pages) | ~$0.30 | 2-5 minutes |
| Full Crawl (100 pages) | ~$0.15 | 1-3 minutes |
| Full Crawl (1000 pages) | ~$1.50 | 5-15 minutes |
| Full Crawl (10K pages) | ~$15.00 | 15-60 minutes |

**Always confirm with user before:**

- Light Crawl exceeding 20 pages
- Any Full Crawl (involves Bright Data API costs)

---

## Examples

- **Section:** "crawl all pages under docs.example.com/api/" → Light Crawl, url_filter `/api/`, depth 3, page limit 30.
- **Whole site:** "crawl the entire site at smallbusiness.com" → Full Crawl, confirm cost first, depth 3, no filter.
- **Scoped competitive:** "get competitor.com's product pages" → Full Crawl, url_filter `"https://competitor\\.com/products/.*"`, depth 2.

---

## Related Workflows

- **FourTierScrape.md:** Single-page scraping used internally by Light Crawl
