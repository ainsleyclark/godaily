# Adding a New Source

Follow these steps to add a new news source. Use `hackernews.go` and `hackernews_test.go` as the
reference implementation.

---

## 1. Register the source constant

In `internal/news/sources.go`, add a constant and append it to the `Sources` slice:

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

Create `internal/source/foo.go`:

```go
package source

import (
    "context"
    "encoding/json" // or encoding/xml

    "github.com/ainsleyclark/godaily/internal/news"
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
    response, err := fetch[fooResponse](ctx, f.url, "foo", json.Unmarshal)
    if err != nil {
        return nil, err
    }
    return transformAll(response.Items), nil
}

func (i fooItem) transform() news.Item {
    return news.Item{
        Source:    news.SourceFoo,
        Title:     i.Title,
        URL:       i.Link,
        Author:    i.Author,
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
        PublishedAt time.Time `json:"published_at"`
    }
)
```

Key points:
- `fetch[T]()` handles the HTTP GET, status check, and unmarshal — pass `json.Unmarshal` or `xml.Unmarshal`
- `transformAll()` calls `.transform()` on each response item
- The `var _ news.Fetcher = &Foo{}` line enforces the interface at compile time
- `init()` registers the factory; without it `TestValidate` will fail

## 3. Write tests

Create `internal/source/foo_test.go`. Use `httptest.NewServer` to stub the API and swap `httpClient` via the test server's client:

```go
package source

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/ainsleyclark/godaily/internal/news"
    "github.com/stretchr/testify/assert"
)

func TestFoo_Fetch(t *testing.T) {
    t.Parallel()

    tt := map[string]struct {
        stub http.HandlerFunc
        want func([]news.Item, error)
    }{
        "Bad Request": {
            stub: func(w http.ResponseWriter, _ *http.Request) {
                w.WriteHeader(http.StatusBadRequest)
            },
            want: func(items []news.Item, err error) {
                assert.Error(t, err)
                assert.Nil(t, items)
            },
        },
        "OK": {
            stub: func(w http.ResponseWriter, _ *http.Request) {
                w.WriteHeader(http.StatusOK)
                _, err := w.Write([]byte(`{"items":[{"title":"Hello","link":"https://example.com"}]}`))
                assert.NoError(t, err)
            },
            want: func(items []news.Item, err error) {
                assert.NoError(t, err)
                assert.Len(t, items, 1)
                assert.Equal(t, "Hello", items[0].Title)
            },
        },
    }

    for name, test := range tt {
        t.Run(name, func(t *testing.T) {
            s := httptest.NewServer(test.stub)
            defer s.Close()
            got, err := Foo{url: s.URL}.Fetch(t.Context())
            test.want(got, err)
        })
    }
}
```

Cover at minimum: successful response, non-2xx error, and any source-specific edge cases (e.g. missing fields, fallback URLs).

## 4. Verify

```sh
make test        # unit tests
make integration # hits real APIs — ensure the new source returns items
go generate ./... # regenerates any static fixtures or generated files
```

`TestValidate` in `internal/news/registry_test.go` will fail if you forgot the `init()` registration.
