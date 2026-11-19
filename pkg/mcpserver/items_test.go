// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mcpserver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

func TestGetItem(t *testing.T) {
	t.Parallel()

	t.Run("Found", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		item := news.Item{
			ID:          7,
			Source:      news.SourceHN,
			Title:       "Go 1.27 freeze",
			URL:         "https://go.dev/blog",
			OriginalURL: "https://news.ycombinator.com/item?id=1",
			Author:      &news.Author{Name: "gopher"},
			Snippet:     "The freeze begins.",
			Tag:         "official",
			Comments:    12,
			Score:       0.87,
			Published:   sentAt,
			InDigest:    true,
		}
		h.Items.EXPECT().Find(gomock.Any(), int64(7)).Return(item, nil)

		out := decode[news.Item](t, h, "get_item", map[string]any{"id": 7})
		assert.Equal(t, item, out)
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		h := setup(t)
		h.Items.EXPECT().Find(gomock.Any(), int64(99)).Return(news.Item{}, store.ErrNotFound)

		assert.Contains(t, toolError(t, h, "get_item", map[string]any{"id": 99}), "item 99 not found")
	})

	t.Run("Non-positive ID", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		assert.Contains(t, toolError(t, h, "get_item", map[string]any{"id": 0}), "positive integer")
	})
}

func TestSearchItems(t *testing.T) {
	t.Parallel()

	t.Run("Maps every filter", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		check := func(o news.ItemListOptions) {
			assert.Equal(t, "generics", o.Search)
			assert.Equal(t, []news.Source{news.SourceHN, news.SourceReddit}, o.Sources)
			assert.Equal(t, []news.Tag{"trending"}, o.Tags)
			assert.Equal(t, news.ItemSortTop, o.Sort)
			require.NotNil(t, o.InDigest)
			assert.True(t, *o.InDigest)
			require.NotNil(t, o.From)
			require.NotNil(t, o.To)
			assert.Equal(t, "2026-11-01", o.From.Format(time.DateOnly))
			assert.Equal(t, "2026-11-15", o.To.Format(time.DateOnly))
			assert.Equal(t, int64(2), o.Page)
			assert.Equal(t, int64(5), o.PerPage)
		}
		h.Items.EXPECT().CountMatching(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, o news.ItemListOptions) (int64, error) {
			check(o)
			return 6, nil
		})
		h.Items.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, o news.ItemListOptions) ([]news.Item, error) {
			check(o)
			return []news.Item{{ID: 1, Title: "Generics tips", Source: news.SourceHN}}, nil
		})

		out := decode[SearchItemsOutput](t, h, "search_items", map[string]any{
			"query":     "generics",
			"sources":   []string{"hacker_news", "reddit"},
			"tags":      []string{"trending"},
			"sort":      "top",
			"in_digest": true,
			"from":      "2026-11-01",
			"to":        "2026-11-15",
			"page":      2,
			"per_page":  5,
		})
		assert.Equal(t, int64(6), out.Total)
		assert.Equal(t, int64(2), out.Page)
		require.Len(t, out.Items, 1)
	})

	t.Run("Period resolves to a rolling window", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		h.Items.EXPECT().CountMatching(gomock.Any(), gomock.Any()).Return(int64(0), nil)
		h.Items.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, o news.ItemListOptions) ([]news.Item, error) {
			require.NotNil(t, o.From)
			require.NotNil(t, o.To)
			assert.InDelta(t, 7*24*time.Hour, o.To.Sub(*o.From), float64(time.Minute))
			assert.Nil(t, o.InDigest)
			assert.Equal(t, int64(1), o.Page)
			assert.Equal(t, int64(20), o.PerPage)
			return nil, nil
		})

		out := decode[SearchItemsOutput](t, h, "search_items", map[string]any{"period": "week"})
		assert.Empty(t, out.Items)
	})

	t.Run("Invalid date is a tool error", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		assert.Contains(t, toolError(t, h, "search_items", map[string]any{"from": "last tuesday"}), "invalid from date")
	})

	t.Run("Inverted window is a tool error", func(t *testing.T) {
		t.Parallel()
		h := setup(t)

		assert.Contains(t, toolError(t, h, "search_items", map[string]any{"from": "2026-11-10", "to": "2026-11-01"}), "from must be before to")
	})
}

func TestListItemFacets(t *testing.T) {
	t.Parallel()

	h := setup(t)
	h.Items.EXPECT().Count(gomock.Any()).Return(int64(120), nil)
	h.Items.EXPECT().SourceCounts(gomock.Any()).Return([]news.SourceCount{{Source: news.SourceHN, Count: 80}, {Source: news.SourceReddit, Count: 40}}, nil)
	h.Items.EXPECT().TagCounts(gomock.Any()).Return([]news.TagCount{{Tag: "trending", Count: 12}}, nil)

	out := decode[ItemFacetsOutput](t, h, "list_item_facets", map[string]any{})
	assert.Equal(t, int64(120), out.TotalItems)
	assert.Len(t, out.Sources, 2)
	assert.Equal(t, news.Tag("trending"), out.Tags[0].Tag)
}
