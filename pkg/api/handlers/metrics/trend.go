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

type trendRequest struct {
	From   string `schema:"from"`
	To     string `schema:"to"`
	Period string `schema:"period"`
	Metric string `schema:"metric"`
	Bucket string `schema:"bucket"`
}

func (req trendRequest) validate() error {
	return validation.ValidateStruct(
		&req,
		validation.Field(&req.Metric, validation.When(
			req.Metric != "",
			validation.In("delivered", "unique_opens", "total_opens", "unique_clicks", "total_clicks", "open_rate", "click_rate").
				Error("invalid metric: use delivered, unique_opens, total_opens, unique_clicks, total_clicks, open_rate, or click_rate"),
		)),
		validation.Field(&req.Bucket, validation.When(
			req.Bucket != "",
			validation.In("day", "week").
				Error("invalid bucket: use day or week"),
		)),
	)
}

// TrendResponse is the response envelope for GET /metrics/trend.
type TrendResponse struct {
	Status  int                  `json:"status"`
	Error   bool                 `json:"error"`
	Message string               `json:"message" example:"Successfully retrieved trend data"`
	Data    engagement.TrendData `json:"data"`
} //@name TrendResponse

// Trend godoc
//
//	@Summary		Engagement time series.
//	@Description	Returns a time series for a chosen engagement metric, bucketed by day or week.
//	@Tags			metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			period	query		string	false	"Relative window: day, week, month, year, all"
//	@Param			from	query		string	false	"Start date (YYYY-MM-DD)"
//	@Param			to		query		string	false	"End date (YYYY-MM-DD)"
//	@Param			metric	query		string	false	"Metric: delivered, unique_opens, total_opens, unique_clicks, total_clicks, open_rate, click_rate"
//	@Param			bucket	query		string	false	"Bucket: day or week"
//	@Success		200		{object}	TrendResponse								"Successfully retrieved trend data"
//	@Failure		400		{object}	api.Response							"Invalid query parameters"
//	@Failure		500		{object}	api.Response							"Failed to fetch trend data"
//	@Router			/metrics/trend [get]
func (h *Handler) Trend(c *webkit.Context) error {
	ctx := c.Context()

	var req trendRequest
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

	metric := req.Metric
	if metric == "" {
		metric = "click_rate"
	}
	bucket := req.Bucket
	if bucket == "" {
		bucket = "day"
	}

	data, err := h.metricsRepo.Trend(ctx, engagement.MetricsFilter{From: from, To: to}, metric, bucket)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to fetch trend data", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch trend data")
	}

	return api.OK(c, http.StatusOK, data, "Successfully retrieved trend data")
}
