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

// metaDescriptionSelectors lists the meta tags we read, in priority order.
var metaDescriptionSelectors = []string{
	`meta[property="og:description"]`,
	`meta[name="twitter:description"]`,
	`meta[name="description"]`,
}

// EnrichSnippets fills in missing snippets by fetching each item's URL and
// extracting an HTML meta description.
//
// Items with a non-empty Snippet are left untouched. Discussion-page URLs
// (HN self-post permalinks, Reddit self-post threads) are skipped — they
// don't carry article-level meta tags. Per-item failures are logged at
// debug level and never propagate.
func EnrichSnippets(ctx context.Context, items []news.Item) {
	if len(items) == 0 {
		return
	}

	sem := make(chan struct{}, enrichConcurrency)
	var wg sync.WaitGroup

	for i := range items {
		if items[i].Snippet != "" {
			continue
		}
		if !shouldEnrich(items[i].URL) {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			fetchCtx, cancel := context.WithTimeout(ctx, enrichTimeout)
			defer cancel()

			desc, err := fetchMetaDescription(fetchCtx, items[idx].URL)
			if err != nil {
				slog.DebugContext(ctx, "snippet enrichment failed",
					"url", items[idx].URL, "err", err)
				return
			}
			items[idx].Snippet = truncate(sanitise(desc), maxSnippetLen)
		}(i)
	}
	wg.Wait()
}

// shouldEnrich reports whether url is worth fetching for snippet enrichment.
func shouldEnrich(url string) bool {
	if url == "" {
		return false
	}
	lower := strings.ToLower(url)
	if strings.Contains(lower, "news.ycombinator.com/item") {
		return false
	}
	if strings.Contains(lower, "reddit.com/r/") {
		return false
	}
	return true
}

// fetchMetaDescription returns the first non-empty description meta tag
// found in url's HTML head, preferring og:description, then
// twitter:description, then the standard description.
func fetchMetaDescription(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", errors.Wrap(err, "creating enrich request")
	}
	req.Header.Set("User-Agent", enrichUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "fetching url")
	}
	defer resp.Body.Close()

	if !httputil.Is2xx(resp.StatusCode) {
		return "", errors.Errorf("unexpected status %d", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		if !strings.Contains(strings.ToLower(ct), "html") {
			return "", errors.Errorf("non-html content-type %q", ct)
		}
	}

	doc, err := goquery.NewDocumentFromReader(io.LimitReader(resp.Body, enrichBodyMax))
	if err != nil {
		return "", errors.Wrap(err, "parsing html")
	}

	for _, sel := range metaDescriptionSelectors {
		if v, ok := doc.Find(sel).First().Attr("content"); ok {
			if v = strings.TrimSpace(v); v != "" {
				return v, nil
			}
		}
	}

	return "", nil
}
