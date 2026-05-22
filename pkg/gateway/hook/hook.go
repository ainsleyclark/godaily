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

package hook

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

// Heartbeat pings a BetterStack (or compatible) heartbeat URL to signal that a
// job completed successfully. It is a no-op when url is empty.
func Heartbeat(ctx context.Context, url string) {
	fire(ctx, http.MethodGet, url, "heartbeat")
}

// Deploy triggers a Vercel deploy hook via POST. It is a no-op when url is empty.
func Deploy(ctx context.Context, url string) {
	fire(ctx, http.MethodPost, url, "deploy hook")
}

func fire(ctx context.Context, method, url, label string) {
	if url == "" {
		slog.DebugContext(ctx, "Skipping "+label+" — URL not configured")
		return
	}
	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(""))
	if err != nil {
		slog.ErrorContext(ctx, "Creating "+label+" request", "error", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "Firing "+label, "error", err)
		return
	}
	if err = resp.Body.Close(); err != nil {
		slog.ErrorContext(ctx, "Closing "+label+" response body", "error", err)
	}
}
