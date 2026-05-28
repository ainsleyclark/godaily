// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"errors"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/store"
)

type issuesRequest struct {
	From   string `schema:"from"`
	To     string `schema:"to"`
	Period string `schema:"period"`
	Limit  int    `schema:"limit"`
	Sort   string `schema:"sort"`
}

func (req issuesRequest) validate() error {
	return validation.ValidateStruct(
		&req,
		validation.Field(&req.Sort, validation.When(
			req.Sort != "",
			validation.In("click_rate", "open_rate", "total_clicks", "unique_clicks", "total_opens", "unique_opens", "delivered", "sent_at").
				Error("invalid sort: use click_rate, open_rate, total_clicks, unique_clicks, total_opens, unique_opens, delivered, or sent_at"),
		)),
		validation.Field(&req.Limit, validation.Min(0), validation.Max(MaxMetricsLimit)),
	)
}

// IssueMetricsResponse is the response envelope for GET /metrics/issues.
type IssueMetricsResponse struct {
	Status  int                          `json:"status"`
	Error   bool                         `json:"error"`
	Message string                       `json:"message" example:"Successfully retrieved issue metrics"`
	Data    []engagement.IssueEngagement `json:"data"`
} //@name IssueMetricsResponse

// Issues godoc
//
//	@Summary		Per-issue engagement stats.
//	@Description	Returns engagement metrics per issue with optional date filtering and sorting.
//	@Tags			metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			period	query		string	false	"Relative window: day, week, month, year, all"
//	@Param			from	query		string	false	"Start date (YYYY-MM-DD)"
//	@Param			to		query		string	false	"End date (YYYY-MM-DD)"
//	@Param			limit	query		int		false	"Max rows (max 100)"
//	@Param			sort	query		string	false	"Sort key: click_rate, open_rate, total_clicks, unique_clicks, total_opens, unique_opens, delivered, sent_at"
//	@Success		200		{object}	IssueMetricsResponse							"Successfully retrieved issue metrics"
//	@Failure		400		{object}	api.Response									"Invalid query parameters"
//	@Failure		500		{object}	api.Response									"Failed to fetch issue metrics"
//	@Router			/metrics/issues [get]
func (h *Handler) Issues(c *webkit.Context) error {
	ctx := c.Context()

	var req issuesRequest
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

	sort := req.Sort
	if sort == "" {
		sort = "sent_at"
	}
	limit := req.Limit
	if limit == 0 {
		limit = DefaultMetricsLimit
	}

	rows, err := h.metricsRepo.IssueList(ctx, engagement.MetricsFilter{From: from, To: to, Limit: limit}, sort)
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch issue metrics")
	}

	return api.OK(c, http.StatusOK, rows, "Successfully retrieved issue metrics")
}

// topLinksLimit is the maximum number of top-clicked links returned per issue.
const topLinksLimit = 10

// IssueBySlug godoc
//
//	@Summary		Single-issue stats and top links.
//	@Description	Returns engagement stats plus the top-clicked links for one issue identified by its date slug.
//	@Tags			metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string			true	"Issue date slug"
//	@Success		200		{object}	api.Response	"Issue stats and top links"
//	@Failure		400		{object}	api.Response	"Slug is required"
//	@Failure		404		{object}	api.Response	"Issue not found"
//	@Failure		500		{object}	api.Response	"Failed to fetch issue metrics"
//	@Router			/metrics/issues/{slug} [get]
func (h *Handler) IssueBySlug(c *webkit.Context) error {
	ctx := c.Context()

	slug := c.Param("slug")
	if slug == "" {
		return api.Error(c, http.StatusBadRequest, "Slug is required")
	}

	issue, err := h.issuesRepo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.Error(c, http.StatusNotFound, "Issue not found")
		}
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch issue")
	}

	stats, err := h.emailEvents.IssueStats(ctx, issue.ID)
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch issue stats")
	}

	links, err := h.emailEvents.TopLinks(ctx, issue.ID, topLinksLimit)
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch top links")
	}

	return api.OK(c, http.StatusOK, map[string]any{
		"stats": stats,
		"links": links,
	}, "Successfully retrieved issue metrics")
}
