// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mcpserver

import (
	"time"

	"github.com/ainsleyclark/godaily/pkg/api"
)

const (
	// defaultLimit is the row count returned by ranked metrics tools when
	// the caller does not ask for a specific number.
	defaultLimit = 10

	// maxLimit caps every limit/per_page argument to keep responses within a
	// sensible size for a model's context window.
	maxLimit = 100
)

// Window is the date window shared by every tool that aggregates over time.
// It is embedded in tool inputs so the JSON schema exposes period, from and
// to as top-level arguments.
type Window struct {
	Period string `json:"period,omitempty" jsonschema:"Relative window ending now. Ignored when from or to is set. Omit period, from and to for all time."`
	From   string `json:"from,omitempty"   jsonschema:"Start date in YYYY-MM-DD format."`
	To     string `json:"to,omitempty"     jsonschema:"End date in YYYY-MM-DD format. Must be after from."`
}

// bounds resolves the window to optional time bounds using the same rules
// as the metrics HTTP API.
func (w Window) bounds() (from, to *time.Time, err error) {
	return api.ParseDateWindow(w.From, w.To, w.Period)
}

// windowEnums is the enum set for any input that embeds Window.
var windowEnums = map[string][]string{
	"period": api.Periods,
}

// withEnums merges extra enum constraints into a copy of base.
func withEnums(base map[string][]string, extra map[string][]string) map[string][]string {
	out := make(map[string][]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// clampLimit normalises a caller-supplied limit: zero or negative values fall
// back to def, and anything above maxLimit is capped.
func clampLimit(n, def int) int {
	if n <= 0 {
		return def
	}
	return min(n, maxLimit)
}

// pagination normalises page and per_page arguments using the API defaults.
func pagination(page, perPage int64) (int64, int64) {
	if page < 1 {
		page = api.DefaultPage
	}
	if perPage < 1 {
		perPage = api.DefaultPerPage
	}
	return page, min(perPage, api.MaxPerPage)
}
