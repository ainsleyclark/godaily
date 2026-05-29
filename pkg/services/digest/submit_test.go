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
	t.Parallel()

	start, end := collectWindow(time.Now())
	inWindow := start.Add(time.Hour)
	beforeWindow := start.Add(-time.Hour)
	afterWindow := end.Add(time.Hour)

	listWindow := func(t *testing.T, repo news.ItemRepository) []news.Item {
		t.Helper()
		got, err := repo.List(t.Context(), news.ItemListOptions{From: &start, To: &end})
		require.NoError(t, err)
		return got
	}

	t.Run("Persists In-Window Items In Score Order", func(t *testing.T) {
		t.Parallel()

		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "low", URL: "https://e.test/1", Tag: news.TagDiscussion, Score: 0.1, Published: inWindow},
			{Title: "high", URL: "https://e.test/2", Tag: news.TagDiscussion, Score: 0.9, Published: inWindow},
		})
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Received)
		assert.Equal(t, 2, resp.Persisted)
		assert.Zero(t, resp.Duplicates)

		got := listWindow(t, itemRepo)
		require.Len(t, got, 2)
		assert.Equal(t, "high", got[0].Title)
		assert.Equal(t, "low", got[1].Title)
		assert.Equal(t, news.SourceReddit, got[0].Source)
	})

	t.Run("Drops Before-Window And Zero-Date Items", func(t *testing.T) {
		t.Parallel()

		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "before", URL: "https://e.test/1", Tag: news.TagDiscussion, Published: beforeWindow},
			{Title: "zero", URL: "https://e.test/2", Tag: news.TagDiscussion},
			{Title: "in", URL: "https://e.test/3", Tag: news.TagDiscussion, Published: inWindow},
		})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Received)
		assert.Equal(t, 1, resp.Persisted)
		assert.Zero(t, resp.Duplicates)

		got := listWindow(t, itemRepo)
		require.Len(t, got, 1)
		assert.Equal(t, "in", got[0].Title)
	})

	t.Run("Clamps Future-Dated Items Into Window", func(t *testing.T) {
		t.Parallel()

		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "after", URL: "https://e.test/1", Tag: news.TagDiscussion, Published: afterWindow},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Persisted)

		got := listWindow(t, itemRepo)
		require.Len(t, got, 1)
		assert.Equal(t, start.Add(time.Hour), got[0].Published)
	})

	t.Run("De-duplicates Within The Payload", func(t *testing.T) {
		t.Parallel()

		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "dup a", URL: "https://e.test/same", Tag: news.TagDiscussion, Published: inWindow},
			{Title: "dup b", URL: "https://e.test/same", Tag: news.TagDiscussion, Published: inWindow},
		})
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Received)
		assert.Equal(t, 1, resp.Persisted)
		assert.Equal(t, 1, resp.Duplicates)

		assert.Len(t, listWindow(t, itemRepo), 1)
	})

	t.Run("Re-submitting Same Items Adds No Duplicates", func(t *testing.T) {
		t.Parallel()

		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		payload := []news.Item{
			{Title: "first", URL: "https://e.test/1", Tag: news.TagDiscussion, Published: inWindow},
		}

		first, err := svc.Submit(t.Context(), news.SourceReddit, payload)
		require.NoError(t, err)
		assert.Equal(t, 1, first.Persisted)

		second, err := svc.Submit(t.Context(), news.SourceReddit, payload)
		require.NoError(t, err)
		assert.Zero(t, second.Persisted)
		assert.Equal(t, 1, second.Duplicates)

		assert.Len(t, listWindow(t, itemRepo), 1, "re-submission must not create duplicates")
	})

	t.Run("Incremental Run Adds Only New Items", func(t *testing.T) {
		t.Parallel()

		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		_, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "existing", URL: "https://e.test/1", Tag: news.TagDiscussion, Published: inWindow},
		})
		require.NoError(t, err)

		// A later run contains the existing post plus a new one.
		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "existing", URL: "https://e.test/1", Tag: news.TagDiscussion, Published: inWindow},
			{Title: "fresh", URL: "https://e.test/2", Tag: news.TagDiscussion, Published: inWindow},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Persisted)
		assert.Equal(t, 1, resp.Duplicates)

		got := listWindow(t, itemRepo)
		assert.Len(t, got, 2)
	})

	t.Run("De-duplicates Against Other Sources On URL And Tag", func(t *testing.T) {
		t.Parallel()

		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		// Same (url, tag) already present from another source — matches the
		// items_url_tag_unique index, so it must not be inserted again.
		_, err := itemRepo.Create(t.Context(), nil, 1, news.Item{
			Source: news.SourceHN, Title: "hn", URL: "https://e.test/shared",
			Tag: news.TagDiscussion, Published: inWindow,
		})
		require.NoError(t, err)

		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "reddit dup", URL: "https://e.test/shared", Tag: news.TagDiscussion, Published: inWindow},
			{Title: "reddit new", URL: "https://e.test/new", Tag: news.TagDiscussion, Published: inWindow},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Persisted)
		assert.Equal(t, 1, resp.Duplicates)

		assert.Len(t, listWindow(t, itemRepo), 2)
	})

	t.Run("Same URL Different Tag Is Not A Duplicate", func(t *testing.T) {
		t.Parallel()

		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		_, err := itemRepo.Create(t.Context(), nil, 1, news.Item{
			Source: news.SourceHN, Title: "article", URL: "https://e.test/x",
			Tag: news.TagArticle, Published: inWindow,
		})
		require.NoError(t, err)

		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "discussion", URL: "https://e.test/x", Tag: news.TagDiscussion, Published: inWindow},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Persisted)
		assert.Zero(t, resp.Duplicates)
	})

	t.Run("No In-Window Items Persists Nothing", func(t *testing.T) {
		t.Parallel()

		_, itemRepo := newTestStores(t)
		svc := Service{items: itemRepo}

		resp, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "before", URL: "https://e.test/1", Tag: news.TagDiscussion, Published: beforeWindow},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Received)
		assert.Zero(t, resp.Persisted)
		assert.Zero(t, resp.Duplicates)
	})

	t.Run("Propagates List Error", func(t *testing.T) {
		t.Parallel()

		svc := Service{items: errItemRepo{err: errors.New("boom")}}
		_, err := svc.Submit(t.Context(), news.SourceReddit, []news.Item{
			{Title: "in", URL: "https://e.test/1", Tag: news.TagDiscussion, Published: inWindow},
		})
		assert.ErrorContains(t, err, "boom")
	})
}
