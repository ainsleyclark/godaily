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

// gen-examples fetches live data from every registered source and writes the
// results to internal/examples/rendered and internal/examples/raw. Run via:
//
//	go generate ./...
//
//go:generate go run main.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/source"
	_ "github.com/ainsleyclark/godaily/internal/source"
)

func main() {
	renderedDir := filepath.Join("..", "..", "internal", "examples", "rendered")
	rawDir := filepath.Join("..", "..", "internal", "examples", "raw")
	for _, dir := range []string{renderedDir, rawDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			slog.Error("create dir", "dir", dir, "err", err)
			os.Exit(1)
		}
	}

	ctx := context.Background()

	for _, s := range news.Sources {
		fetcher, err := news.Get(s)
		if err != nil {
			slog.Warn("skipping source", "source", s, "err", err)
			continue
		}

		rec := &recordingTransport{base: http.DefaultTransport}
		source.SetHTTPClient(&http.Client{Transport: rec})

		items, err := fetcher.Fetch(ctx)
		if err != nil {
			slog.Error("fetch failed", "source", s, "err", err)
			continue
		}

		// Write raw API response, pretty-printed if JSON.
		if rec.body != nil {
			rawPath := filepath.Join(rawDir, string(s)+".json")
			if err := os.WriteFile(rawPath, prettyJSON(rec.body), os.ModePerm); err != nil {
				slog.Error("write raw", "source", s, "err", err)
			}
		}

		// Write transformed items.
		data, err := json.MarshalIndent(items, "", "\t")
		if err != nil {
			slog.Error("marshal", "source", s, "err", err)
			continue
		}
		renderedPath := filepath.Join(renderedDir, string(s)+".json")
		if err := os.WriteFile(renderedPath, data, os.ModePerm); err != nil {
			slog.Error("write rendered", "source", s, "err", err)
			continue
		}

		slog.Info("wrote", "source", s, "items", len(items))
	}
}

// recordingTransport records the raw response body while passing the response
// through to the caller unchanged.
type recordingTransport struct {
	base http.RoundTripper
	body []byte
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	r.body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(r.body))
	return resp, err
}

// prettyJSON returns src pretty-printed with tab indentation if it is valid
// JSON, otherwise it returns src unchanged (e.g. for XML responses).
func prettyJSON(src []byte) []byte {
	var buf bytes.Buffer
	if err := json.Indent(&buf, src, "", "\t"); err != nil {
		return src
	}
	return buf.Bytes()
}

