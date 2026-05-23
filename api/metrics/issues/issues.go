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

package handler

import (
	"context"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
)

// issueSorts is the allowlist of sort fields for the issues list endpoint.
var issueSorts = []string{
	"click_rate",
	"open_rate",
	"total_clicks",
	"unique_clicks",
	"total_opens",
	"unique_opens",
	"delivered",
	"sent_at",
}

// Handler is the Vercel serverless function entry point for GET /api/metrics/issues.
// Returns per-issue engagement stats with optional filtering and sorting.
func Handler(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		q, err := api.ParseMetricsQuery(r, issueSorts, "sent_at")
		if err != nil {
			api.Error(w, http.StatusBadRequest, err.Error())
			return
		}

		rows, err := a.Repository.Metrics.IssueList(ctx, q.ToFilter(), q.Sort)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "failed to fetch issue metrics")
			return
		}

		api.JSON(w, http.StatusOK, map[string]any{"data": rows})
	})(w, r)
}
