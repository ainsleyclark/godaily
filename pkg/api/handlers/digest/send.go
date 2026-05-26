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

// Send handles GET /digest/send.
func (h *Handler) Send(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	force := c.Request.URL.Query().Get("force") == "true"
	if !force && api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping send — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackSendHeartbeatURL)
		return c.NoContent(http.StatusOK)
	}

	today := now.Truncate(24 * time.Hour)

	if err := h.runner.SendDigest(ctx, today, force); err != nil {
		h.slack.MustSend(ctx, "Send digest failed: "+err.Error())
		return fmt.Errorf("send digest failed: %w", err)
	}

	hook.Deploy(ctx, h.config.VercelDeployHookURL)
	hook.Heartbeat(ctx, h.config.BetterStackSendHeartbeatURL)

	return c.NoContent(http.StatusOK)
}
