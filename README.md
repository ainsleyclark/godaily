<div align="center">

<img src="web/assets/favicon/favicon.png" width="160" alt="GoDaily">

# GoDaily

**Daily Go news, straight to your inbox.**

The best stories from the Go community — ranked, summarised,
and delivered before standup.

[**Subscribe**](https://godaily.dev) · [How it works](#how-it-works) · [Sources](#sources) · [API](#api)

<br>

[![Website](https://img.shields.io/badge/godaily.dev-00ADD8?logo=go&logoColor=white)](https://godaily.dev)
[![Made with Go](https://img.shields.io/badge/Made%20with-Go-00ADD8.svg?logo=go)](https://go.dev)
[![Go Report Card](https://goreportcard.com/badge/github.com/ainsleyclark/godaily)](https://goreportcard.com/report/github.com/ainsleyclark/godaily)
[![Twitter](https://img.shields.io/twitter/follow/ainsleydev)](https://twitter.com/ainsleydev)

</div>

---

**GoDaily** is a free daily newsletter for Go developers. Every weekday morning it
gathers the day's news from across the Go community, uses Claude AI to rank and
summarise what actually matters, and sends you one short read — before standup.

No drip campaigns. No upsell. No tracking pixels. Read it by email, or browse
every issue at **[godaily.dev](https://godaily.dev)**.

## Why GoDaily

- **Every corner of the Go community** — Hacker News, r/golang, Lobsters, the Go
  Blog, GitHub releases, YouTube and more, deduplicated into a single read.
- **Ranked, summarised, opinionated** — every story is read, summarised and
  scored for relevance, novelty and signal. Only what matters to a working Go
  developer makes the cut.
- **Lands before your first coffee** — a weekday-morning email that skips
  weekends and public holidays.
- **No noise** — no drip sequences, no "upgrade for full access", no tracking
  pixels. Free, forever.

## How it works

GoDaily runs a fully automated pipeline every weekday:

1. **Collect** — fetches the latest posts from 18 community sources.
2. **Deduplicate** — merges the same story surfaced in multiple places.
3. **Rank** — Claude scores every item on relevance, novelty and signal quality.
4. **Summarise** — the highest-signal items are written up into a single digest.
5. **Deliver** — the digest lands in your inbox, on the web archive, and across
   GoDaily's social feeds.

## Sources

GoDaily reads from 18 sources across the Go ecosystem:

Hacker News · r/golang · Lobsters · the Go Blog · Go release notes ·
GitHub releases · GitHub trending · Dev.to · Medium · YouTube · Go Podcast ·
Fallthrough · Ardan Labs · JetBrains · golang-nuts · Golang Bridge ·
Awesome Go · Mastodon

## API

Every issue is also available as JSON. Base URL: `https://godaily.dev`

| Endpoint | Description |
|---|---|
| `GET /api/issues` | Paginated list of digest issues (`page`, `per_page`) |
| `GET /api/issues/{slug}` | A single digest issue |
| `GET /api/items/{id}` | A single news item |
| `GET /healthz` | Service health check |

## Built with

Go · [Templ](https://templ.guide) · PostgreSQL · [Resend](https://resend.com) ·
[Anthropic Claude](https://www.anthropic.com) · deployed on [Vercel](https://vercel.com)

---

<div align="center">

### Read the Go community's best, every weekday.

[**Subscribe at godaily.dev →**](https://godaily.dev)

</div>
