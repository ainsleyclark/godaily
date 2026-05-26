// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbtypes_test

import (
	"database/sql"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/store/internal/dbtypes"
	"github.com/stretchr/testify/assert"
)

func TestNullString(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want sql.NullString
	}{
		"Empty":     {in: "", want: sql.NullString{}},
		"Non-empty": {in: "hello", want: sql.NullString{String: "hello", Valid: true}},
		"Spaces":    {in: "  ", want: sql.NullString{String: "  ", Valid: true}},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, dbtypes.NullString(tc.in))
		})
	}
}
