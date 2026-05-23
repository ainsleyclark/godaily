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

package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/gorilla/schema"
)

const (
	defaultMetricsLimit = 10
	maxMetricsLimit     = 100
)

// periodDays maps period shorthand values to rolling day counts.
// A value of 0 means "all time" (no window applied).
var periodDays = map[string]int{
	"day":   1,
	"week":  7,
	"month": 30,
	"year":  365,
	"all":   0,
}

// queryDecoder is the shared gorilla/schema decoder for query parameters.
// IgnoreUnknownKeys allows endpoint-specific params (e.g. bucket, metric) to be
// present in the query string without causing decode errors in the shared parser.
var queryDecoder = func() *schema.Decoder {
	d := schema.NewDecoder()
	d.IgnoreUnknownKeys(true)
	return d
}()

// rawMetricsQuery is the intermediate struct decoded from URL query parameters
// using gorilla/schema before validation.
type rawMetricsQuery struct {
	From   string `schema:"from"`
	To     string `schema:"to"`
	Period string `schema:"period"`
	Sort   string `schema:"sort"`
	Limit  int    `schema:"limit"`
}

// MetricsQuery holds the parsed and validated common query parameters accepted
// by every /api/metrics list endpoint.
type MetricsQuery struct {
	From  *time.Time
	To    *time.Time
	Sort  string
	Limit int
}

// ToFilter converts the parsed query into the store-layer MetricsFilter.
func (q MetricsQuery) ToFilter() engagement.MetricsFilter {
	return engagement.MetricsFilter{
		From:  q.From,
		To:    q.To,
		Limit: q.Limit,
	}
}

// HTTPError is an HTTP error that carries its status code and can write itself
// to a ResponseWriter as a JSON body {"error": "..."}.
type HTTPError struct {
	Status  int
	Message string
}

// Write writes the error to w as JSON.
func (e *HTTPError) Write(w http.ResponseWriter) {
	Error(w, e.Status, e.Message)
}

// ParseMetricsQuery decodes and validates the common query parameters for
// /api/metrics/* endpoints. It uses gorilla/schema to map the query string
// into rawMetricsQuery, then applies business-rule validation.
//
// Validation errors return a non-nil *HTTPError with Status 400.
func ParseMetricsQuery(r *http.Request, allowedSorts []string, defaultSort string) (MetricsQuery, *HTTPError) {
	raw := rawMetricsQuery{Limit: defaultMetricsLimit}
	if err := queryDecoder.Decode(&raw, r.URL.Query()); err != nil {
		return MetricsQuery{}, &HTTPError{Status: http.StatusBadRequest, Message: "invalid query parameters"}
	}

	q := MetricsQuery{
		Sort:  defaultSort,
		Limit: raw.Limit,
	}

	// Zero limit means the param was absent; keep the default.
	if q.Limit == 0 {
		q.Limit = defaultMetricsLimit
	}

	// Parse from/to dates.
	if raw.From != "" {
		t, err := time.Parse("2006-01-02", raw.From)
		if err != nil {
			return MetricsQuery{}, &HTTPError{Status: http.StatusBadRequest, Message: fmt.Sprintf("invalid from date: %q", raw.From)}
		}
		q.From = &t
	}
	if raw.To != "" {
		t, err := time.Parse("2006-01-02", raw.To)
		if err != nil {
			return MetricsQuery{}, &HTTPError{Status: http.StatusBadRequest, Message: fmt.Sprintf("invalid to date: %q", raw.To)}
		}
		q.To = &t
	}

	// from must be strictly before to when both are present.
	if q.From != nil && q.To != nil && !q.From.Before(*q.To) {
		return MetricsQuery{}, &HTTPError{Status: http.StatusBadRequest, Message: "from must be before to"}
	}

	// Resolve period only when neither from nor to is set.
	if raw.Period != "" && q.From == nil && q.To == nil {
		days, ok := periodDays[raw.Period]
		if !ok {
			return MetricsQuery{}, &HTTPError{Status: http.StatusBadRequest, Message: fmt.Sprintf("unknown period: %q", raw.Period)}
		}
		if days > 0 {
			now := time.Now().UTC()
			from := now.AddDate(0, 0, -days)
			q.From = &from
			q.To = &now
		}
	} else if raw.Period != "" && !isKnownPeriod(raw.Period) {
		// period is set alongside from/to; still validate it even though it's ignored.
		return MetricsQuery{}, &HTTPError{Status: http.StatusBadRequest, Message: fmt.Sprintf("unknown period: %q", raw.Period)}
	}

	// Validate sort if explicitly provided.
	if raw.Sort != "" {
		if !contains(allowedSorts, raw.Sort) {
			return MetricsQuery{}, &HTTPError{Status: http.StatusBadRequest, Message: fmt.Sprintf("unknown sort: %q", raw.Sort)}
		}
		q.Sort = raw.Sort
	}

	// Validate limit bounds.
	if q.Limit < 1 {
		return MetricsQuery{}, &HTTPError{Status: http.StatusBadRequest, Message: "limit must be at least 1"}
	}
	if q.Limit > maxMetricsLimit {
		return MetricsQuery{}, &HTTPError{Status: http.StatusBadRequest, Message: fmt.Sprintf("limit must be at most %d", maxMetricsLimit)}
	}

	return q, nil
}

// contains reports whether s is in the slice.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// isKnownPeriod reports whether period is a recognised period shorthand.
func isKnownPeriod(period string) bool {
	_, ok := periodDays[period]
	return ok
}
