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

// Build godoc
//
//	@Summary		Build the draft digest.
//	@Description	Assembles today's collected items into a draft digest issue and sends the owner preview as a best-effort follow-up. Skipped at weekends unless force=true.
//	@Tags			digest
//	@Produce		json
//	@Security		BearerAuth
//	@Param			force	query		bool			false	"Force build even at weekends"
//	@Success		200		{object}	api.MessageResponse	"Successfully built digest"
//	@Failure		500		{object}	api.MessageResponse	"Failed to build digest"
//	@Router			/digest/build [get]
func (h *Handler) Build(c *webkit.Context) error {
	ctx := c.Context()
	now := time.Now().UTC()
	force := c.Request.URL.Query().Get("force") == "true"
	if !force && api.IsWeekend(now) {
		slog.InfoContext(ctx, "Skipping build — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackBuildHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped build — weekend")
	}

	if err := h.runner.Build(ctx, now); err != nil {
		slog.ErrorContext(ctx, "Build failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to build digest")
	}

	// Regenerate the static site so today's freshly built draft is published
	// as a live copy at /issues/{slug}/ — the page is rendered but kept out of
	// the archive, sitemap, and RSS feed until the issue is actually sent.
	hook.Deploy(ctx, h.config.VercelDeployHookURL)
	hook.Heartbeat(ctx, h.config.BetterStackBuildHeartbeatURL)

	return api.OK(c, http.StatusOK, nil, "Successfully built digest")
}
