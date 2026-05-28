// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/hook"
)

// Rotation godoc
//
//	@Summary		Post a rotation social update.
//	@Description	Posts the next rotating social update across platforms. Skipped at weekends or when the social service is not wired.
//	@Tags			social
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	api.MessageResponse	"Rotated, or skipped (weekend/not wired)"
//	@Failure		500	{object}	api.MessageResponse	"Failed to rotate"
//	@Router			/social/rotation [get]
func (h *Handler) Rotation(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	if api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping rotation — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackSocialRotationHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped rotation — weekend")
	}

	if h.social == nil {
		slog.InfoContext(ctx, "Skipping rotation — social service not wired")
		hook.Heartbeat(ctx, h.config.BetterStackSocialRotationHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped rotation — service not wired")
	}

	results, err := h.social.Rotate(ctx, social.RotateOptions{Now: now})
	if err != nil {
		h.slack.MustSend(ctx, "Rotation post failed: "+err.Error())
		slog.ErrorContext(ctx, "Rotation post failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to rotate")
	}

	slog.InfoContext(ctx, "Rotation run complete", "platforms", len(results))
	hook.Heartbeat(ctx, h.config.BetterStackSocialRotationHeartbeatURL)
	return api.OK(c, http.StatusOK, nil, "Successfully rotated")
}
