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

package ingest

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleydev/webkit/pkg/util/httputil"
	"github.com/pkg/errors"
)

const (
	enrichConcurrency = 6
	enrichTimeout     = 5 * time.Second
	enrichBodyMax     = 64 * 1024
	enrichUserAgent   = "godaily/1.0"
)

// metaDescriptionSelectors lists the description meta tags we read, in priority order.
var metaDescriptionSelectors = []string{
	`meta[property="og:description"]`,
	`meta[name="twitter:description"]`,
	`meta[name="description"]`,
}

// metaImageSelectors lists the image meta tags we read, in priority order.
var metaImageSelectors = []string{
	`meta[property="og:image:secure_url"]`,
	`meta[property="og:image"]`,
	`meta[name="twitter:image"]`,
}

// enrichTarget pairs a URL to crawl with the item that should receive the
// extracted snippet/image. The URL is supplied by the source's
// Transformer.EnrichmentURL so it can differ from Item.URL when needed.
type enrichTarget struct {
	URL  string
	Item *news.Item
}

// enrich fills empty Snippet and ImageURL fields by fetching each target's
// URL once and extracting og:/twitter: meta tags. Items where both fields
// are already set incur no HTTP. Per-target failures are logged at debug
// level and never propagate.
func enrich(ctx context.Context, targets []enrichTarget) {
	if len(targets) == 0 {
		return
	}

	sem := make(chan struct{}, enrichConcurrency)
	var wg sync.WaitGroup

	for i := range targets {
		t := targets[i]
		if t.Item == nil || t.URL == "" {
			continue
		}
		if t.Item.Snippet != "" && t.Item.ImageURL != "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			fetchCtx, cancel := context.WithTimeout(ctx, enrichTimeout)
			defer cancel()

			doc, base, err := fetchPage(fetchCtx, t.URL)
			if err != nil {
				slog.DebugContext(ctx, "Enrichment failed", "url", t.URL, "err", err)
				return
			}
			if t.Item.Snippet == "" {
				if v := extractMeta(doc, metaDescriptionSelectors); v != "" {
					t.Item.Snippet = truncate(sanitise(v), maxSnippetLen)
				}
			}
			if t.Item.ImageURL == "" {
				if v := extractMeta(doc, metaImageSelectors); v != "" {
					if abs := resolveImageURL(base, v); abs != "" {
						t.Item.ImageURL = abs
					}
				}
			}
		}()
	}
	wg.Wait()
}

// fetchPage returns the parsed HTML document for url and the parsed URL
// (used as the base for resolving relative meta tag values).
func fetchPage(ctx context.Context, target string) (*goquery.Document, *url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating enrich request")
	}
	req.Header.Set("User-Agent", enrichUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "fetching url")
	}
	defer resp.Body.Close()

	if !httputil.Is2xx(resp.StatusCode) {
		return nil, nil, errors.Errorf("unexpected status %d", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		if !strings.Contains(strings.ToLower(ct), "html") {
			return nil, nil, errors.Errorf("non-html content-type %q", ct)
		}
	}

	doc, err := goquery.NewDocumentFromReader(io.LimitReader(resp.Body, enrichBodyMax))
	if err != nil {
		return nil, nil, errors.Wrap(err, "parsing html")
	}

	base, err := url.Parse(target)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parsing base url")
	}
	return doc, base, nil
}

// extractMeta returns the first non-empty content attribute matching any of
// the given selectors, in order.
func extractMeta(doc *goquery.Document, selectors []string) string {
	for _, sel := range selectors {
		if v, ok := doc.Find(sel).First().Attr("content"); ok {
			if v = strings.TrimSpace(v); v != "" {
				return v
			}
		}
	}
	return ""
}

// resolveImageURL turns a meta tag value into an absolute http(s) URL.
// Relative paths are resolved against base; non-http(s) schemes (e.g.
// data:) and unparseable values return "".
func resolveImageURL(base *url.URL, raw string) string {
	ref, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	abs := base.ResolveReference(ref)
	if abs.Scheme != "http" && abs.Scheme != "https" {
		return ""
	}
	return abs.String()
}
