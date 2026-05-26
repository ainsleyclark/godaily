// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

const (
	// DefaultPage is the default page number for paginated responses.
	DefaultPage int64 = 1

	// DefaultPerPage is the default number of items per page.
	DefaultPerPage int64 = 20

	// MaxPerPage is the maximum number of items allowed per page.
	MaxPerPage int64 = 100
)

// PaginatedResponse is the JSON envelope returned by paginated list endpoints.
type PaginatedResponse[T any] struct {
	Data    []T   `json:"data"`
	Page    int64 `json:"page"`
	PerPage int64 `json:"per_page"`
	Total   int64 `json:"total"`
}
