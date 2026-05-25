# Future uses for the synth (AI) package

The `internal/synth` package currently has one job: take the day's
scored news and draft one short, punchy post about the single most
notable item. The Anthropic client, cached system prompt, and JSON
parsing scaffolding are general-purpose, so there is room to extend
them without restructuring the package.

This doc captures candidate extensions, ordered by ROI, along with the
main tradeoffs.

## 2. Semantic dedup / clustering across sources

When HN, Reddit, golangbridge, and Lobsters all cover the same release
or proposal, the digest currently shows the same story up to four
times. Scoring is purely numeric (`internal/news/score.go`) and has no
notion of topic identity.

- **Approach:** a separate, cheap call (Haiku) that takes the merged
  item list and returns clusters keyed by topic, with one canonical
  item per cluster.
- **Tradeoff:** adds a second model call to the critical path. Could
  be opt-in behind a flag on `RunOptions` until the clustering proves
  itself.

## 3. Snippet enrichment

Several sources ship weak or empty `Snippet` fields (see
`internal/ingest/enrich.go`). A Haiku-class model could normalise them
from title + URL + whatever excerpt the source provides.

- **Tradeoff:** N model calls per run (one per item) unless batched.
  Worth caching by item URL so we only pay once per item across runs.

## 5. Quality / off-topic filter

Filter out blogspam, listicles, and items that aren't actually about
Go before they hit scoring. Today the `Source` weight is the only
defence against a noisy day on Reddit.

- **Tradeoff:** false positives are silent — an item the model wrongly
  rejects never appears in the digest. Log decisions for auditability,
  and start permissive.

## 6. Weekly roundup

A separate prompt + cron entry (Friday or Sunday) that consumes five
days of items and produces a longer-form weekly summary post.

- **Where it lands:** new prompt file in `internal/synth`, new command
  in `cmd/godaily`, new GitHub Actions workflow alongside
  `daily.yaml`.
- **Tradeoff:** requires persistence of past items (Stage 1 from
  `docs/stage-1-database.md` covers this).

## 7. Auto-tagging beyond the source-specific tags

Today `news.Tag` is a small enum populated by the source fetchers
(`article`, `proposal`, `video`, ...). A model pass could add finer
domain tags — `runtime`, `tooling`, `web`, `observability`,
`performance` — that drive better email sectioning and per-tag pages
in Stage 3.

## Cross-cutting tradeoffs

- **Latency and cost:** every new call adds both. Lean on the existing
  cached system block where possible, and prefer Haiku for anything
  that doesn't need Sonnet's quality.
- **Failure tolerance:** the digest must still ship if any AI step
  fails. Today `Run` already logs and continues on `synth` errors;
  keep that pattern for every new step.
- **Determinism in tests:** `client_test.go` already redirects to an
  `httptest.Server`. Any new call should follow that pattern so the
  test suite stays offline.
