// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"testing"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisteredSources(t *testing.T) {
	t.Parallel()

	for _, s := range news.Sources {
		t.Run(string(s), func(t *testing.T) {
			t.Parallel()

			got, err := news.Get(s)
			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}
