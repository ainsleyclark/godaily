// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"log/slog"
	"net/http"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleydev/webkit/pkg/webkit"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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

// Subscribers handles GET /metrics/subscribers.
// Returns subscriber growth and churn bucketed over time.
func (h *Handler) Subscribers(c *webkit.Context) error {
	var req subscribersRequest
	if err := decoder.Decode(&req, c.Request.URL.Query()); err != nil {
		return webkit.NewError(http.StatusBadRequest, "invalid query parameters")
	}
	if err := req.validate(); err != nil {
		return webkit.NewError(http.StatusBadRequest, err.Error())
	}

	from, to, err := parseDateWindow(req.From, req.To, req.Period)
	if err != nil {
		return webkit.NewError(http.StatusBadRequest, err.Error())
	}

	bucket := req.Bucket
	if bucket == "" {
		bucket = "day"
	}

	data, err := h.metricsRepo.SubscriberGrowth(c.Context(), engagement.MetricsFilter{From: from, To: to}, bucket)
	if err != nil {
		slog.ErrorContext(c.Context(), "failed to fetch subscriber data", "error", err)
		return webkit.NewError(http.StatusInternalServerError, "failed to fetch subscriber data")
	}

	return c.JSON(http.StatusOK, map[string]any{"data": data})
}
