// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTop_HasItems(t *testing.T) {
	t.Parallel()

	t.Run("True", func(t *testing.T) {
		t.Parallel()
		in := Top{Items: []RankedItem{
			{},
		}}
		got := in.HasItems()
		assert.True(t, got)
	})

	t.Run("False", func(t *testing.T) {
		t.Parallel()
		in := Top{Items: nil}
		got := in.HasItems()
		assert.False(t, got)
	})
}
