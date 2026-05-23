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

	validation "github.com/go-ozzo/ozzo-validation/v4"
	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

type trendRequest struct {
	From   string `schema:"from"`
	To     string `schema:"to"`
	Period string `schema:"period"`
	Metric string `schema:"metric"`
	Bucket string `schema:"bucket"`
}

func (req trendRequest) validate() error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.Metric, validation.When(req.Metric != "",
			validation.In("delivered", "unique_opens", "total_opens", "unique_clicks", "total_clicks", "open_rate", "click_rate").
				Error("invalid metric: use delivered, unique_opens, total_opens, unique_clicks, total_clicks, open_rate, or click_rate"),
		)),
		validation.Field(&req.Bucket, validation.When(req.Bucket != "",
			validation.In("day", "week").
				Error("invalid bucket: use day or week"),
		)),
	)
}

// Handler is the Vercel serverless function entry point for GET /api/metrics/trend.
// Returns a time series for a chosen engagement metric, bucketed by day or week.
func Handler(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		var req trendRequest
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

		metric := req.Metric
		if metric == "" {
			metric = "click_rate"
		}
		bucket := req.Bucket
		if bucket == "" {
			bucket = "day"
		}

		data, err := a.Repository.Metrics.Trend(ctx, engagement.MetricsFilter{From: from, To: to}, metric, bucket)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "failed to fetch trend data")
			return
		}

		api.JSON(w, http.StatusOK, map[string]any{"data": data})
	})(w, r)
}
