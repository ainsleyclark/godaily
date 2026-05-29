// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ingest holds the cross-cutting plumbing shared by every
// per-provider source: HTTP fetch, response transformation, and snippet
// enrichment. Source packages compose these primitives; ingest itself
// has no knowledge of any specific provider.
package ingest

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ainsleyclark/godaily/pkg/util/gohttp"
	"github.com/ainsleydev/webkit/pkg/util/httputil"
	"github.com/pkg/errors"
)

// htmlBodyMax caps the response size FetchHTML reads from a page. Most pages
// (e.g. github.com/trending/go) sit well under this; the limit prevents a
// pathological response from eating memory without rejecting realistic ones.
const htmlBodyMax = 2 * 1024 * 1024

// httpClient is the shared HTTP client used by Fetch and EnrichSnippets.
// The timeout is raised to 2 minutes because ScraperAPI proxying (e.g. Reddit)
// can take ~45s+ per response, well past gohttp's 30s default.
var httpClient = gohttp.New(gohttp.WithTimeout(2 * time.Minute))

// SetHTTPClient replaces the shared HTTP client used by all sources.
func SetHTTPClient(c *http.Client) {
	httpClient = c
}

// Fetch performs a GET request to url, checks for a 2xx status, reads the body
// into bytes, then calls unmarshal to decode it into T.
//
// Callers pass json.Unmarshal or xml.Unmarshal — both match the required
// func([]byte, any) error signature. Optional headers are merged onto the
// request; existing callers that pass none continue to work unchanged.
func Fetch[T any](
	ctx context.Context,
	url string,
	name string,
	unmarshal func([]byte, any) error,
	headers ...http.Header,
) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return zero, errors.Wrap(err, name+" request creation failed")
	}

	for _, h := range headers {
		for k, vs := range h {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return zero, errors.Wrap(err, "fetch "+name)
	}
	defer resp.Body.Close()

	if !httputil.Is2xx(resp.StatusCode) {
		return zero, errors.Errorf("unexpected status code from %s: %d", name, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, errors.Wrap(err, "reading response")
	}

	var out T
	if err = unmarshal(data, &out); err != nil {
		return zero, errors.Wrap(err, "parsing response")
	}

	return out, nil
}

// FetchHTML performs a GET, verifies a 2xx response, and returns the parsed
// goquery document. Sources use this when they need to scrape HTML directly
// (e.g. github.com/trending/go has no JSON API). The response body is capped
// at htmlBodyMax bytes.
func FetchHTML(ctx context.Context, url, name string, headers ...http.Header) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, name+" request creation failed")
	}

	req.Header.Set("User-Agent", "godaily/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	for _, h := range headers {
		for k, vs := range h {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "fetch "+name)
	}
	defer resp.Body.Close()

	if !httputil.Is2xx(resp.StatusCode) {
		return nil, errors.Errorf("unexpected status code from %s: %d", name, resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(io.LimitReader(resp.Body, htmlBodyMax))
	if err != nil {
		return nil, errors.Wrap(err, "parsing html")
	}
	return doc, nil
}
