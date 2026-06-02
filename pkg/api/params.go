// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"net/http"
	"strconv"
)

// ParseID parses a resource path parameter as a positive int64 identifier,
// returning (id, true) on success. It rejects empty, non-numeric, zero and
// negative values so handlers can reply with a single "must be a positive
// integer" 400 rather than repeating the strconv dance.
func ParseID(raw string) (int64, bool) {
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

// QueryInt returns the named query parameter as int64, or fallback if the
// parameter is absent or cannot be parsed.
func QueryInt(r *http.Request, key string, fallback int64) int64 {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return v
}
