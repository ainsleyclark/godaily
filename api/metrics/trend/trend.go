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

// validMetrics is the allowlist of trend metric values.
var validMetrics = map[string]struct{}{
	"delivered":     {},
	"unique_opens":  {},
	"total_opens":   {},
	"unique_clicks": {},
	"total_clicks":  {},
	"open_rate":     {},
	"click_rate":    {},
}

// validBuckets is the allowlist of trend bucket values.
var validBuckets = map[string]struct{}{
	"day":  {},
	"week": {},
}

// Handler is the Vercel serverless function entry point for GET /api/metrics/trend.
// Returns a time series for a chosen engagement metric, bucketed by day or week.
func Handler(w http.ResponseWriter, r *http.Request) {
	api.HandleAuth(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		q, err := api.ParseMetricsQuery(r, nil, "")
		if err != nil {
			api.Error(w, http.StatusBadRequest, err.Error())
			return
		}

		metric := r.URL.Query().Get("metric")
		if metric == "" {
			metric = "click_rate"
		}
		if _, ok := validMetrics[metric]; !ok {
			api.Error(w, http.StatusBadRequest, "invalid metric: use delivered, unique_opens, total_opens, unique_clicks, total_clicks, open_rate, or click_rate")
			return
		}

		bucket := r.URL.Query().Get("bucket")
		if bucket == "" {
			bucket = "day"
		}
		if _, ok := validBuckets[bucket]; !ok {
			api.Error(w, http.StatusBadRequest, "invalid bucket: use day or week")
			return
		}

		data, err := a.Repository.Metrics.Trend(ctx, q.ToFilter(), metric, bucket)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "failed to fetch trend data")
			return
		}

		api.JSON(w, http.StatusOK, map[string]any{"data": data})
	})(w, r)
}
