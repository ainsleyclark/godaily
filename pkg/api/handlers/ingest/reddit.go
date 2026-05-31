// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ingest

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/source"
)

// Reddit godoc
//
//	@Summary		Manually ingest raw Reddit listing JSON.
//	@Description	Fallback for when the live Reddit fetch is blocked (e.g. ScraperAPI 403). Accepts the raw JSON body returned by https://www.reddit.com/r/golang/new.json, transforms it through the standard pipeline, and persists items that fall within the current collection window. De-duplicates on (url, tag), so it is safe to run repeatedly (e.g. on a schedule).
//	@Tags			ingest
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	api.MessageResponse	"Items ingested, all duplicates, or none in window"
//	@Failure		400	{object}	api.MessageResponse	"Unreadable or invalid Reddit JSON payload"
//	@Failure		500	{object}	api.MessageResponse	"Failed to ingest items"
//	@Router			/ingest/reddit [post]
func (h *Handler) Reddit(c *webkit.Context) error {
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
		slog.WarnContext(ctx, "Rejected Reddit ingest with invalid JSON", "err", err)
		return api.Error(c, http.StatusBadRequest, "Invalid Reddit JSON payload")
	}

	resp, err := h.runner.Submit(ctx, news.SourceReddit, items)
	if err != nil {
		h.slack.MustSend(ctx, slack.Error("Reddit ingest failed", err))
		slog.ErrorContext(ctx, "Reddit ingest failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to ingest items")
	}

	data := map[string]any{
		"received":   resp.Received,
		"persisted":  resp.Persisted,
		"duplicates": resp.Duplicates,
	}
	switch {
	case resp.Persisted > 0:
		return api.OK(c, http.StatusOK, data,
			fmt.Sprintf("Ingested %d new Reddit items (%d duplicates skipped)", resp.Persisted, resp.Duplicates))
	case resp.Duplicates > 0:
		return api.OK(c, http.StatusOK, data, "All submitted Reddit items already present — nothing new added")
	default:
		return api.OK(c, http.StatusOK, data, "No submitted items fall within the current collection window")
	}
}
