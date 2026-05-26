// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIssueStatus_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "draft", IssueStatusDraft.String())
}
