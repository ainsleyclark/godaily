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

type subscribersRequest struct {
	From   string `schema:"from"`
	To     string `schema:"to"`
	Period string `schema:"period"`
	Bucket string `schema:"bucket"`
}

func (req subscribersRequest) validate() error {
	return validation.ValidateStruct(
		&req,
		validation.Field(&req.Bucket, validation.When(
			req.Bucket != "",
			validation.In("day", "week", "month").
				Error("invalid bucket: use day, week, or month"),
		)),
	)
}

// SubscriberMetricsResponse is the response envelope for GET /metrics/subscribers.
type SubscriberMetricsResponse struct {
	Status  int                       `json:"status"`
	Error   bool                      `json:"error"`
	Message string                    `json:"message" example:"Successfully retrieved subscriber data"`
	Data    engagement.SubscriberData `json:"data"`
} //@name SubscriberMetricsResponse

// Subscribers godoc
//
//	@Summary		Subscriber growth and churn.
//	@Description	Returns subscriber growth and churn bucketed over time.
//	@Tags			metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			period	query		string	false	"Relative window: day, week, month, year, all"
//	@Param			from	query		string	false	"Start date (YYYY-MM-DD)"
//	@Param			to		query		string	false	"End date (YYYY-MM-DD)"
//	@Param			bucket	query		string	false	"Bucket: day, week, or month"
//	@Success		200		{object}	SubscriberMetricsResponse					"Successfully retrieved subscriber data"
//	@Failure		400		{object}	api.Response								"Invalid query parameters"
//	@Failure		500		{object}	api.Response								"Failed to fetch subscriber data"
//	@Router			/metrics/subscribers [get]
func (h *Handler) Subscribers(c *webkit.Context) error {
	ctx := c.Context()

	var req subscribersRequest
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

	bucket := req.Bucket
	if bucket == "" {
		bucket = "day"
	}

	data, err := h.metricsRepo.SubscriberGrowth(ctx, engagement.MetricsFilter{From: from, To: to}, bucket)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to fetch subscriber data", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch subscriber data")
	}

	return api.OK(c, http.StatusOK, data, "Successfully retrieved subscriber data")
}
