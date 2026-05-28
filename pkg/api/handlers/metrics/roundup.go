// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"log/slog"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
)

// Roundup handles GET /metrics/roundup. It is scheduled via vercel.json
// to fire every Friday at 15:00 UTC. The handler gathers the last seven days of
// engagement data (with a week-over-week comparison) and posts a formatted
// summary to the configured Slack channel.
func (h *Handler) Roundup(c *webkit.Context) error {
	ctx := c.Context()
	if err := h.metricsService.Roundup(ctx); err != nil {
		slog.ErrorContext(ctx, "Weekly roundup failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to send weekly roundup")
	}
	return api.OK(c, http.StatusOK, nil, "Successfully sent weekly roundup")
}
