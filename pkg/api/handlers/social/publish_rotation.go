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

// rotationKinds are the post kinds the rotation publish cron is allowed
// to promote. Featured is deliberately excluded so the 15:00 slot never
// catches a featured draft the 11:00 slot missed — those wait for
// tomorrow's featured slot or manual intervention.
var rotationKinds = []social.PostKind{
	social.PostKindRecap,
	social.PostKindCommunity,
	social.PostKindNewSource,
	social.PostKindSpotlight,
	social.PostKindCTA,
}

// PublishRotation godoc
//
//	@Summary		Publish rotation social drafts.
//	@Description	Publishes the day's rotation drafts (recap on Monday, community on Wednesday, new_source/spotlight/cta on Friday). Featured drafts are deliberately excluded — they belong to the 11:00 cron. Skipped at weekends.
//	@Tags			social
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	api.MessageResponse	"Published, or skipped (weekend/not wired)"
//	@Failure		500	{object}	api.MessageResponse	"Failed to publish rotation drafts"
//	@Router			/social/publish/rotation [get]
func (h *Handler) PublishRotation(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	if api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping rotation publish — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackSocialRotationHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped rotation publish — weekend")
	}

	if h.social == nil {
		slog.InfoContext(ctx, "Skipping rotation publish — social service not wired")
		hook.Heartbeat(ctx, h.config.BetterStackSocialRotationHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped rotation publish — service not wired")
	}

	results, err := h.social.PublishDrafts(ctx, social.PostOptions{
		Date:  now.Truncate(24 * time.Hour),
		Kinds: rotationKinds,
	})
	if err != nil {
		h.slack.MustSend(ctx, slack.Error("Rotation publish failed", err))
		slog.ErrorContext(ctx, "Rotation publish failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to publish rotation drafts")
	}

	slog.InfoContext(ctx, "Rotation publish run complete", "posts", len(results))
	hook.Heartbeat(ctx, h.config.BetterStackSocialRotationHeartbeatURL)
	return api.OK(c, http.StatusOK, nil, "Successfully published rotation drafts")
}
