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

package metrics

import (
	"context"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
		validation.Field(&req.Limit, validation.Min(0), validation.Max(api.MaxMetricsLimit)),
	)
}

// HandleIssues is the Vercel serverless function entry point for GET /api/metrics/issues.
// Returns per-issue engagement stats with optional filtering and sorting.
func HandleIssues(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		var req issuesRequest
		if err := api.Decoder.Decode(&req, r.URL.Query()); err != nil {
			api.Error(w, http.StatusBadRequest, "invalid query parameters")
			return
		}
		if err := req.validate(); err != nil {
			api.Error(w, http.StatusBadRequest, err.Error())
			return
		}

		from, to, err := api.ParseDateWindow(req.From, req.To, req.Period)
		if err != nil {
			api.Error(w, http.StatusBadRequest, err.Error())
			return
		}

		sort := req.Sort
		if sort == "" {
			sort = "sent_at"
		}
		limit := req.Limit
		if limit == 0 {
			limit = api.DefaultMetricsLimit
		}

		rows, err := a.Repository.Metrics.IssueList(ctx, engagement.MetricsFilter{From: from, To: to, Limit: limit}, sort)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "failed to fetch issue metrics")
			return
		}

		api.JSON(w, http.StatusOK, map[string]any{"data": rows})
	})(w, r)
}
