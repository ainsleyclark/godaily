// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
	if err := h.metricsService.Roundup(c.Context()); err != nil {
		slog.ErrorContext(c.Context(), "Weekly roundup failed", "err", err)
		return webkit.NewError(http.StatusInternalServerError, "failed to send weekly roundup")
	}
	return c.NoContent(http.StatusOK)
}
