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

type sourcesRequest struct {
	From   string `schema:"from"`
	To     string `schema:"to"`
	Period string `schema:"period"`
	Limit  int    `schema:"limit"`
}

func (req sourcesRequest) validate() error {
	return validation.ValidateStruct(
		&req,
		validation.Field(&req.Limit, validation.Min(0), validation.Max(MaxMetricsLimit)),
	)
}

// SourceMetricsResponse is the response envelope for GET /metrics/sources.
type SourceMetricsResponse = api.Response[[]engagement.SourceMetrics] //@name SourceMetricsResponse

// Sources godoc
//
//	@Summary		Clicks aggregated by source.
//	@Description	Returns total clicks aggregated by item source.
//	@Tags			metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			period	query		string	false	"Relative window: day, week, month, year, all"
//	@Param			from	query		string	false	"Start date (YYYY-MM-DD)"
//	@Param			to		query		string	false	"End date (YYYY-MM-DD)"
//	@Param			limit	query		int		false	"Max rows (max 100)"
//	@Success		200		{object}	SourceMetricsResponse						"Successfully retrieved source metrics"
//	@Failure		400		{object}	api.MessageResponse								"Invalid query parameters"
//	@Failure		500		{object}	api.MessageResponse								"Failed to fetch source metrics"
//	@Router			/metrics/sources [get]
func (h *Handler) Sources(c *webkit.Context) error {
	ctx := c.Context()

	var req sourcesRequest
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

	rows, err := h.metricsRepo.SourceList(ctx, engagement.MetricsFilter{From: from, To: to, Limit: limit})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to fetch source metrics", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch source metrics")
	}

	return api.OK(c, http.StatusOK, rows, "Successfully retrieved source metrics")
}
