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
	"time"

	"github.com/gorilla/schema"
)

const (
	DefaultMetricsLimit = 10
	MaxMetricsLimit     = 100
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

// Decoder is the shared gorilla/schema decoder for metrics query parameters.
// IgnoreUnknownKeys is set so endpoint-specific params (bucket, metric, etc.)
// do not cause decode errors when processed by handlers that don't declare them.
var Decoder = func() *schema.Decoder {
	d := schema.NewDecoder()
	d.IgnoreUnknownKeys(true)
	return d
}()

// ParseDateWindow resolves optional from/to time bounds from raw query string values
// and a period shorthand. It enforces cross-field rules that cannot be expressed as
// single-field ozzo rules:
//   - from and to must parse as YYYY-MM-DD
//   - from must be strictly before to when both are present
//   - period is resolved only when neither from nor to is set
//   - an unrecognised period is always rejected, even when from/to are also set
func ParseDateWindow(rawFrom, rawTo, period string) (from, to *time.Time, err error) {
	if rawFrom != "" {
		t, parseErr := time.Parse("2006-01-02", rawFrom)
		if parseErr != nil {
			return nil, nil, fmt.Errorf("invalid from date: %q", rawFrom)
		}
		from = &t
	}
	if rawTo != "" {
		t, parseErr := time.Parse("2006-01-02", rawTo)
		if parseErr != nil {
			return nil, nil, fmt.Errorf("invalid to date: %q", rawTo)
		}
		to = &t
	}
	if from != nil && to != nil && !from.Before(*to) {
		return nil, nil, fmt.Errorf("from must be before to")
	}
	if period != "" && from == nil && to == nil {
		days, ok := periodDays[period]
		if !ok {
			return nil, nil, fmt.Errorf("unknown period: %q", period)
		}
		if days > 0 {
			now := time.Now().UTC()
			f := now.AddDate(0, 0, -days)
			from = &f
			to = &now
		}
	} else if period != "" && !isKnownPeriod(period) {
		return nil, nil, fmt.Errorf("unknown period: %q", period)
	}
	return from, to, nil
}

func isKnownPeriod(period string) bool {
	_, ok := periodDays[period]
	return ok
}
