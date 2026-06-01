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

// PublishFeatured godoc
//
//	@Summary		Publish featured social drafts.
//	@Description	Publishes the day's featured draft rows (one per configured platform) — recap and other rotation kinds are left untouched so the 15:00 rotation slot picks them up. Skipped at weekends.
//	@Tags			social
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	api.MessageResponse	"Published, or skipped (weekend/not wired)"
//	@Failure		500	{object}	api.MessageResponse	"Failed to publish featured drafts"
//	@Router			/social/publish/featured [get]
func (h *Handler) PublishFeatured(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	if api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping featured publish — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackSocialFeaturedHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped featured publish — weekend")
	}

	if h.social == nil {
		slog.InfoContext(ctx, "Skipping featured publish — social service not wired")
		hook.Heartbeat(ctx, h.config.BetterStackSocialFeaturedHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped featured publish — service not wired")
	}

	results, err := h.social.PublishDrafts(ctx, social.PostOptions{
		Date:  now.Truncate(24 * time.Hour),
		Kinds: []social.PostKind{social.PostKindFeatured},
	})
	if err != nil {
		h.slack.MustSend(ctx, slack.Error("Featured publish failed", err))
		slog.ErrorContext(ctx, "Featured publish failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to publish featured drafts")
	}

	slog.InfoContext(ctx, "Featured publish run complete", "posts", len(results))
	hook.Heartbeat(ctx, h.config.BetterStackSocialFeaturedHeartbeatURL)
	return api.OK(c, http.StatusOK, nil, "Successfully published featured drafts")
}
