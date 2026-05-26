// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"net/http"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleydev/webkit/pkg/webkit"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type tagsRequest struct {
	From   string `schema:"from"`
	To     string `schema:"to"`
	Period string `schema:"period"`
	Limit  int    `schema:"limit"`
}

func (req tagsRequest) validate() error {
	return validation.ValidateStruct(
		&req,
		validation.Field(&req.Limit, validation.Min(0), validation.Max(MaxMetricsLimit)),
	)
}

// Tags handles GET /metrics/tags.
// Returns total clicks aggregated by item tag.
func (h *Handler) Tags(c *webkit.Context) error {
	var req tagsRequest
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

	limit := req.Limit
	if limit == 0 {
		limit = DefaultMetricsLimit
	}

	rows, err := h.metricsRepo.TagList(c.Context(), engagement.MetricsFilter{From: from, To: to, Limit: limit})
	if err != nil {
		return webkit.NewError(http.StatusInternalServerError, "failed to fetch tag metrics")
	}

	return c.JSON(http.StatusOK, map[string]any{"data": rows})
}
