---
name: weekly-metrics
description: >
  Run the weekly GoDaily metrics analysis — email engagement, social post
  performance, subscriber growth, and top content — then produce a structured
  summary with actionable suggestions. Use when the user wants a performance
  report, engagement summary, weekly review, or asks how GoDaily is performing.
  Trigger on phrases like "weekly metrics", "how are we performing", "metrics
  report", "analyse the last two weeks", "engagement summary", "how is the
  newsletter doing", or "weekly review".
---

# GoDaily Weekly Metrics Analysis

Analyse GoDaily performance for the requested period and produce an actionable
summary. If the user passes a period as an argument, use it. Otherwise default
to the last 14 days.

## Step 1 — Resolve the date range

Use today's date to compute `FROM` (14 days ago) and `TO` (today). Example for
2026-05-31: `from=2026-05-17&to=2026-05-31`.

If `$ARGUMENTS` contains a period keyword (`week`, `month`, `year`) use the
`period=` parameter instead of `from`/`to`. If it contains explicit dates
(`from=YYYY-MM-DD to=YYYY-MM-DD`), use those.

## Step 2 — Fetch all metrics in parallel

Run these six curl commands. Always source the key first:

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"
RANGE="from=FROM&to=TO"   # substitute computed dates

curl -sL -H "Authorization: Bearer $GODAILY_API_KEY" "$BASE/api/metrics/summary?$RANGE"
curl -sL -H "Authorization: Bearer $GODAILY_API_KEY" "$BASE/api/metrics/issues?$RANGE&sort=click_rate&limit=20"
curl -sL -H "Authorization: Bearer $GODAILY_API_KEY" "$BASE/api/metrics/subscribers?$RANGE&bucket=day"
curl -sL -H "Authorization: Bearer $GODAILY_API_KEY" "$BASE/api/metrics/items?$RANGE&limit=10"
curl -sL -H "Authorization: Bearer $GODAILY_API_KEY" "$BASE/api/metrics/tags?$RANGE&limit=10"
curl -sL -H "Authorization: Bearer $GODAILY_API_KEY" "$BASE/api/metrics/sources?$RANGE&limit=10"
curl -sL -H "Authorization: Bearer $GODAILY_API_KEY" "$BASE/api/metrics/social?$RANGE"
```

## Step 3 — Present the analysis

Structure your response as follows:

### Email Performance

Lead with a summary table:

| Metric | Value | Benchmark |
|---|---|---|
| Issues sent | N | — |
| Avg open rate | X% | Industry avg: 20–30% |
| Avg click rate | X% | Industry avg: 2–5% |
| Bounces / Complaints | N / N | — |
| Unique subscribers engaged | N | — |

Then a per-issue breakdown table sorted by CTR (highest first):

| Issue | Delivered | Open Rate | CTR |
|---|---|---|---|
| YYYY-MM-DD | N | X% | X% |

Call out the best and worst performing issue and what might explain the gap
(content type, day of week, list size at time of send).

### Subscriber Growth

- Net new active subscribers over the period
- Total active at end of period
- Any significant single-day spikes (≥10 new in a day) — note them as potential
  acquisition events worth investigating
- Unconfirmed gap: compare total `new` signups vs total `confirmed` over the
  period. If confirmation rate is below 85%, flag it.

### Top Content

**By tag** — present as a ranked list with click counts.

**By source** — present as a ranked list. Call out the top 3 and note if any
source is punching above or below expectations.

**Top 5 items** — title, source, clicks, one-line note on why it likely
resonated (content type pattern).

### Social Performance

For each platform (LinkedIn, Mastodon, Bluesky) list posts with their
impressions, likes, reposts. Highlight the single best-performing post.

Note any platform with consistently 0 impressions — this may indicate a
tracking gap or a dead audience.

### Suggestions

Based on the data, generate 4–6 specific, actionable suggestions under two
headings:

**To grow subscriber count:**
Focus on: acquisition spikes (what caused them and how to replicate), social
channels driving reach, confirmation rate improvements.

**To improve engagement:**
Focus on: content types/tags/sources with the highest CTR, post formats that
convert on social, underperforming slots that could be cut or improved.

Keep suggestions data-driven: reference the specific numbers that support each
recommendation. Avoid generic advice.
