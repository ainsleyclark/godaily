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
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

// Send godoc
//
//	@Summary		Send the digest to subscribers.
//	@Description	Sends today's digest by email to all active subscribers. Skipped at weekends unless force=true.
//	@Tags			digest
//	@Produce		json
//	@Security		BearerAuth
//	@Param			force	query		bool			false	"Force send even at weekends"
//	@Success		200		{object}	api.MessageResponse	"Successfully sent digest"
//	@Failure		500		{object}	api.MessageResponse	"Failed to send digest"
//	@Router			/digest/send [get]
func (h *Handler) Send(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	force := c.Request.URL.Query().Get("force") == "true"
	if !force && api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping send — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackSendHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped send — weekend")
	}

	today := now.Truncate(24 * time.Hour)

	if err := h.runner.SendDigest(ctx, today, force); err != nil {
		h.slack.MustSend(ctx, slack.Error("Send digest failed", err))
		slog.ErrorContext(ctx, "Send digest failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to send digest")
	}

	hook.Deploy(ctx, h.config.VercelDeployHookURL)
	hook.Heartbeat(ctx, h.config.BetterStackSendHeartbeatURL)

	return api.OK(c, http.StatusOK, nil, "Successfully sent digest")
}
