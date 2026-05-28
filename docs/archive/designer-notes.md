# Browse Archive — Notes for the Designer

The mockup is great and most of it maps cleanly onto data we already have. A few
things to know so the final design matches what we can actually ship without a
big backend lift. Items are flagged **✅ have the data**, **⚠️ needs a decision**,
or **❌ no data (avoid for v1)**.

## Data reality check (per mockup element)

- ✅ **Story rows** (source, title, snippet, section tag, link) — all present.
- ✅ **Source sidebar with counts** (Hacker News 412, r/golang 386, …) — we can
  produce exact per-source counts.
- ✅ **Section tabs** (All, Articles, Discussion, Releases, Show & Tell,
  Trending, Videos, Security, Jobs) with counts — these come from our section
  tags. Note: our canonical sections are Release, Proposal, Conference,
  Discussion, Event, Article, Tutorial, Video, Trending, Security, Jobs.
  "Show & Tell" and "Podcasts" aren't distinct sections today (podcasts fold
  into Video). Either rename tabs to our sections, or tell us the exact tab set
  you want and we'll map it.
- ✅ **"Stories indexed" total** + **"Next digest in"** countdown — fine.
  ("Next digest in" is derived from our send schedule, not stored data.)
- ✅ **Sort: New / Top** — New = newest published, Top = our relevance score.
- ⚠️ **Sort: "Hot"** — we can do a simple recency-weighted score. It won't be a
  true engagement-velocity "hot" unless we back it with click data later. Fine
  to show; just know what's behind it.
- ❌ **Sort: "Discussed"** + **comment counts** ("142", "38" on rows) — we do
  **not** store comment counts. Please **drop the Discussed tab and comment
  numbers for v1**, or treat them as a later addition.
- ❌ **"+47 today" / "Pulled in last hour"** stat cards — we don't record *when*
  we collected an item (only when it was published). These need a small schema
  change. Decision needed: add it, or **drop these two stat cards for v1** and
  keep just "Stories indexed" + "Next digest in".
- ❌ **Multiple hashtag tags per story** (`#go1.25` `#release`, `#vulnerability`
  `#net/http`) — each story has **one** section tag, not free-form tags. For v1
  show the single section tag as one chip. Multi-tag chips would need a new
  tags system.
- ⚠️ **Per-row score/vote number** (the `268`, `213` in the left gutter) — we
  have a `score` (a float relevance score, not upvotes). We can show a derived
  number, but it's not a community vote count. Decide if you want to show it,
  reframe it, or drop it.
- ⚠️ **Read time ("3 min")** — not stored. Either drop, or we estimate from
  snippet/title length (rough). Your call.
- ⚠️ **Trending sidebar ("Trending right now")** — for v1 this is "top stories
  by our score", not real-time click trending. Real engagement-driven trending
  is possible later (we do track clicks per item).
- ⚠️ **Thumbnails/images** — we have an image field but it's sparsely populated.
  Design should look good **without** images as the default; treat thumbnails as
  optional/enhancement.

## The one genuinely new visual element: the "In digest" marker

The biggest product idea here: this page shows **everything we collected**, not
just what went into the newsletter. Only a curated subset per day actually gets
sent. We want a clear visual marker on rows that **made it into a sent digest**
vs. the raw firehose that didn't.

Please design this marker:

- It's a per-row boolean ("made the digest" / "didn't").
- Could be a chip ("In digest" / "Featured"), an icon, a left accent bar, a
  subtle background — your call.
- Consider an optional filter/toggle: "Only digest picks" vs "Everything".
- Optional: the marker could link to the specific issue it appeared in. Tell us
  if you want that so we wire the link.

This distinction (raw archive vs. curated digest) is the page's main reason to
exist beyond the existing `/issues/` archive — worth making it legible.

## Layout / behaviour notes

- Design **empty states** (no results for a filter combo) and the **loading**
  state for live filtering.
- Plan the **responsive collapse**: 3 columns (sources · feed · trending) →
  single column on tablet/mobile. Indicate what the sidebars become on mobile
  (drawer? accordion? hidden?).
- Tabs row likely overflows on mobile — horizontal scroll vs. dropdown?
- The `⌘K` search hint implies a keyboard palette — confirm you want that.
- List vs. grid view toggle is in the mockup — confirm both are wanted for v1.

## Summary of decisions we need from you

1. Exact **tab set** and how it maps to our sections (esp. Show & Tell,
   Podcasts).
2. Keep or drop: **Discussed sort**, **comment counts**, **"today"/"last hour"
   stat cards**, **read time**, **per-row score number**, **multi-tags**.
3. Design for the **"In digest" marker** (style + optional issue link + optional
   filter).
4. **Mobile** behaviour for the two sidebars and the tab row.
