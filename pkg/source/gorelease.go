// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package source

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
	"github.com/ainsleyclark/godaily/pkg/util/gohttp"
)

// GoRelease defines the type that implements news.Fetcher for go.dev/dl/.
type GoRelease struct {
	url        string
	dlBase     string
	limit      int
	httpClient *http.Client
	dateFor    func(ctx context.Context, fileURL string) time.Time
}

var _ news.Fetcher = &GoRelease{}

func init() {
	news.Register(news.SourceGoRelease, func(cfg env.Config) news.Fetcher { return NewGoRelease(cfg) })
}

const (
	goReleaseURL    = "https://go.dev/dl/?mode=json&include=all"
	goReleaseDLBase = "https://go.dev/dl/"
	goReleaseLimit  = 3
)

// NewGoRelease creates a Go release downloads client. The endpoint returns
// every historical release; we cap at goReleaseLimit so the digest never
// drowns in years of past versions.
func NewGoRelease(_ env.Config) *GoRelease {
	g := &GoRelease{
		url:        goReleaseURL,
		dlBase:     goReleaseDLBase,
		limit:      goReleaseLimit,
		httpClient: gohttp.New(),
	}
	g.dateFor = func(ctx context.Context, u string) time.Time {
		return lastModified(ctx, u, g.httpClient)
	}
	return g
}

// Fetch retrieves the Go release index and returns the most recent stable
// releases as news items. The endpoint payload carries no publish date, so
// we issue a HEAD against the source tarball and read Last-Modified — this
// keeps the daily digest's 24h freshness filter useful.
func (g GoRelease) Fetch(ctx context.Context) ([]news.Item, error) {
	releases, err := ingest.Fetch[[]goRelease](ctx, g.url, "go release", json.Unmarshal)
	if err != nil {
		return nil, err
	}
	if g.limit > 0 && len(releases) > g.limit {
		releases = releases[:g.limit]
	}
	for i := range releases {
		if file := releases[i].sourceFile(); file != "" && g.dateFor != nil {
			releases[i].published = g.dateFor(ctx, g.dlBase+file)
		}
	}
	return ingest.TransformAll(ctx, releases), nil
}

func (r goRelease) ShouldInclude() bool   { return r.Stable }
func (r goRelease) EnrichmentURL() string { return "" }

// Transform maps a goRelease to a news.Item. The payload carries no release
// notes, so the snippet is fixed; Published is populated by Fetch via a
// HEAD request against the source tarball.
func (r goRelease) Transform() news.Item {
	version := strings.TrimPrefix(r.Version, "go")
	return news.Item{
		Source:    news.SourceGoRelease,
		Title:     "Go " + version + " released",
		URL:       "https://go.dev/doc/devel/release#" + r.Version,
		Snippet:   "Stable Go release. See release notes for changes.",
		Tag:       news.TagRelease,
		Score:     news.ScoreOf(news.SourceGoRelease, news.TagRelease, 0, false),
		Published: r.published,
	}
}

// sourceFile returns the filename of the .src.tar.gz file in this release,
// or empty if the release has no source download (shouldn't happen in
// practice, but keeps the HEAD lookup defensive).
func (r goRelease) sourceFile() string {
	for _, f := range r.Files {
		if f.Kind == "source" {
			return f.Filename
		}
	}
	return ""
}

// lastModified issues a HEAD against url and parses the Last-Modified
// header. Returns zero time on any error — the cron pipeline drops zero-date
// items, which is the right fallback when the lookup fails.
func lastModified(ctx context.Context, url string, c *http.Client) time.Time {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return time.Time{}
	}
	req.Header.Set("User-Agent", "godaily/1.0")
	resp, err := c.Do(req)
	if err != nil {
		return time.Time{}
	}
	defer resp.Body.Close()
	t, err := time.Parse(time.RFC1123, resp.Header.Get("Last-Modified"))
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

type (
	goRelease struct {
		Version string          `json:"version"`
		Stable  bool            `json:"stable"`
		Files   []goReleaseFile `json:"files"`

		published time.Time `json:"-"`
	}
	goReleaseFile struct {
		Filename string `json:"filename"`
		Kind     string `json:"kind"`
	}
)
