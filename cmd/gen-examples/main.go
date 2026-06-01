// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// gen-examples fetches live data from every registered source and writes the
// results to examples/rendered and examples/raw.
//
// Pass -source to regenerate a single source. Pass -replay <file> together with
// -source to run a saved raw capture (e.g. a Wayback Machine snapshot of a
// feed) through the real Fetch/filter/transform pipeline instead of fetching
// live — useful for verifying how a source's listings would render on a past
// day when the upstream API exposes no date parameter.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	_ "github.com/ainsleyclark/godaily/pkg/source"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

func main() {
	only := flag.String("source", "", "regenerate only this source (e.g. golang_nuts)")
	replay := flag.String("replay", "", "replay a saved raw response file (e.g. a Wayback snapshot) through the pipeline instead of fetching live; requires -source")
	flag.Parse()

	if *replay != "" && *only == "" {
		log.Fatal("-replay requires -source to identify which source the file belongs to")
	}

	var replayBody []byte
	if *replay != "" {
		b, err := os.ReadFile(*replay)
		if err != nil {
			log.Fatalf("read replay file: %v", err)
		}
		replayBody = b
	}

	ctx := context.Background()

	cfg, err := env.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if err := news.Materialise(cfg); err != nil {
		log.Fatal(err)
	}

	renderedDir := filepath.Join("examples", "rendered")
	rawDir := filepath.Join("examples", "raw")
	for _, dir := range []string{renderedDir, rawDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			slog.Error("Create dir", "dir", dir, "err", err)
			os.Exit(1)
		}
	}

	for _, s := range news.Sources {
		if *only != "" && string(s) != *only {
			continue
		}
		fetcher, err := news.Get(s)
		if err != nil {
			slog.Warn("Skipping source", "source", s, "err", err)
			continue
		}

		// In replay mode the HTTP client serves the captured file for every
		// request, so the real parse/filter/transform runs against past-day
		// data; otherwise record the live response for the raw example.
		var rec *recordingTransport
		if *replay != "" {
			ct := replayContentType(*replay)
			ingest.SetHTTPClient(&http.Client{Transport: &staticTransport{body: replayBody, contentType: ct}})
		} else {
			rec = &recordingTransport{base: http.DefaultTransport}
			ingest.SetHTTPClient(&http.Client{Transport: rec})
		}

		items, err := fetcher.Fetch(ctx)
		if err != nil {
			slog.Error("Fetch failed", "source", s, "err", err)
			continue
		}

		if *replay != "" {
			slog.Info("Replayed", "source", s, "file", *replay, "items", len(items))
		}

		// Write raw API response. The extension is chosen from the recorded
		// Content-Type so HTML/XML sources don't masquerade as .json.
		if rec != nil && rec.body != nil {
			ext, body := rawExtAndBody(rec.contentType, rec.body)
			rawPath := filepath.Join(rawDir, string(s)+"."+ext)
			if err := os.WriteFile(rawPath, body, 0o600); err != nil {
				slog.Error("Write raw", "source", s, "err", err)
			}
		}

		// Write transformed items.
		data, err := json.MarshalIndent(items, "", "\t")
		if err != nil {
			slog.Error("Marshal", "source", s, "err", err)
			continue
		}
		renderedPath := filepath.Join(renderedDir, string(s)+".json")
		if err := os.WriteFile(renderedPath, data, 0o600); err != nil {
			slog.Error("Write rendered", "source", s, "err", err)
			continue
		}

		slog.Info("Wrote", "source", s, "items", len(items))
	}
}

// recordingTransport records the FIRST response body it sees together with
// its Content-Type. We keep only the first because per-source enrichment
// (ingest.enrich) issues follow-up HTTP calls through the same client, and
// overwriting on every round-trip would leave us with an enrichment page
// instead of the source's primary response.
type recordingTransport struct {
	base        http.RoundTripper
	body        []byte
	contentType string
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	if r.body == nil {
		r.body = body
		r.contentType = resp.Header.Get("Content-Type")
	}
	return resp, readErr
}

// staticTransport serves the same captured body for every request, used by
// -replay so a saved (e.g. archived) response runs through the real pipeline.
// Follow-up enrichment calls receive the same body, which is harmless for
// verifying source parsing and filtering.
type staticTransport struct {
	body        []byte
	contentType string
}

func (t *staticTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {t.contentType}},
		Body:       io.NopCloser(bytes.NewReader(t.body)),
	}, nil
}

// replayContentType infers a Content-Type from the replay file's extension so
// sources that branch on it (HTML vs XML vs JSON) parse the capture correctly.
func replayContentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return "application/json"
	case ".xml", ".rss", ".atom":
		return "application/xml"
	case ".html", ".htm":
		return "text/html"
	default:
		return "text/plain"
	}
}

// rawExtAndBody picks a file extension and (where useful) reformats the body
// based on the response Content-Type. JSON gets pretty-printed; HTML/XML are
// written verbatim under their native extension.
func rawExtAndBody(contentType string, body []byte) (string, []byte) {
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "json"):
		return "json", prettyJSON(body)
	case strings.Contains(ct, "html"):
		return "html", body
	case strings.Contains(ct, "xml"), strings.Contains(ct, "atom"), strings.Contains(ct, "rss"):
		return "xml", body
	default:
		return "txt", body
	}
}

// prettyJSON returns src pretty-printed with tab indentation if it is valid
// JSON, otherwise it returns src unchanged.
func prettyJSON(src []byte) []byte {
	var buf bytes.Buffer
	if err := json.Indent(&buf, src, "", "\t"); err != nil {
		return src
	}
	return buf.Bytes()
}
