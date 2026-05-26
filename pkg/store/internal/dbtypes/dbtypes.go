// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dbtypes provides helpers for converting Go values to database/sql types.
package dbtypes

import "database/sql"

// NullString converts s to a sql.NullString, treating empty string as NULL.
func NullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
