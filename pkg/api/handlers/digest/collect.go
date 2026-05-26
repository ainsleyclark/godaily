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

package digest

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/gateway/hook"
	"github.com/ainsleydev/webkit/pkg/webkit"
	// Register all news-source fetchers (lingua-go + scrapers) so the
	// registry populates in this single binary.
	_ "github.com/ainsleyclark/godaily/pkg/source"
)

// Collect handles GET /digest/collect.
func (h *Handler) Collect(c *webkit.Context) error {
	ctx := c.Context()
	force := c.Request.URL.Query().Get("force") == "true"
	if !force && api.IsWeekend(time.Now().UTC()) {
		slog.InfoContext(ctx, "Skipping collect — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackCollectHeartbeatURL)
		return c.NoContent(http.StatusOK)
	}

	resp, err := h.runner.Collect(ctx, digest.CollectOptions{})
	if err != nil {
		h.slack.MustSend(ctx, "Collect failed: "+err.Error())
		return fmt.Errorf("collect failed: %w", err)
	}

	hook.Heartbeat(ctx, h.config.BetterStackCollectHeartbeatURL)

	type sourceResult struct {
		Count int     `json:"count"`
		Error *string `json:"error"`
	}
	sources := make(map[string]sourceResult, len(resp.Sources))
	for _, si := range resp.Sources {
		sources[string(si.Source)] = sourceResult{Count: len(si.Items)}
	}
	for src, srcErr := range resp.Errors {
		msg := srcErr.Error()
		sources[string(src)] = sourceResult{Error: &msg}
	}

	return c.JSON(http.StatusOK, map[string]any{"ok": true, "sources": sources})
}
