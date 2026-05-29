// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

func TestService_Submit(t *testing.T) {
	start, end := collectWindow(time.Now())
	inWindow := start.Add(time.Hour)
	beforeWindow := start.Add(-time.Hour)
	afterWindow := end.Add(time.Hour)

	t.Run("Persists In-Window Items In Score Order", func(t *testing.T) {
		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "low", URL: "https://e.test/1", Tag: news.TagDiscussion, Score: 0.1, Published: inWindow},
			{Title: "high", URL: "https://e.test/2", Tag: news.TagDiscussion, Score: 0.9, Published: inWindow},
		})
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Received)
		assert.Equal(t, 2, resp.Persisted)
		assert.False(t, resp.Skipped)

		got, err := itemRepo.List(t.Context(), news.ItemListOptions{From: &start, To: &end})
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "high", got[0].Title)
		assert.Equal(t, "low", got[1].Title)
		assert.Equal(t, news.SourceReddit, got[0].Source)
	})

	t.Run("Drops Before-Window And Zero-Date Items", func(t *testing.T) {
		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "before", URL: "https://e.test/1", Published: beforeWindow},
			{Title: "zero", URL: "https://e.test/2"},
			{Title: "in", URL: "https://e.test/3", Published: inWindow},
		})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Received)
		assert.Equal(t, 1, resp.Persisted)

		got, err := itemRepo.List(t.Context(), news.ItemListOptions{From: &start, To: &end})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "in", got[0].Title)
	})

	t.Run("Clamps Future-Dated Items Into Window", func(t *testing.T) {
		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "after", URL: "https://e.test/1", Published: afterWindow},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Persisted)

		got, err := itemRepo.List(t.Context(), news.ItemListOptions{From: &start, To: &end})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, start.Add(time.Hour), got[0].Published)
	})

	t.Run("Skips When Source Already Has Items For Window", func(t *testing.T) {
		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		_, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "first", URL: "https://e.test/1", Published: inWindow},
		})
		require.NoError(t, err)

		// Re-submitting for the same source/window is a no-op.
		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "second", URL: "https://e.test/2", Published: inWindow},
		})
		require.NoError(t, err)
		assert.True(t, resp.Skipped)
		assert.Zero(t, resp.Persisted)

		got, err := itemRepo.List(t.Context(), news.ItemListOptions{From: &start, To: &end})
		require.NoError(t, err)
		assert.Len(t, got, 1, "skipped submission must not add items")
	})

	t.Run("Other Source In Window Does Not Block Submission", func(t *testing.T) {
		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		// A different source already collected for the window (the common case:
		// the rest of the run succeeded, only Reddit was blocked).
		_, err := itemRepo.Create(t.Context(), nil, 1, news.Item{
			Source: news.SourceHN, Title: "hn", URL: "https://e.test/hn", Published: inWindow,
		})
		require.NoError(t, err)

		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "reddit", URL: "https://e.test/r", Published: inWindow},
		})
		require.NoError(t, err)
		assert.False(t, resp.Skipped)
		assert.Equal(t, 1, resp.Persisted)
	})

	t.Run("No In-Window Items Persists Nothing", func(t *testing.T) {
		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "before", URL: "https://e.test/1", Published: beforeWindow},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Received)
		assert.Zero(t, resp.Persisted)
		assert.False(t, resp.Skipped)
	})

	t.Run("Propagates List Error", func(t *testing.T) {
		svc := Service{items: errItemRepo{err: errors.New("boom")}}
		_, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "in", Published: inWindow},
		})
		assert.ErrorContains(t, err, "boom")
	})
}
