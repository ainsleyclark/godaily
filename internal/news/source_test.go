// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package news

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSource_String(t *testing.T) {
	input := SourceDevTo
	got := input.String()
	assert.IsType(t, got, "devto")
}
