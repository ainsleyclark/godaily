// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
)

// IssueListResponse is the response envelope for GET /issues.
type IssueListResponse = api.Response[api.PaginatedResponse[digest.Issue]] //@name IssueListResponse

// List godoc
//
//	@Summary		List digest issues.
//	@Description	Returns a paginated list of digest issues, optionally filtered by status.
//	@Tags			issues
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status		query		string	false	"Filter by issue status (e.g. draft, sent)"
//	@Param			page		query		int		false	"Page number (default 1)"
//	@Param			per_page	query		int		false	"Items per page (default 20, max 100)"
//	@Success		200			{object}	IssueListResponse										"Successfully retrieved issues"
//	@Failure		500			{object}	api.MessageResponse											"Failed to list issues"
//	@Router			/issues [get]
func (h *Handler) List(c *webkit.Context) error {
	ctx := c.Context()
	r := c.Request

	page := api.QueryInt(r, "page", api.DefaultPage)
	perPage := api.QueryInt(r, "per_page", api.DefaultPerPage)

	if page < 1 {
		page = api.DefaultPage
	}
	if perPage < 1 || perPage > api.MaxPerPage {
		perPage = api.DefaultPerPage
	}

	opts := digest.IssueListOptions{Page: page, PerPage: perPage}
	if s := r.URL.Query().Get("status"); s != "" {
		status := digest.IssueStatus(s)
		opts.Status = &status
	}

	total, err := h.issuesRepo.Count(ctx, opts)
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to count issues")
	}
	issues, err := h.issuesRepo.List(ctx, opts)
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to list issues")
	}

	return api.OK(c, http.StatusOK, api.PaginatedResponse[digest.Issue]{
		Data:    issues,
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}, "Successfully retrieved issues")
}
