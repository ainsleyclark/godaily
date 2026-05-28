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

// Preview godoc
//
//	@Summary		Send the digest preview to the owner.
//	@Description	Sends the draft digest and AI synth suggestion to the owner ahead of the full subscriber send. Skipped at weekends.
//	@Tags			digest
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	api.MessageResponse	"Successfully sent preview"
//	@Failure		500	{object}	api.MessageResponse	"Failed to send preview"
//	@Router			/digest/preview [get]
func (h *Handler) Preview(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	if api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping preview — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackSendHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped preview — weekend")
	}

	today := now.Truncate(24 * time.Hour)

	if err := h.runner.SendPreview(ctx, today); err != nil {
		h.slack.MustSend(ctx, "Send preview failed: "+err.Error())
		slog.ErrorContext(ctx, "Send preview failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to send preview")
	}

	hook.Heartbeat(ctx, h.config.BetterStackSendHeartbeatURL)

	return api.OK(c, http.StatusOK, nil, "Successfully sent preview")
}
