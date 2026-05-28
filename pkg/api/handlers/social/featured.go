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

// Featured godoc
//
//	@Summary		Post the featured social update.
//	@Description	Posts today's featured social update across platforms. Skipped at weekends and outside the chosen 10-minute slot; idempotent via the social_posts table.
//	@Tags			social
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	api.Response	"Posted, or skipped (weekend/wrong slot/not wired)"
//	@Failure		500	{object}	api.Response	"Failed to post featured"
//	@Router			/social/featured [get]
func (h *Handler) Featured(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	if api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping featured — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackSocialFeaturedHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped featured — weekend")
	}

	today := now.Truncate(24 * time.Hour)

	if !social.ShouldRun(now, today) {
		slog.InfoContext(
			ctx, "Skipping featured — wrong slot",
			"minute", now.Minute(), "picked", social.PickSlot(today),
		)
		return api.OK(c, http.StatusOK, nil, "Skipped featured — wrong slot")
	}

	if h.social == nil {
		slog.InfoContext(ctx, "Skipping featured — social service not wired")
		hook.Heartbeat(ctx, h.config.BetterStackSocialFeaturedHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped featured — service not wired")
	}

	results, err := h.social.Post(ctx, social.PostOptions{Date: today})
	if err != nil {
		h.slack.MustSend(ctx, "Featured post failed: "+err.Error())
		slog.ErrorContext(ctx, "Featured post failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to post featured")
	}

	slog.InfoContext(ctx, "Featured run complete", "platforms", len(results))
	hook.Heartbeat(ctx, h.config.BetterStackSocialFeaturedHeartbeatURL)

	return api.OK(c, http.StatusOK, nil, "Successfully posted featured")
}
