# SEO methodology

Tool-agnostic. This is the reasoning layer; `SKILL.md` maps it onto the
Ubersuggest tools.

## The pyramid: technical → content → authority

Work bottom-up. Effort spent on the wrong layer is wasted.

1. **Technical.** Can search engines and LLMs crawl, render and index the site,
   fast enough? Broken canonicals, noindex left in production, 4xx/5xx, slow
   LCP, no sitemap, JS-only content. A brilliant article on an unindexable page
   earns nothing. This layer is mostly *fix once, benefit everywhere*.
2. **Content.** Does a page exist that matches the query's intent better than
   what currently ranks? This is where most of the upside lives for most sites.
3. **Authority.** Do other credible sites vouch for this one? Slowest and least
   controllable layer. Chasing links before layers 1 and 2 are sound is the
   most common way to waste an SEO budget.

Diagnostic shortcut: if pages are indexed but not ranking, the problem is
content or authority. If pages rank but traffic is flat, the problem is intent
match or SERP features. If pages are not indexed at all, it is technical.

## Prioritisation: impact × effort

Score every recommendation on both axes and sort. Concretely:

- **Impact** ≈ (realistic traffic gain) × (business value of that traffic).
  Traffic gain comes from volume × achievable CTR at an achievable position.
  Business value is why a 200-volume commercial keyword beats a 20,000-volume
  informational one.
- **Effort** = engineering + content + link work, honestly estimated.

Then present in this order:

1. **Quick wins** — high impact, low effort. Pages already ranking 4–15 for a
   decent keyword and just needing a better title, more depth or internal
   links. Always lead with these; they buy credibility for the rest.
2. **Projects** — high impact, high effort. New content clusters, site
   migrations, an information-architecture rework.
3. **Hygiene** — low impact, low effort. Batch it, don't dwell on it.
4. **Drop** — low impact, high effort. Say explicitly that it is not worth
   doing, and why.

Never hand over a flat list of 40 issues. A prioritised list of 8 with a stated
reason for the order is more useful and more likely to get done.

## Keyword clustering and intent

A cluster is a set of queries that one page can satisfy — because they share
intent, not merely words. "best running shoes", "top running shoes 2026" and
"running shoes ranked" are one page. "best running shoes" and "running shoe
size chart" are two, despite the overlap.

Practical test: look at who ranks. If the top results for two queries are
substantially the same URLs, one page can serve both. If they differ, Google
reads the intents as different, and you need separate pages.

The four intents and what they need:

| Intent | Query shape | Page type |
| --- | --- | --- |
| Informational | how, what, why, guide | Article, tutorial, explainer |
| Commercial | best, top, review, vs, alternatives | Comparison, listicle, review |
| Transactional | buy, price, pricing, discount, near me | Product, category, pricing page |
| Navigational | brand name, brand + login | Homepage, specific brand page |

Mismatching these is the single most common reason good content does not rank.

## Topical authority

Search engines and LLMs both reward evident depth on a subject over scattered
one-off posts. One page on a topic looks accidental; twelve interlinked pages
covering the topic and its adjacent questions look authoritative, and they
reinforce each other through internal links.

So: plan in clusters, not in posts. Pick a pillar topic the business genuinely
has a right to own, map its sub-questions, publish across them, and link them
to a hub page. Breadth without depth — one post each across thirty unrelated
topics — is the pattern that reliably underperforms.

## AEO / GEO: getting cited by AI answers

Answer Engine Optimisation (ChatGPT, Perplexity, Google AI Overviews, Claude)
overlaps with SEO but scores differently.

What changes:

- **Citation replaces the click.** The win is being *the source the model
  quotes*, whether or not anyone clicks through. Rank alone doesn't guarantee
  it — models cite pages that are easy to extract from.
- **Extractability matters more.** Clear headings that mirror real questions, a
  direct answer in the first paragraph under each heading, short factual
  sentences, tables for comparisons, explicit dates and numbers. Burying the
  answer under 600 words of preamble costs you the citation.
- **Entities over keywords.** Models reason about things and their relations.
  Name your entities explicitly and consistently — product names, categories,
  competitors, locations — instead of leaning on pronouns and synonyms.
- **Off-site consensus counts.** Models synthesise across sources, so what
  third-party sites, reviews, forums and directories say about a brand shapes
  the answer as much as the brand's own site does. A brand invisible in reviews
  and roundups tends to be invisible in AI answers.
- **Measurement is share-of-voice, not position.** The questions are: for the
  prompts that matter, how often does the brand appear, at what rank among
  mentions, with what sentiment, and which competitors appear instead. That is
  what the AI Search Visibility tools measure.

What does *not* change: crawlable, fast, factually accurate pages. AEO is a
layer on a healthy site, not a replacement for one.
