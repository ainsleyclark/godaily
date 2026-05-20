# GoDaily Launch Playbook

Copy-paste ready content for every channel. Execute in order on launch day.

---

## Pre-Launch Checklist (D-3)

- [ ] GitHub README polished (hero GIF, subscribe badge, source list, self-hosting steps)
- [ ] 3–5 example newsletters publicly archived on the site
- [ ] UTM links ready (see bottom of this file)
- [ ] Demo GIF created (600px wide, ≤5MB, 15–30s loop)
- [ ] Newsletter screenshot (annotated with AI scores visible)
- [ ] Dev.to article published (D-1)
- [ ] YouTube video uploaded unlisted (D-1)

---

## Launch Day Order

1. **07:00 UTC** — Hacker News (Show HN)
2. **09:00 UTC** — Reddit r/golang
3. **09:30 UTC** — Twitter/X thread
4. **09:30 UTC** — Bluesky post
5. **10:00 UTC** — LinkedIn post
6. **12:00 UTC** — YouTube video goes public
7. **14:00 UTC** — Email Golang Weekly
8. **D+1** — Gopher Slack + Discord

---

## 1. Hacker News — Show HN

**Best time:** Tuesday or Wednesday, 07:00–09:00 AM PST

**Title** (copy exactly):
```
Show HN: GoDaily – Free daily Go newsletter, 18+ sources ranked by Claude AI
```

**Body:**
```
I built GoDaily (https://godaily.dev) after getting frustrated piecing together Go news from a dozen different sources every morning.

It runs a pipeline that fetches from 18+ sources (Go blog, r/golang, Hacker News Go posts, GitHub release feeds, Go proposals, community blogs), deduplicates across them, then uses Claude to score each item on relevance, novelty, and signal quality. The top items get summarized and land in your inbox before 9am on weekdays.

It's completely free, MIT licensed, and self-hostable — the full source is on GitHub. No paid tier, no drip campaigns. I'm a solo dev and I run it because I want to read it myself.

A few things I'd love feedback on:
- The AI ranking: does scoring by "signal quality" feel useful, or is it just noise reduction theater?
- Source coverage: what Go sources do you read that I'm probably missing?
- The format: daily is aggressive — would you prefer 3x/week?

GitHub: https://github.com/ainsleyclark/godaily
```

**Stay online for 3 hours after posting. Respond to every comment within minutes — early velocity is everything on HN.**

Prepared answers for likely questions:
- *"Why not just use Golang Weekly?"* — "Golang Weekly is weekly and curated by one person. GoDaily is daily, multi-source, and the curation is automated so it scales."
- *"How does the Claude scoring work?"* — Explain your prompt/criteria in detail.
- *"Isn't this just a feed aggregator?"* — "The deduplication + AI ranking is the product. Without it you'd need to subscribe to 18 things and do the triage yourself."

---

## 2. Reddit — r/golang

**Best time:** Tuesday, 09:00–10:00 AM UTC

**Title:**
```
I built a free daily Go newsletter that aggregates 18+ sources and uses Claude AI to rank them — open source, no spam, self-hostable
```

**Body:**
```
Like most of you I was drowning in Go content. Golang Weekly is great but weekly. Reddit is noisy. Twitter/X is chaos. I kept missing good posts.

So I built GoDaily (https://godaily.dev) — a free weekday newsletter that:

- Pulls from 18+ sources (blogs, Reddit, HN, GitHub releases, Go proposals, conference talks, YouTube)
- Uses Claude AI to score and rank every item by relevance, novelty, and signal quality
- Deduplicates across sources so you never see the same link twice
- Hits your inbox before 9am, before standup
- Is MIT licensed and fully self-hostable if you want to run your own version

It's genuinely free — no paid tier planned, no drip campaigns, no "upgrade for full access." The whole pipeline is open on GitHub.

I've been running it for a few weeks and wanted to share it with the community that inspired it.

Would love your honest feedback — especially on what sources I'm missing or what makes a Go newsletter actually useful to you.

[godaily.dev](https://godaily.dev?utm_source=reddit) | [GitHub](https://github.com/ainsleyclark/godaily)
```

*Attach: annotated screenshot of a sample email*

---

## 3. Reddit — r/programming (D+5)

**Title:**
```
I used Claude AI to build a self-curating Go developer newsletter — here's how the ranking works
```

**Body:** Link to your Dev.to technical article. Lead with the architecture/AI angle, not the newsletter angle.

---

## 4. Twitter/X — Launch Thread

Post as a thread (5 tweets). Give each tweet a moment to breathe before posting the next.

**Tweet 1:**
```
I built a thing Go devs have been asking for.

A free daily newsletter that pulls from 18+ sources, uses Claude AI to rank every item, and hits your inbox before standup.

No spam. No paid tier. MIT licensed. Self-hostable.

🧵 Here's how it works and why I built it:
```

**Tweet 2:**
```
Every morning I was doing the same ritual:
→ Check r/golang
→ Scan HN for Go posts
→ Read 4 Go blogs
→ Check GitHub releases
→ Miss half of it anyway

There's great Go content everywhere. The problem is aggregation, not supply.
```

**Tweet 3:**
```
GoDaily fetches from 18+ sources every weekday:
- Go blog + proposals
- r/golang top posts
- Hacker News Go items
- GitHub release feeds
- Community blogs
- Conference talks

Claude scores every item for relevance + signal quality. Top items get summarized. You get one email.
```

**Tweet 4:**
```
What makes it different from Golang Weekly:

✅ Daily (not weekly)
✅ AI-ranked (not human-curated)
✅ 18+ sources (not one curator's bookmarks)
✅ Self-hostable (MIT license)
✅ Free forever (not "free tier")
✅ Before 9am (not "sometime this week")
```

**Tweet 5:**
```
It's live at godaily.dev

If you want Go news without the noise — subscribe.
If you want to run your own version — fork it on GitHub.
If you have feedback — reply here.

Building this in public. Follow along. 🔨
```

**Attach hero banner image to Tweet 1.**

**Hashtags to add to Tweet 5:** `#golang #gophercon #gophers #go #newsletter #buildinpublic #opensource`

---

## 5. Bluesky

**Single post (not a thread):**
```
Just launched GoDaily — a free daily Go newsletter.

18+ sources → Claude AI ranking → one email before 9am.

No spam. No paid tier. MIT licensed. Self-hostable.

godaily.dev

Feedback welcome from the gopher community here. 🐹
```

---

## 6. LinkedIn

**Post (not article — native posts get more reach):**

```
I spent the last few months building a tool I use every morning before standup.

I was tired of checking Reddit, Hacker News, four Go blogs, and GitHub releases just to stay current on the Go ecosystem. There had to be a better way.

So I built GoDaily (godaily.dev) — a free weekday newsletter for Go developers that:

→ Pulls from 18+ community sources
→ Uses Claude AI to rank and summarize every item by relevance and signal quality
→ Deduplicates across all sources so you never see the same link twice
→ Lands in your inbox before 9am, before standup

It's free. No paid tier, no drip campaigns, no upsells. The source code is MIT licensed and fully self-hostable on GitHub.

If you're a Go developer, engineering lead, or just someone who wants to stay current without spending 30 minutes on it every morning — it might be worth a look.

godaily.dev

#golang #softwaredevelopment #newsletter #opensource #ai
```

*Attach hero banner image. Reply to every comment for 48 hours.*

---

## 7. Golang Weekly Submission

**Email:**

```
To: [editor address from golangweekly.com]
Subject: Submission: GoDaily — Free daily Go newsletter with AI curation

Hi,

I wanted to share GoDaily (https://godaily.dev) for potential inclusion in Golang Weekly.

GoDaily is a free daily newsletter for Go developers that aggregates 18+ community sources, uses Claude AI to rank and summarize content by relevance and signal quality, and delivers a deduplicated digest before 9am on weekdays.

It's MIT licensed and self-hostable: https://github.com/ainsleyclark/godaily

Happy to provide more details if useful. Thank you for everything you do for the Go community.

Best,
Ainsley Clark
@ainsleydev
```

---

## 8. Gopher Slack (D+1)

**Channels:** `#tools`, `#golang` (check rules first)

```
Hey folks — I just launched GoDaily (godaily.dev), a free weekday newsletter that aggregates 18+ Go sources and uses Claude AI to rank/summarize content before it lands in your inbox at 9am.

MIT licensed, self-hostable, no spam, no drip campaigns.

Would love feedback from this community on what sources I might be missing.

GitHub: github.com/ainsleyclark/godaily
```

---

## 9. Discord (D+1 to D+3)

**Target servers:** Gophers Discord, The Primeagen's Discord, any Go-adjacent servers you're in.
**Find:** `#show-and-tell`, `#projects`, or `#tools` channel.

```
Built something for Go devs:

**GoDaily** — free daily newsletter, 18+ sources ranked by Claude AI

→ godaily.dev
→ github.com/ainsleyclark/godaily

MIT licensed, self-hostable, no spam. Feedback welcome.
```

*Attach demo GIF directly — Discord renders it inline.*

---

## Dev.to Article Outline (publish D-1)

**Title:** `I built a daily Go newsletter powered by Claude AI — here's how the curation pipeline works`

**Structure:**
1. **The problem** (200 words) — The daily ritual of checking too many sources
2. **What GoDaily does** (300 words) — Feature overview with newsletter screenshot
3. **The architecture** (500 words) — Fetch → deduplicate → Claude score → email; include diagram
4. **The Claude scoring methodology** (400 words) — Criteria, weighting, sanitized prompt structure
5. **What I learned** (200 words) — Honest reflection, what surprised you
6. **Try it** (100 words) — Subscribe link + GitHub (minimal CTA)

**Tags:** `#go #golang #newsletter #showdev #claude`

---

## YouTube Video Outline (upload D-1, go public D-Day)

**Title:** `GoDaily: Free AI-powered Go newsletter (self-hostable, MIT license)`

```
0:00 – 0:30   Hook: show the morning ritual problem
0:30 – 1:30   Demo: scroll through a real email, point out AI scores
1:30 – 3:00   Architecture: screen share of GitHub repo + diagram
3:00 – 4:00   Self-hosting: 60-second local setup walkthrough
4:00 – 4:30   Subscribe CTA + GitHub link
```

---

## Ongoing Twitter/Bluesky Content (D+1 to D+14)

| Day | Post |
|-----|------|
| D+1 | Screenshot of today's newsletter — "Here's what Go devs were talking about on [date]" |
| D+3 | Thread: "How Claude AI scores Go content — the exact criteria I use" |
| D+5 | Stats: "X sources checked, Y items deduplicated, Z made the cut — in today's GoDaily" |
| D+7 | Thread: "What I learned building a Go newsletter for a week" |
| D+10 | Question: "What Go source am I definitely missing?" |
| D+14 | Two-week retrospective with open metrics |

---

## Email Footer (add to every issue)

```
Know a Go developer drowning in RSS feeds? Forward this email.
If they subscribe and mention your name, I'll add you to the GoDaily Gopher Hall of Fame on GitHub. ⭐
```

---

## UTM Links (use these everywhere)

| Channel | Link |
|---------|------|
| Hacker News | `https://godaily.dev?utm_source=hackernews` |
| Reddit | `https://godaily.dev?utm_source=reddit` |
| Twitter/X | `https://godaily.dev?utm_source=twitter` |
| Bluesky | `https://godaily.dev?utm_source=bluesky` |
| LinkedIn | `https://godaily.dev?utm_source=linkedin` |
| Dev.to | `https://godaily.dev?utm_source=devto` |
| YouTube | `https://godaily.dev?utm_source=youtube` |
| Email | `https://godaily.dev?utm_source=email` |

---

## Week 1 Benchmarks

- **200–500 subscribers** — Expected for a well-executed solo launch
- **800–1,500** — Excellent; one channel resonated strongly
- **Step-change** — Golang Weekly inclusion can add 500–2,000 in a single week

## Media Assets Needed

| Asset | Specs |
|-------|-------|
| Hero banner | 1200×628px PNG |
| Newsletter screenshot | Full email, annotated |
| Demo GIF | 600px wide, ≤5MB, 15–30s loop |
| Architecture diagram | PNG, 1200px wide (use Excalidraw) |
| YouTube thumbnail | 1280×720px |
| Demo video | 1080p, 3–5 min screen recording |
