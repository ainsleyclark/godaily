// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

type summaryRequest struct {
	From   string `schema:"from"`
	To     string `schema:"to"`
	Period string `schema:"period"`
}

// Summary handles GET /metrics/summary.
// Returns headline engagement numbers for a period.
func (h *Handler) Summary(c *webkit.Context) error {
	var req summaryRequest
	if err := decoder.Decode(&req, c.Request.URL.Query()); err != nil {
		return api.Error(c, http.StatusBadRequest, "Invalid query parameters")
	}

	from, to, err := parseDateWindow(req.From, req.To, req.Period)
	if err != nil {
		return api.Error(c, http.StatusBadRequest, err.Error())
	}

	stats, err := h.metricsRepo.Summary(c.Context(), engagement.MetricsFilter{From: from, To: to})
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch summary stats")
	}

	return api.OK(c, http.StatusOK, stats, "Successfully retrieved summary stats")
}
