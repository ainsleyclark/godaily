// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

func (r goRelease) ShouldInclude() bool   { return true }
func (r goRelease) EnrichmentURL() string { return "" }

// Transform maps a goRelease to a news.Item. Stable releases read
// "Go 1.26.2 released"; pre-releases surface their candidate/beta label so the
// heading reads "Go 1.27 RC1 released" rather than the raw "Go 1.27rc1". The
// payload carries no release notes, so the snippet is fixed per release kind;
// Published is populated by Fetch via a HEAD request against the source tarball.
func (r goRelease) Transform() news.Item {
	title, snippet := r.titleAndSnippet()
	return news.Item{
		Source:    news.SourceGoRelease,
		Title:     title,
		URL:       "https://go.dev/doc/devel/release#" + r.Version,
		Snippet:   snippet,
		Tag:       news.TagRelease,
		Score:     news.ScoreOf(news.SourceGoRelease, news.TagRelease, 0, false),
		Published: r.published,
	}
}

// titleAndSnippet renders the digest title and snippet for the release. Stable
// releases keep the plain "Go <version> released" form; pre-releases split the
// version into its base and an rc/beta label so the heading is readable.
func (r goRelease) titleAndSnippet() (title, snippet string) {
	version := strings.TrimPrefix(r.Version, "go")
	if r.Stable {
		return "Go " + version + " released", "Stable Go release. See release notes for changes."
	}
	if base, label := splitPreRelease(version); label != "" {
		version = base + " " + label
	}
	return "Go " + version + " released", "Go pre-release — try it in dev and prod, and file bugs."
}

// splitPreRelease separates a pre-release version such as "1.27rc1" into its
// base ("1.27") and a display label ("RC1"). Returns an empty label when the
// version carries no recognised rc/beta marker.
func splitPreRelease(version string) (base, label string) {
	for _, kind := range []struct{ token, display string }{
		{"rc", "RC"},
		{"beta", "Beta"},
	} {
		if i := strings.Index(version, kind.token); i != -1 {
			return version[:i], kind.display + version[i+len(kind.token):]
		}
	}
	return version, ""
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
