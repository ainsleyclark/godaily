// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"log/slog"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
)

// NudgeResult reports how many confirmation reminders were sent and failed.
type NudgeResult struct {
	Sent   int `json:"sent"`
	Failed int `json:"failed"`
}

// Nudge godoc
//
//	@Summary		Send confirmation nudges.
//	@Description	Sends a one-time reminder to subscribers who signed up but never confirmed.
//	@Tags			digest
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	api.MessageResponse	"Successfully sent confirmation nudges"
//	@Failure		500	{object}	api.MessageResponse	"Failed to send confirmation nudges"
//	@Router			/digest/nudge [get]
func (h *Handler) Nudge(c *webkit.Context) error {
	ctx := c.Context()

	sent, failed, err := h.subscribers.SendConfirmationNudges(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Send confirmation nudges failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to send confirmation nudges")
	}

	slog.InfoContext(ctx, "Sent confirmation nudges", "sent", sent, "failed", failed)

	return api.OK(c, http.StatusOK, NudgeResult{Sent: sent, Failed: failed}, "Successfully sent confirmation nudges")
}
