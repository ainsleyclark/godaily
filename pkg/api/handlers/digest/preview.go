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

// Preview handles GET /digest/preview.
// It runs at 6 AM UTC on weekdays, sending the draft digest and AI synth suggestion
// to the owner before the full subscriber send at 8 AM.
func (h *Handler) Preview(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	if api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping preview — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackSendHeartbeatURL)
		return c.NoContent(http.StatusOK)
	}

	today := now.Truncate(24 * time.Hour)

	if err := h.runner.SendPreview(ctx, today); err != nil {
		h.slack.MustSend(ctx, "Send preview failed: "+err.Error())
		return fmt.Errorf("send preview failed: %w", err)
	}

	hook.Heartbeat(ctx, h.config.BetterStackSendHeartbeatURL)

	return c.NoContent(http.StatusOK)
}
