// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"fmt"
	"time"
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

// Periods lists the period shorthands accepted by ParseDateWindow, in
// ascending order of span.
var Periods = []string{"day", "week", "month", "year", "all"}

// ParseDateWindow resolves optional from/to time bounds from raw YYYY-MM-DD
// values and a period shorthand. It is shared by the metrics HTTP handlers
// and the MCP tools so both surfaces interpret windows identically.
//
// It enforces cross-field rules that cannot be expressed as single-field
// validation rules:
//   - from and to must parse as YYYY-MM-DD
//   - from must be strictly before to when both are present
//   - period is resolved only when neither from nor to is set
//   - an unrecognised period is always rejected, even when from/to are also set
func ParseDateWindow(rawFrom, rawTo, period string) (from, to *time.Time, err error) {
	if rawFrom != "" {
		t, parseErr := time.Parse(time.DateOnly, rawFrom)
		if parseErr != nil {
			return nil, nil, fmt.Errorf("invalid from date: %q", rawFrom)
		}
		from = &t
	}
	if rawTo != "" {
		t, parseErr := time.Parse(time.DateOnly, rawTo)
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
