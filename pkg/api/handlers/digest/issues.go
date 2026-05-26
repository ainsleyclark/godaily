// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"net/http"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Issues handles GET /digest/issues.
// An optional ?status= query parameter filters by issue status (e.g. "draft", "sent").
func (h *Handler) Issues(c *webkit.Context) error {
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

	opts := store.ListOptions{Page: page, PerPage: perPage}
	statusParam := r.URL.Query().Get("status")

	var (
		total  int64
		issues []digest.Issue
		err    error
	)

	if statusParam != "" {
		status := digest.IssueStatus(statusParam)
		total, err = h.issuesRepo.CountByStatus(ctx, status)
		if err != nil {
			return webkit.NewError(http.StatusInternalServerError, "failed to count issues")
		}
		issues, err = h.issuesRepo.ListByStatus(ctx, status, opts)
		if err != nil {
			return webkit.NewError(http.StatusInternalServerError, "failed to list issues")
		}
	} else {
		total, err = h.issuesRepo.Count(ctx)
		if err != nil {
			return webkit.NewError(http.StatusInternalServerError, "failed to count issues")
		}
		issues, err = h.issuesRepo.List(ctx, opts)
		if err != nil {
			return webkit.NewError(http.StatusInternalServerError, "failed to list issues")
		}
	}

	return c.JSON(http.StatusOK, api.PaginatedResponse[digest.Issue]{
		Data:    issues,
		Page:    page,
		PerPage: perPage,
		Total:   total,
	})
}
