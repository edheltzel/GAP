---
name: site-audit
description: >
  Crawl a site for technical SEO problems and report prioritised fixes with
  Core Web Vitals. Use when the user asks for a site audit, a technical SEO
  check, why their site is slow, whether their site has SEO errors, about
  broken links, missing titles or H1s, duplicate content, Core Web Vitals,
  PageSpeed or a health score.
argument-hint: "<domain>"
---

# Site audit

**Input:** `$ARGUMENTS` — the domain to crawl. If that placeholder is empty or
still literal, take the domain from what the user asked; if they named none, ask
which domain.

## Before you start: this needs a login

`site_audit`, `site_audit_status`, `site_audit_results` and `site_audit_pages`
all require an authenticated Ubersuggest account. **Call `auth_status` first.**

If the user is not logged in: tell them the crawl needs a connected Ubersuggest
account, ask for the connection, then offer `pagespeed_audit` — it is *not*
login-gated and still delivers Core Web Vitals for the domain. Do not fire the
audit tools just to surface a raw auth error, and do not stand in for the crawl
by fetching pages yourself: a crawl of 340 pages is not something you can eyeball.

## Steps

1. **Validate the domain.** `validate_site` if the input is at all doubtful.
   Pass the root domain (`example.com`). To audit a single page instead, pass
   the domain plus the page path in `path` — and then use that same `path` in
   *every* subsequent call.

2. **Start the crawl.** `site_audit` with the domain. `crawlMaxPages` defaults
   to 150 (free tier); pass a higher value only if the user's plan allows it.
   Use `recrawl: true` only when the user explicitly wants fresh data — it
   bypasses the cache and costs a full crawl.

3. **Poll until done.** `site_audit_status` with the *same* `domain`, `path` and
   `crawlMaxPages` you passed in step 2 — mismatched arguments look up a
   different report. Watch two fields:
   - `result.done` — `false` while crawling, `true` when the report is ready.
   - `result.status` — `'no_errors'` on success; **any other value means the
     crawl failed**, so stop and report it rather than polling on.

   Wait several seconds between polls and **cap at ~10 attempts**. If it is
   still crawling, hand over the partial report (the payload is populated while
   crawling) and say the crawl had not finished.

4. **Read the issue breakdown.** The finished report gives:
   - `overview` — health score and totals.
   - `issues_per_category` — `{errors, warnings, recommendations}`, each with
     issue **ids** and counts.

5. **Drill into what matters.** `site_audit_results` with the domain and an
   `issue` id. The id must come from `issues_per_category` (e.g.
   `seo_missing_h1`) — it is not a free-text label, so never invent one. Pull
   the top 3–6 issues by impact, not all of them. Each call returns the
   affected URLs with issue-specific detail.

6. **Add performance.** `pagespeed_audit` for Core Web Vitals. Pass `devices`
   if the user cares specifically about mobile or desktop; mobile is the more
   common problem. This is async — poll within the same cap.

7. **Optional inventory.** `site_audit_pages` lists every crawled URL. Only
   worth pulling if the user asks what was crawled or suspects pages are
   missing from the crawl.

## Prioritising the findings

Do not dump the issue list. Rank by real impact:

1. **Blocks indexing** — noindex in production, robots.txt blocks, broken
   canonicals, 5xx. Nothing else matters until these are clear.
2. **Breaks pages or links** — 4xx, broken internal links, redirect chains.
3. **Costs rankings at scale** — missing/duplicate titles and meta
   descriptions, missing H1s, thin or duplicate content, especially where the
   affected count is high.
4. **Core Web Vitals** — poor LCP or CLS, worst on mobile.
5. **Hygiene** — image alt text, minor markup issues. Batch them in one line.

Weight each by how many URLs it hits: one missing H1 is noise, 400 missing H1s
is a template bug and a top finding.

## Deliverable

- **Health score** and totals (errors / warnings / recommendations), plus the
  page count crawled.
- **Top findings table:**

| # | Issue | Severity | Pages affected | Why it matters | Fix |
| --- | --- | --- | --- | --- | --- |

- **Core Web Vitals** — LCP, CLS, INP per device, with pass/fail read out.
- **Fix order** — a short numbered list, quick wins first, calling out anything
  that is one template change fixing hundreds of URLs.
- Example URLs for each finding, so the user can verify it themselves.

## When something fails

- `result.status` is not `'no_errors'` → the crawl failed. Report the status
  value; do not present partial data as a completed audit.
- Poll cap reached with `done: false` → deliver the partial report, labelled as
  partial.
- Not logged in → the fallback in *Before you start*.
- Quota / plan error → say which limit was hit and that the crawl did not run.
