// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package store

import (
	"errors"
)

var (
	// ErrNotFound is returned by a Find function when the requested
	// item is not found.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists is returned by a create function when a
	// constraint is violated.
	ErrAlreadyExists = errors.New("item already exists")
)

const defaultPerPage int64 = 20

// ListOptions controls filtering and pagination for List queries.
// A zero value returns all results (no pagination).
type ListOptions struct {
	// Page is 1-based. Zero means no pagination.
	Page int64

	// PerPage is the number of items per page. Zero uses the default (20).
	PerPage int64
}

// Limit returns the SQL LIMIT value for this page.
// Returns a large sentinel (10000) when pagination is disabled.
func (o ListOptions) Limit() int64 {
	if o.Page == 0 {
		return 10000
	}
	if o.PerPage <= 0 {
		return defaultPerPage
	}
	return o.PerPage
}

// Offset returns the SQL OFFSET value for the current page.
func (o ListOptions) Offset() int64 {
	if o.Page <= 1 {
		return 0
	}
	return (o.Page - 1) * o.Limit()
}
