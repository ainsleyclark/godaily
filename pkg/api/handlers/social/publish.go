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
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

// Publish godoc
//
//	@Summary		Publish every pending social draft.
//	@Description	Walks every row with status='draft' (any kind, any platform) and posts it to its platform. Cancelled rows are skipped. Idempotent via row status transitions. Skipped at weekends.
//	@Tags			social
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	api.MessageResponse	"Published, or skipped (weekend/not wired)"
//	@Failure		500	{object}	api.MessageResponse	"Failed to publish drafts"
//	@Router			/social/publish [get]
func (h *Handler) Publish(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	if api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping publish — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackSocialFeaturedHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped publish — weekend")
	}

	today := now.Truncate(24 * time.Hour)

	if h.social == nil {
		slog.InfoContext(ctx, "Skipping publish — social service not wired")
		hook.Heartbeat(ctx, h.config.BetterStackSocialFeaturedHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped publish — service not wired")
	}

	results, err := h.social.PublishDrafts(ctx, social.PostOptions{Date: today})
	if err != nil {
		h.slack.MustSend(ctx, slack.Error("Publish drafts failed", err))
		slog.ErrorContext(ctx, "Publish drafts failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to publish drafts")
	}

	slog.InfoContext(ctx, "Publish run complete", "posts", len(results))
	hook.Heartbeat(ctx, h.config.BetterStackSocialFeaturedHeartbeatURL)

	return api.OK(c, http.StatusOK, nil, "Successfully published social drafts")
}
