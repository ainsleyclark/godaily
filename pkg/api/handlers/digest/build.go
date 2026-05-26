// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/gateway/hook"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Build handles GET /digest/build.
func (h *Handler) Build(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	force := c.Request.URL.Query().Get("force") == "true"
	if !force && api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping build — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackBuildHeartbeatURL)
		return c.NoContent(http.StatusOK)
	}

	if err := h.runner.Build(ctx, now); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	hook.Heartbeat(ctx, h.config.BetterStackBuildHeartbeatURL)

	return c.NoContent(http.StatusOK)
}
