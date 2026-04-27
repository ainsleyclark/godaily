# Candidate Sources — Backlog

Pick up as and when. Each row lists the API endpoint (what the fetcher hits)
and the content URL (the human-facing page) so the right URL ends up in
`news.Item.URL` vs `OriginalURL`.

| # | Source                                       | API endpoint | Content URL | Format | Auth | Notes / risk |
|---|----------------------------------------------|---|---|---|---|---|
| 1 | Go release downloads                         | `https://go.dev/dl/?mode=json&include=all` | `https://go.dev/doc/devel/release` | JSON | none | Official Go release JSON. Entries have `version`, `stable`, `files[]`. No release notes in payload — link to `https://go.dev/doc/devel/release#go<version>`. |
| 2 | Mastodon `#golang`                           | `https://mastodon.social/api/v1/timelines/tag/golang?limit=40` | `https://mastodon.social/tags/golang` | JSON | none | Strict `ShouldInclude`: drop boosts (`reblog != null`), require ≥3 favourites. Single-instance dependency. |
| 3 | Gopher Academy YouTube (GopherCon publisher) | `https://www.googleapis.com/youtube/v3/search?part=snippet&channelId=UCx9QVEApa5BKLw9r8cnOFEA&order=date&maxResults=25&key=$YOUTUBE_API_KEY` | `https://www.youtube.com/@GopherAcademy/videos` | JSON | `YOUTUBE_API_KEY` (already set) | Verify channel ID before shipping. Shares YT quota (10k units/day). |
| 4 | Go vulnerability DB                          | Index: `https://vuln.go.dev/ID/index.json` → details: `https://vuln.go.dev/ID/<GO-YYYY-NNNN>.json` | `https://pkg.go.dev/vuln/<GO-YYYY-NNNN>` | JSON (OSV) | none | Two-step fetch (chatty). Detail has `summary`, `details`, `aliases`, `affected[]`. Bound concurrency. |
| 5 | Awesome Go commits                           | `https://api.github.com/repos/avelino/awesome-go/commits?per_page=20` | `https://github.com/avelino/awesome-go` | JSON | reuse `GITHUB_TOKEN` | Commit messages are noisy ("Add foo", "Merge PR…"). Won't surface the new package URL without diff parsing. |
| 6 | The New Stack — Go category                  | `https://thenewstack.io/category/programming-languages/go-programming-language/feed/` | `https://thenewstack.io/category/programming-languages/go-programming-language/` | RSS/XML | none | Editorial Go coverage; weekly cadence. Verify feed is live before shipping. |
| 7 | Bluesky `#golang` search                     | `https://public.api.bsky.app/xrpc/app.bsky.feed.searchPosts?q=%23golang&limit=25` | `https://bsky.app/search?q=%23golang` | JSON | none | Public read API, no auth. Lower volume than Mastodon today — pick one. |
| 8 | JetBrains GoLand blog                        | `https://blog.jetbrains.com/go/feed/` | `https://blog.jetbrains.com/go/` | RSS/XML | none | Official GoLand editorial blog — tutorials, IDE tips, ecosystem coverage. Low volume (few posts/month) but high quality. WordPress feed (standard `<item>` schema with `title`, `link`, `description`, `pubDate`, `dc:creator`). Verify feed is live before shipping. |

## Skipped (and why)

- **Go Time podcast** — show stopped producing new episodes.
- **Stack Overflow `[go]`** — low signal-to-noise for digest.
- **`golang/go` GitHub Releases** — Go doesn't publish to that endpoint; releases live at `go.dev/dl/`.
- **pkg.go.dev trending** — no public trending feed exists; GitHub Trending Go is the closest stable surrogate (already wired up).
- **GitHub Trending (all-langs)** — Go-only trending already covered.

## Adding a new source — quick reference

See `README.md` in this directory for the full step-by-step. Summary:

1. Add constant + `NiceName` + `Priority` in `internal/news/sources.go`.
2. Create `internal/source/<name>.go` with a struct, `Fetch(ctx)`, response types implementing `Transformer` (`Transform`, `ShouldInclude`, `EnrichmentURL`), and an `init()` that calls `news.Register(...)`.
3. Use `ingest.Fetch[T]()` for JSON/XML, `ingest.FetchHTML()` for scraping; finish with `ingest.TransformAll(ctx, items)`.
4. Add `internal/source/<name>_test.go` with `httptest.NewServer` stubs (success, non-2xx, edge cases).
5. `make test && make integration && go generate ./...`.
