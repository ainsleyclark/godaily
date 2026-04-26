//go:build integration

package source_test

import (
	"context"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSources_Integration(t *testing.T) {
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
