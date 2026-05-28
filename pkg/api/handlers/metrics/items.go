// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"log/slog"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

type itemsRequest struct {
	From   string `schema:"from"`
	To     string `schema:"to"`
	Period string `schema:"period"`
	Limit  int    `schema:"limit"`
}

func (req itemsRequest) validate() error {
	return validation.ValidateStruct(
		&req,
		validation.Field(&req.Limit, validation.Min(0), validation.Max(MaxMetricsLimit)),
	)
}

// Items handles GET /metrics/items.
// Returns the top-clicked news items enriched with title, tag, and source metadata.
func (h *Handler) Items(c *webkit.Context) error {
	ctx := c.Context()

	var req itemsRequest
	if err := decoder.Decode(&req, c.Request.URL.Query()); err != nil {
		return api.Error(c, http.StatusBadRequest, "Invalid query parameters")
	}
	if err := req.validate(); err != nil {
		return api.Error(c, http.StatusBadRequest, err.Error())
	}

	from, to, err := parseDateWindow(req.From, req.To, req.Period)
	if err != nil {
		return api.Error(c, http.StatusBadRequest, err.Error())
	}

	limit := req.Limit
	if limit == 0 {
		limit = DefaultMetricsLimit
	}

	rows, err := h.metricsRepo.ItemList(ctx, engagement.MetricsFilter{From: from, To: to, Limit: limit})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to fetch item metrics", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch item metrics")
	}

	return api.OK(c, http.StatusOK, rows, "Successfully retrieved item metrics")
}
