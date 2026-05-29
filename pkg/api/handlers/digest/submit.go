// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/source"
)

// SubmitReddit godoc
//
//	@Summary		Manually submit raw Reddit listing JSON.
//	@Description	Fallback for when the live Reddit fetch is blocked (e.g. ScraperAPI 403). Accepts the raw JSON body returned by https://www.reddit.com/r/golang/new.json, transforms it through the standard pipeline, and persists items that fall within the current collection window. Idempotent: skipped if Reddit items already exist for the window.
//	@Tags			digest
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	api.MessageResponse	"Items submitted, skipped (already present), or none in window"
//	@Failure		400	{object}	api.MessageResponse	"Unreadable or invalid Reddit JSON payload"
//	@Failure		500	{object}	api.MessageResponse	"Failed to submit items"
//	@Router			/digest/submit-reddit [post]
func (h *Handler) SubmitReddit(c *webkit.Context) error {
	ctx := c.Context()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return api.Error(c, http.StatusBadRequest, "Error reading request body")
	}
	if len(body) == 0 {
		return api.Error(c, http.StatusBadRequest, "Empty request body")
	}

	items, err := source.ParseReddit(ctx, body)
	if err != nil {
		slog.WarnContext(ctx, "Rejected Reddit submission with invalid JSON", "err", err)
		return api.Error(c, http.StatusBadRequest, "Invalid Reddit JSON payload")
	}

	resp, err := h.runner.Submit(ctx, news.SourceReddit, items)
	if err != nil {
		h.slack.MustSend(ctx, "Reddit submission failed: "+err.Error())
		slog.ErrorContext(ctx, "Reddit submission failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to submit items")
	}

	data := map[string]any{
		"received":  resp.Received,
		"persisted": resp.Persisted,
		"skipped":   resp.Skipped,
	}
	switch {
	case resp.Skipped:
		return api.OK(c, http.StatusOK, data, "Reddit items already collected for this window — skipped")
	case resp.Persisted == 0:
		return api.OK(c, http.StatusOK, data, "No submitted items fall within the current collection window")
	default:
		return api.OK(c, http.StatusOK, data, "Successfully submitted Reddit items")
	}
}
