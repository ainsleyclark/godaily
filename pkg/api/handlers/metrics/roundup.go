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

package metrics

import (
	"log/slog"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Roundup handles GET /metrics/roundup. It is scheduled via vercel.json
// to fire every Friday at 15:00 UTC. The handler gathers the last seven days of
// engagement data (with a week-over-week comparison) and posts a formatted
// summary to the configured Slack channel.
func (h *Handler) Roundup(c *webkit.Context) error {
	if err := h.metricsReporter.Roundup(c.Context()); err != nil {
		slog.ErrorContext(c.Context(), "Weekly roundup failed", "err", err)
		return webkit.NewError(http.StatusInternalServerError, "failed to send weekly roundup")
	}
	return c.NoContent(http.StatusOK)
}
