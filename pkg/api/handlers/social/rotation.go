// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/hook"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Rotation handles GET /social/rotation.
func (h *Handler) Rotation(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	if api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping rotation — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackSocialRotationHeartbeatURL)
		return c.NoContent(http.StatusOK)
	}

	if h.social == nil {
		slog.InfoContext(ctx, "Skipping rotation — social service not wired")
		hook.Heartbeat(ctx, h.config.BetterStackSocialRotationHeartbeatURL)
		return c.NoContent(http.StatusOK)
	}

	results, err := h.social.Rotate(ctx, social.RotateOptions{Now: now})
	if err != nil {
		h.slack.MustSend(ctx, "Rotation post failed: "+err.Error())
		return webkit.NewError(http.StatusInternalServerError, "rotation post failed: "+err.Error())
	}

	slog.InfoContext(ctx, "Rotation run complete", "platforms", len(results))
	hook.Heartbeat(ctx, h.config.BetterStackSocialRotationHeartbeatURL)
	return c.NoContent(http.StatusOK)
}
