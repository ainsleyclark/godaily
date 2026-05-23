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

package issues

import (
	"context"
	"errors"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// topLinksLimit is the maximum number of top-clicked links returned for an issue.
const topLinksLimit = 10

// Handler is the Vercel serverless function entry point for GET /api/metrics/issues/:slug.
// The slug path segment is injected by Vercel as the "slug" query parameter.
func Handler(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		slug := r.URL.Query().Get("slug")
		if slug == "" {
			api.Error(w, http.StatusBadRequest, "slug is required")
			return
		}

		issue, err := a.Repository.Issues.FindBySlug(ctx, slug)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				api.Error(w, http.StatusNotFound, "issue not found")
				return
			}
			api.Error(w, http.StatusInternalServerError, "failed to fetch issue")
			return
		}

		stats, err := a.Repository.EmailEvents.IssueStats(ctx, issue.ID)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "failed to fetch issue stats")
			return
		}

		links, err := a.Repository.EmailEvents.TopLinks(ctx, issue.ID, topLinksLimit)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "failed to fetch top links")
			return
		}

		api.JSON(w, http.StatusOK, map[string]any{
			"stats": stats,
			"links": links,
		})
	})(w, r)
}
