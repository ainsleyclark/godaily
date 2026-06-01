// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"errors"
	"net/http"
	"time"

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
type IssueMetricsResponse = api.Response[[]engagement.IssueEngagement] //@name IssueMetricsResponse

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
//	@Failure		400		{object}	api.MessageResponse									"Invalid query parameters"
//	@Failure		500		{object}	api.MessageResponse									"Failed to fetch issue metrics"
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

// IssueDetail bundles single-issue engagement stats with its top-clicked links.
type IssueDetail struct {
	Stats engagement.IssueStats   `json:"stats"`
	Links []engagement.LinkClicks `json:"links"`
}

// IssueDetailResponse is the response envelope for GET /metrics/issues/{slug}.
type IssueDetailResponse = api.Response[IssueDetail] //@name IssueDetailResponse

// IssueBySlug godoc
//
//	@Summary		Single-issue stats and top links.
//	@Description	Returns engagement stats plus the top-clicked links for one issue identified by its date slug.
//	@Tags			metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string				true	"Issue date slug"
//	@Success		200		{object}	IssueDetailResponse	"Issue stats and top links"
//	@Failure		400		{object}	api.MessageResponse	"Slug is required"
//	@Failure		404		{object}	api.MessageResponse	"Issue not found"
//	@Failure		500		{object}	api.MessageResponse	"Failed to fetch issue metrics"
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

	return api.OK(c, http.StatusOK, IssueDetail{
		Stats: stats,
		Links: links,
	}, "Successfully retrieved issue metrics")
}

// IssueTrend godoc
//
//	@Summary		Single-issue engagement time series.
//	@Description	Returns a time series for one issue's chosen engagement metric, bucketed by day or week. When no window is supplied it defaults to the issue's send date through now.
//	@Tags			metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string			true	"Issue date slug"
//	@Param			period	query		string			false	"Relative window: day, week, month, year, all"
//	@Param			from	query		string			false	"Start date (YYYY-MM-DD)"
//	@Param			to		query		string			false	"End date (YYYY-MM-DD)"
//	@Param			metric	query		string			false	"Metric: delivered, unique_opens, total_opens, unique_clicks, total_clicks, open_rate, click_rate"
//	@Param			bucket	query		string			false	"Bucket: day or week"
//	@Success		200		{object}	TrendResponse		"Successfully retrieved issue trend data"
//	@Failure		400		{object}	api.MessageResponse	"Invalid query parameters"
//	@Failure		404		{object}	api.MessageResponse	"Issue not found"
//	@Failure		500		{object}	api.MessageResponse	"Failed to fetch issue trend data"
//	@Router			/metrics/issues/{slug}/trend [get]
func (h *Handler) IssueTrend(c *webkit.Context) error {
	ctx := c.Context()

	slug := c.Param("slug")
	if slug == "" {
		return api.Error(c, http.StatusBadRequest, "Slug is required")
	}

	var req trendRequest
	if err := decoder.Decode(&req, c.Request.URL.Query()); err != nil {
		return api.Error(c, http.StatusBadRequest, "Invalid query parameters")
	}
	if err := req.validate(); err != nil {
		return api.Error(c, http.StatusBadRequest, err.Error())
	}

	issue, err := h.issuesRepo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return api.Error(c, http.StatusNotFound, "Issue not found")
		}
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch issue")
	}

	from, to, err := parseDateWindow(req.From, req.To, req.Period)
	if err != nil {
		return api.Error(c, http.StatusBadRequest, err.Error())
	}
	// Default to the issue's lifetime (send date through now) so the series is
	// zero-filled from when the issue went out rather than collapsing to the
	// buckets that happen to have events.
	if from == nil && to == nil {
		sent := issue.SentAt.UTC().Truncate(24 * time.Hour)
		now := time.Now().UTC()
		from = &sent
		to = &now
	}

	metric := req.Metric
	if metric == "" {
		metric = "unique_clicks"
	}
	bucket := req.Bucket
	if bucket == "" {
		bucket = "day"
	}

	data, err := h.metricsRepo.IssueTrend(ctx, issue.ID, engagement.MetricsFilter{From: from, To: to}, metric, bucket)
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to fetch issue trend data")
	}

	return api.OK(c, http.StatusOK, data, "Successfully retrieved issue trend data")
}
