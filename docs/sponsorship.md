# Sponsor GoDaily

**Reach engaged Go developers every weekday morning — before standup.**

GoDaily is a daily email newsletter for Go developers. Every weekday it gathers
the day's news from across the Go community, uses Claude AI to rank and
summarise what actually matters, and delivers one short, high-signal read.

This document is both the **one-pager** (the pitch, below) and the **media kit**
(audience, inventory, pricing and booking, further down). Stats marked
`{{LIKE_THIS}}` are placeholders — fill them from the live metrics API before
sending to a sponsor (`GET /api/metrics/summary`, `GET /api/digest/subscribers`).

---

## The one-pager

### Who reads GoDaily

Working Go developers, backend and platform engineers, devops and SRE
practitioners, and engineering leads — the people who choose, buy and influence
the tools their teams ship with. It's a hard audience to reach and a high-intent
one to put your product in front of.

### Why it's a good place to advertise

- **A pure-Go, pure-developer audience.** No general-interest dilution. Every
  reader self-selected into a daily Go newsletter.
- **High engagement.** A short, curated, no-noise format that readers open by
  habit — `{{OPEN_RATE}}` open rate, `{{CLICK_RATE}}` click rate.
- **Trusted editorial.** Every story is ranked and summarised for signal, so the
  newsletter carries credibility your placement borrows from.
- **Daily cadence.** ~22 issues a month means flexible scheduling and the option
  to run a sustained campaign rather than a one-off.

### The numbers at a glance

| Metric | Value |
|---|---|
| Confirmed subscribers | `{{SUBSCRIBERS}}` |
| Open rate | `{{OPEN_RATE}}` |
| Click rate | `{{CLICK_RATE}}` |
| Cadence | Daily, weekday mornings (skips weekends & public holidays) |
| Also published to | Web archive at [godaily.dev](https://godaily.dev) + social feeds |
| Audience | Go developers, backend/platform/devops engineers, eng leads |

### Get in touch

Email **home@ainsley.dev** with the product you'd like to promote and the dates
you're interested in. See pricing and formats below.

---

## Media kit

### Audience profile

GoDaily reaches developers who work with Go day to day. Typical readers:

- Backend, platform and infrastructure engineers
- Devops / SRE / cloud practitioners
- Engineering leads and founders making tooling decisions
- Open-source maintainers and contributors in the Go ecosystem

This maps directly onto the buyers and champions for cloud platforms, databases,
observability, CI/CD, auth, payments and developer-tooling products.

### Reach & engagement

| Metric | Value | Notes |
|---|---|---|
| Confirmed subscribers | `{{SUBSCRIBERS}}` | Double opt-in; confirmed addresses only |
| Monthly issues | ~22 | Weekday mornings |
| Open rate | `{{OPEN_RATE}}` | Trailing 30 days |
| Click rate | `{{CLICK_RATE}}` | Trailing 30 days |
| Subscriber growth | `{{NET_GROWTH_30D}}` / mo | Net new confirmed, trailing 30 days |
| Web archive | Every issue, permanently | Adds long-tail impressions beyond the send |

> These figures are pulled from GoDaily's own metrics pipeline (Resend delivery
> events + click tracking). We share a current snapshot with any serious
> prospect.

### How GoDaily is built (why placements are trusted)

GoDaily runs a fully automated, editorially-ranked pipeline every weekday:

1. **Collect** — 18 community sources (Hacker News, r/golang, Lobsters, the Go
   Blog, Go release notes, GitHub releases & trending, Dev.to, Medium, YouTube,
   Go Podcast, Fallthrough, Ardan Labs, JetBrains, golang-nuts, Golang Bridge,
   Awesome Go, Mastodon).
2. **Deduplicate** — the same story across multiple sources is merged.
3. **Rank** — Claude scores every item on relevance, novelty and signal.
4. **Summarise** — only the highest-signal items make the issue.
5. **Deliver** — email, web archive, and across GoDaily's social feeds.

No drip campaigns, no upsells, no tracking pixels for readers — which is exactly
why the audience stays engaged and why a clearly-labelled sponsor slot performs.

### Ad formats & inventory

Each issue has one **primary sponsor slot** plus optional secondary placements.
All placements are clearly labelled as sponsored.

| Placement | Position | Spec |
|---|---|---|
| **Primary slot** | Top of the issue, above the stories | Logo + heading (≤60 chars) + 30–50 words of body + CTA link |
| **Secondary slot** | Mid-issue, between sections | Heading + 20–30 words + CTA link |
| **Classified / text link** | Footer block | One line, ≤100 chars + link |
| **Web archive** | Persists on the issue's permanent page | Bundled with email placement |

Creative guidance:
- Plain-text-friendly, dev-appropriate copy converts best — no hype, lead with
  the technical value.
- Provide a destination URL; we append UTM parameters and report clicks.
- We reserve the right to decline creative that doesn't fit the audience.

### Pricing

Sponsorship is priced per issue. Two ways to buy:

**1. Flat rate (recommended to start)**

| Placement | Price / issue |
|---|---|
| Primary slot | `{{PRIMARY_RATE}}` |
| Secondary slot | `{{SECONDARY_RATE}}` |
| Classified / text link | `{{CLASSIFIED_RATE}}` |

**2. CPM (cost per 1,000 sends)**

For larger or programmatic buys, the primary slot is available at
`{{CPM}}` CPM. Developer newsletters typically price in the **$50–$100+ CPM**
range; the exact rate reflects current list size and engagement.

> Quick maths: at a `{{CPM}}` CPM and `{{SUBSCRIBERS}}` subscribers, a single
> primary placement is roughly `{{PRIMARY_RATE}}`.

**Packages & discounts**

| Package | Detail | Discount |
|---|---|---|
| Weekly | 4–5 consecutive primary slots | `{{WEEKLY_DISCOUNT}}` |
| Monthly | 8+ primary slots in a month | `{{MONTHLY_DISCOUNT}}` |
| First-time | Intro rate on a single test placement | `{{INTRO_RATE}}` |

### Who advertises on GoDaily

Products that want to reach Go developers — their buyers and champions are our
readers:

- **Cloud & deploy platforms** — Google Cloud, AWS, DigitalOcean, Fly.io,
  Railway, Render
- **Databases & data infra** — CockroachDB, PlanetScale, Neon, Redis, MongoDB,
  SurrealDB, ClickHouse, Confluent/Kafka, NATS
- **Observability & reliability** — Datadog, Grafana, Sentry, Honeycomb,
  Better Stack
- **Backend & developer platforms (Go-heavy)** — Temporal, Encore, Buf/Connect,
  Tailscale, Docker, Kubernetes-ecosystem vendors
- **Auth, payments & infra SaaS** — Stripe, WorkOS, Clerk, Ory, LaunchDarkly
- **Tooling & IDEs** — JetBrains (GoLand)
- **Hiring & community** — companies recruiting Go engineers, Go job boards,
  conferences (e.g. GopherCon), courses and books

### Booking & process

1. **Enquire** — email **home@ainsley.dev** with your product, target dates and
   preferred placement.
2. **Confirm** — we check fit and availability and send a current stats snapshot.
3. **Creative** — you send copy + a destination URL (or we draft it for review).
4. **Run** — the placement ships in the issue and lives on the web archive.
5. **Report** — we share delivery and click stats after the send.

**Availability:** one primary slot per issue, booked first-come. Lead time of a
few business days is appreciated.

---

### Maintainer note — filling in the placeholders

Pull live figures and replace the `{{TOKENS}}` before sending this to a sponsor:

```bash
source ~/.zshrc 2>/dev/null
BASE="${GODAILY_API_URL:-https://godaily.dev}"

# {{SUBSCRIBERS}}
curl -sf -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/digest/subscribers?page=1&per_page=1" | jq '.data.total'

# {{OPEN_RATE}}, {{CLICK_RATE}}
curl -sf -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/metrics/summary?period=month" | jq '{open_rate, click_rate}'

# {{NET_GROWTH_30D}}
curl -sf -H "Authorization: Bearer $GODAILY_API_KEY" \
  "$BASE/api/metrics/subscribers?period=month&bucket=month" | jq '.[] .net_change'
```

Rate of thumb for `{{PRIMARY_RATE}}` from a CPM: `subscribers / 1000 * CPM`.
Developer newsletters sit at the premium end ($50–$100+ CPM) because the audience
is targeted and hard to reach. Quote a flat per-issue rate until there's enough
data to justify CPM pricing.
