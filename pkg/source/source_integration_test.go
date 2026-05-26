// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build integration

package source_test

import (
	"context"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSources_Integration(t *testing.T) {
	cfg, err := env.New(t.Context())
	require.NoError(t, err)
	require.NoError(t, news.Materialise(cfg))

	for _, source := range news.Sources {
		t.Run(source.String(), func(t *testing.T) {
			t.Parallel()

			fetcher, err := news.Get(source)
			if err != nil {
				t.Skipf("no fetcher registered for %s", source)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			items, err := fetcher.Fetch(ctx)

			require.NoError(t, err)
			require.NotEmpty(t, items)
			assert.NotEmpty(t, items[0].Title)
			assert.NotEmpty(t, items[0].URL)
		})
	}
}
