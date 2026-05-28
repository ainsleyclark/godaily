// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/gateway/hook"
)

// Build handles GET /digest/build.
func (h *Handler) Build(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	force := c.Request.URL.Query().Get("force") == "true"
	if !force && api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping build — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackBuildHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped build — weekend")
	}

	if err := h.runner.Build(ctx, now); err != nil {
		slog.ErrorContext(ctx, "Build failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to build digest")
	}

	hook.Heartbeat(ctx, h.config.BetterStackBuildHeartbeatURL)

	return api.OK(c, http.StatusOK, nil, "Successfully built digest")
}
