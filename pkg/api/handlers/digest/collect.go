// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/gateway/hook"
	// Register all news-source fetchers (lingua-go + scrapers) so the
	// registry populates in this single binary.
	_ "github.com/ainsleyclark/godaily/pkg/source"
)

// Collect godoc
//
//	@Summary		Run the news collection pipeline.
//	@Description	Fetches and ranks news from all registered sources. Skipped at weekends unless force=true.
//	@Tags			digest
//	@Produce		json
//	@Security		BearerAuth
//	@Param			force	query		bool			false	"Force collection even at weekends"
//	@Success		200		{object}	api.MessageResponse	"Per-source collection results"
//	@Failure		500		{object}	api.MessageResponse	"Failed to collect"
//	@Router			/digest/collect [get]
func (h *Handler) Collect(c *webkit.Context) error {
	ctx := c.Context()
	force := c.Request.URL.Query().Get("force") == "true"
	if !force && api.IsWeekend(time.Now().UTC()) {
		slog.InfoContext(ctx, "Skipping collect — weekend")
		hook.Heartbeat(ctx, h.config.BetterStackCollectHeartbeatURL)
		return api.OK(c, http.StatusOK, nil, "Skipped collect — weekend")
	}

	resp, err := h.runner.Collect(ctx, digest.CollectOptions{})
	if err != nil {
		h.slack.MustSend(ctx, "Collect failed: "+err.Error())
		slog.ErrorContext(ctx, "Collect failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to collect")
	}

	hook.Heartbeat(ctx, h.config.BetterStackCollectHeartbeatURL)

	type sourceResult struct {
		Count int     `json:"count"`
		Error *string `json:"error"`
	}
	sources := make(map[string]sourceResult, len(resp.Sources))
	for _, si := range resp.Sources {
		sources[string(si.Source)] = sourceResult{Count: len(si.Items)}
	}
	var errParts []string
	for src, srcErr := range resp.Errors {
		msg := srcErr.Error()
		sources[string(src)] = sourceResult{Error: &msg}
		errParts = append(errParts, fmt.Sprintf("• %s: %s", src, msg))
	}
	if len(errParts) > 0 {
		h.slack.MustSend(ctx, "Source errors during collection:\n"+strings.Join(errParts, "\n"))
	}

	return api.OK(c, http.StatusOK, map[string]any{"sources": sources}, "Successfully collected sources")
}
