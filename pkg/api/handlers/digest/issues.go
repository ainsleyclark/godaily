// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package digest

import (
	"context"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// HandleIssues handles GET /digest/issues.
// An optional ?status= query parameter filters by issue status (e.g. "draft", "sent").
func HandleIssues(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
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
			total, err = a.Repository.Issues.CountByStatus(ctx, status)
			if err != nil {
				api.Error(w, http.StatusInternalServerError, "failed to count issues")
				return
			}
			issues, err = a.Repository.Issues.ListByStatus(ctx, status, opts)
			if err != nil {
				api.Error(w, http.StatusInternalServerError, "failed to list issues")
				return
			}
		} else {
			total, err = a.Repository.Issues.Count(ctx)
			if err != nil {
				api.Error(w, http.StatusInternalServerError, "failed to count issues")
				return
			}
			issues, err = a.Repository.Issues.List(ctx, opts)
			if err != nil {
				api.Error(w, http.StatusInternalServerError, "failed to list issues")
				return
			}
		}

		api.JSON(w, http.StatusOK, api.PaginatedResponse[digest.Issue]{
			Data:    issues,
			Page:    page,
			PerPage: perPage,
			Total:   total,
		})
	})(w, r)
}
