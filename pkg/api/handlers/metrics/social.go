// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

type socialRequest struct {
	From   string `schema:"from"`
	To     string `schema:"to"`
	Period string `schema:"period"`
}

func (req socialRequest) validate() error {
	return validation.ValidateStruct(&req)
}

// SocialMetricsResponse is the response envelope for GET /metrics/social.
type SocialMetricsResponse = api.Response[[]engagement.SocialPostEngagement] //@name SocialMetricsResponse

// Social godoc
//
//	@Summary		Social post engagement metrics.
//	@Description	Returns social posts joined with their latest engagement counts (likes, reposts, comments, impressions), optionally filtered by date range.
//	@Tags			metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			period	query		string	false	"Relative window: day, week, month, year, all"
//	@Param			from	query		string	false	"Start date (YYYY-MM-DD)"
//	@Param			to		query		string	false	"End date (YYYY-MM-DD)"
//	@Success		200		{object}	SocialMetricsResponse	"Successfully retrieved social metrics"
//	@Failure		400		{object}	api.MessageResponse		"Invalid query parameters"
//	@Failure		500		{object}	api.MessageResponse		"Failed to fetch social metrics"
//	@Router			/metrics/social [get]
func (h *Handler) Social(c *webkit.Context) error {
	ctx := c.Context()

	var req socialRequest
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

	posts, err := h.socialMetrics.List(ctx, engagement.MetricsFilter{From: from, To: to})
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch social metrics")
	}

	return api.OK(c, http.StatusOK, posts, "Successfully retrieved social metrics")
}
