# Adding a New Source

Follow these steps to add a new news source. Use `hackernews.go` and `hackernews_test.go` as the
reference implementation. Shared plumbing — HTTP fetch, transformation, snippet enrichment — lives
in `internal/ingest`; this package only holds per-provider fetchers.

---

## 1. Register the source constant

In `pkg/domain/news/sources.go`, add a constant and append it to the `Sources` slice:

```go
const (
    // ...
    SourceFoo Source = "foo"
)

var Sources = []Source{
    // ...
    SourceFoo,
}
```

## 2. Create the implementation

Create `pkg/source/foo.go`:

```go
package source

import (
    "context"
    "encoding/json" // or encoding/xml

    "github.com/ainsleyclark/godaily/pkg/domain/news"
    "github.com/ainsleyclark/godaily/pkg/source/ingest"
)

type Foo struct {
    url string
}

var _ news.Fetcher = &Foo{}

func init() {
    news.Register(news.SourceFoo, NewFoo())
}

const fooURL = "https://api.example.com/feed"

func NewFoo() *Foo {
    return &Foo{url: fooURL}
}

func (f Foo) Fetch(ctx context.Context) ([]news.Item, error) {
    response, err := ingest.Fetch[fooResponse](ctx, f.url, "foo", json.Unmarshal)
    if err != nil {
        return nil, err
    }
    return ingest.TransformAll(ctx, response.Items), nil
}

func (i fooItem) ShouldInclude() bool { return true }

func (i fooItem) Transform() news.Item {
    return news.Item{
        Source:    news.SourceFoo,
        Title:     i.Title,
        URL:       i.Link,
        Author:    i.Author,
        Snippet:   i.Description, // raw — ingest.TransformAll sanitises and truncates
        Published: i.PublishedAt,
        Tag:       news.TagArticle,
    }
}

type (
    fooResponse struct {
        Items []fooItem `json:"items"`
    }
    fooItem struct {
        Title       string    `json:"title"`
        Link        string    `json:"link"`
        Author      string    `json:"author"`
        Description string    `json:"description"`
        PublishedAt time.Time `json:"published_at"`
    }
)
```

Key points:
- `ingest.Fetch[T]()` handles the HTTP GET, status check, and unmarshal — pass `json.Unmarshal` or `xml.Unmarshal`.
- `ingest.TransformAll()` calls `.Transform()` on each response item, then sanitises the resulting
  snippet (strips HTML, unescapes entities, collapses whitespace) and truncates it. Sources put raw
  API content into `news.Item.Snippet` without per-source cleanup.
- Items whose `ShouldInclude()` returns `false` are silently dropped.
- The `var _ news.Fetcher = &Foo{}` line enforces the interface at compile time.
- `init()` registers the factory; without it `TestValidate` will fail.
- For empty snippets (e.g. external link posts), the cron pipeline calls `ingest.EnrichSnippets`
  after fetch, which fills them in from the article's meta description.

## 3. Add a mark/logo asset

Add a square SVG (or PNG/WebP for raster brands) to `web/assets/images/marks/<source_id>.<ext>` and
register its path in the `sourceMarkURLs` map in `pkg/domain/news/sources.go`:

```go
var sourceMarkURLs = map[Source]string{
    // ...
    SourceFoo: "/assets/images/marks/foo.svg",
}
```

Sources without a `MarkURL` entry fall back to their `ShortLabel` chip — that is acceptable for
sources whose brand assets cannot be freely redistributed, but a mark file is strongly preferred.

## 4. Capture a real-API fixture

Tests load the OK-case payload from `testdata/<source>.{json,xml,atom,html}` rather than embedding
the response inline. Capture a small real response from the upstream once and commit it:

```sh
curl -s 'https://api.example.com/feed?limit=3' -o pkg/source/testdata/foo.json
```

Trim to ~2–3 representative items so the file stays readable but still exercises multi-item parsing.
**If the source enriches** (i.e. its `EnrichmentURL()` returns a non-empty URL), replace every
external URL in the captured response with the literal sentinel `__SERVER_URL__`. The test rewrites
this to the local `httptest` server's URL at runtime, so enrichment requests never hit the live
internet. Sources that return `""` from `EnrichmentURL()` (e.g. dev.to, GolangBridge, YouTube) need
no substitution — leave the captured URLs verbatim.

Edge-case payloads (malformed cards, missing fields, self-posts) stay inline as small `const`
strings — those are crafted negative tests, not real API samples.

## 5. Write tests

Create `pkg/source/foo_test.go`. Use `httptest.NewServer` to stub the API; load the fixture
once at the top of the test and (for enriching sources) `strings.ReplaceAll` the sentinel:

```go
package source

import (
    "net/http"
    "net/http/httptest"
    "os"
    "strings"
    "testing"

    "github.com/ainsleyclark/godaily/pkg/domain/news"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFoo_Fetch(t *testing.T) {
    t.Parallel()

    fixture, err := os.ReadFile("testdata/foo.json")
    require.NoError(t, err)

    tt := map[string]struct {
        stub func(serverURL string) http.HandlerFunc
        want func(t *testing.T, items []news.Item, err error, serverURL string)
    }{
        "Bad Request": {
            stub: func(string) http.HandlerFunc {
                return func(w http.ResponseWriter, _ *http.Request) {
                    w.WriteHeader(http.StatusBadRequest)
                }
            },
            want: func(t *testing.T, items []news.Item, err error, _ string) {
                t.Helper()
                assert.Error(t, err)
                assert.Nil(t, items)
            },
        },
        "OK": {
            stub: func(serverURL string) http.HandlerFunc {
                // Drop the ReplaceAll if Foo doesn't enrich.
                body := strings.ReplaceAll(string(fixture), "__SERVER_URL__", serverURL)
                return func(w http.ResponseWriter, _ *http.Request) {
                    w.WriteHeader(http.StatusOK)
                    _, _ = w.Write([]byte(body))
                }
            },
            want: func(t *testing.T, items []news.Item, err error, serverURL string) {
                t.Helper()
                assert.NoError(t, err)
                assert.Len(t, items, 3)
                assert.Equal(t, "Hello", items[0].Title)
            },
        },
    }

    for name, test := range tt {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            var serverURL string
            s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                test.stub(serverURL)(w, r)
            }))
            defer s.Close()
            serverURL = s.URL
            got, err := Foo{url: s.URL}.Fetch(t.Context())
            test.want(t, got, err, s.URL)
        })
    }
}
```

Cover at minimum: successful response (from the fixture), non-2xx error, and any source-specific
edge cases (missing fields, fallback URLs) using small inline `const` strings.

### Refreshing fixtures

When the upstream API schema changes and the fixture-based test fails, re-run the original `curl`
to capture a fresh response, re-apply the `__SERVER_URL__` substitution if the source enriches, and
update the OK-case assertions to match the new first-item values.

## 6. Verify

```sh
make test        # unit tests
make integration # hits real APIs — ensure the new source returns items
go generate ./... # regenerates any static fixtures or generated files
```

`TestValidate` in `internal/news/registry_test.go` will fail if you forgot the `init()` registration.
