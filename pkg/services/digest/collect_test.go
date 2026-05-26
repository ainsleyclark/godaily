// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package digest

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	digest "github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

func TestAggregator_Collect(t *testing.T) {
	start, end := collectWindow(time.Now())
	inWindow := start.Add(time.Hour)
	beforeWindow := start.Add(-time.Hour)
	afterWindow := end.Add(time.Hour)

	tt := map[string]struct {
		registry map[news.Source]news.Fetcher
		opts     digest.CollectOptions
		want     func(t *testing.T, resp digest.CollectResponse, err error)
	}{
		"DryRun Returns Items Without Persisting": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{
					items: []news.Item{{Title: "in", Published: inWindow}},
				},
			},
			opts: digest.CollectOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, resp digest.CollectResponse, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, resp.Sources, 1)
				assert.Equal(t, news.SourceDevTo, resp.Sources[0].Source)
				assert.Len(t, resp.Sources[0].Items, 1)
			},
		},
		"Default Sources When Empty": {
			registry: allRegistered(),
			opts:     digest.CollectOptions{DryRun: true},
			want: func(t *testing.T, resp digest.CollectResponse, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, resp.Sources)
			},
		},
		"Filters Zero Published Items": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{
					items: []news.Item{{Title: "zero"}},
				},
			},
			opts: digest.CollectOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, resp digest.CollectResponse, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, resp.Sources)
			},
		},
		"Filters Before-Window Items": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{
					items: []news.Item{
						{Title: "before", Published: beforeWindow},
					},
				},
			},
			opts: digest.CollectOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, resp digest.CollectResponse, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, resp.Sources)
			},
		},
		"Clamps Future-Published Items Into Window": {
			// Sources like meetup set Published: time.Now(), which lands after
			// the window's end (today midnight). The pipeline clamps these to
			// start+1h so they are stored in the correct window without the
			// source needing to know the pipeline's date expectations.
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{
					items: []news.Item{
						{Title: "after", Published: afterWindow},
					},
				},
			},
			opts: digest.CollectOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, resp digest.CollectResponse, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, resp.Sources, 1)
				require.Len(t, resp.Sources[0].Items, 1)
				assert.Equal(t, "after", resp.Sources[0].Items[0].Title)
				assert.Equal(t, start.Add(time.Hour), resp.Sources[0].Items[0].Published)
			},
		},
		"Sorts Items By Score Desc": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{
					items: []news.Item{
						{Title: "low", Published: inWindow, Score: 0.1},
						{Title: "high", Published: inWindow, Score: 0.9},
						{Title: "mid", Published: inWindow, Score: 0.5},
					},
				},
			},
			opts: digest.CollectOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, resp digest.CollectResponse, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, resp.Sources, 1)
				require.Len(t, resp.Sources[0].Items, 3)
				assert.Equal(t, "high", resp.Sources[0].Items[0].Title)
				assert.Equal(t, "mid", resp.Sources[0].Items[1].Title)
				assert.Equal(t, "low", resp.Sources[0].Items[2].Title)
			},
		},
		"Sorts Sources By Priority": {
			registry: map[news.Source]news.Fetcher{
				news.SourceMedium: mockFetcher{
					items: []news.Item{{Title: "m", Published: inWindow}},
				},
				news.SourceGoBlog: mockFetcher{
					items: []news.Item{{Title: "g", Published: inWindow}},
				},
				news.SourceReddit: mockFetcher{
					items: []news.Item{{Title: "r", Published: inWindow}},
				},
			},
			opts: digest.CollectOptions{
				DryRun:  true,
				Sources: []news.Source{news.SourceMedium, news.SourceGoBlog, news.SourceReddit},
			},
			want: func(t *testing.T, resp digest.CollectResponse, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, resp.Sources, 3)
				assert.Equal(t, news.SourceGoBlog, resp.Sources[0].Source)
				assert.Equal(t, news.SourceReddit, resp.Sources[1].Source)
				assert.Equal(t, news.SourceMedium, resp.Sources[2].Source)
			},
		},
		"Continues On Fetch Error": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{err: errors.New("boom")},
				news.SourceLobsters: mockFetcher{
					items: []news.Item{{Title: "ok", Published: inWindow}},
				},
			},
			opts: digest.CollectOptions{
				DryRun:  true,
				Sources: []news.Source{news.SourceDevTo, news.SourceLobsters},
			},
			want: func(t *testing.T, resp digest.CollectResponse, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, resp.Sources, 1)
				assert.Equal(t, news.SourceLobsters, resp.Sources[0].Source)
				assert.ErrorContains(t, resp.Errors[news.SourceDevTo], "boom")
			},
		},
		"Empty Results No Persist": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{items: []news.Item{}},
			},
			opts: digest.CollectOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, resp digest.CollectResponse, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, resp.Sources)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(news.SwapRegistry(test.registry))

			agg := Aggregator{}
			got, err := agg.Collect(t.Context(), test.opts)
			test.want(t, got, err)
		})
	}
}

func TestAggregator_Collect_Persistence(t *testing.T) {
	start, _ := collectWindow(time.Now())
	inWindow := start.Add(time.Hour)

	registry := map[news.Source]news.Fetcher{
		news.SourceDevTo: mockFetcher{
			items: []news.Item{
				{
					Title: "first",
					URL:   "https://example.com/1",
					Author: &news.Author{
						Name:       "Ada Lovelace",
						Username:   "ada",
						AvatarURL:  "https://example.com/ada.png",
						ProfileURL: "https://dev.to/ada",
					},
					Score:     0.5,
					Published: inWindow,
				},
				{
					Title:     "second",
					URL:       "https://example.com/2",
					Score:     0.9,
					Published: inWindow,
				},
			},
		},
	}

	t.Run("Persists Items Without Issue", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		_, itemRepo := newTestStores(t)
		agg := Aggregator{
			items: itemRepo,
		}

		_, err := agg.Collect(t.Context(), digest.CollectOptions{Sources: []news.Source{news.SourceDevTo}})
		require.NoError(t, err)

		collStart, collEnd := collectWindow(time.Now())
		got, err := itemRepo.List(t.Context(), news.ItemListOptions{From: &collStart, To: &collEnd})
		require.NoError(t, err)
		require.Len(t, got, 2)
		// Items are persisted in score-descending order, so "second" (0.9) comes first.
		assert.Equal(t, "second", got[0].Title)
		assert.Equal(t, "first", got[1].Title)

		assert.Nil(t, got[0].Author)
		require.NotNil(t, got[1].Author)
		assert.Equal(t, "Ada Lovelace", got[1].Author.Name)
		assert.Equal(t, "ada", got[1].Author.Username)
		assert.Equal(t, "https://example.com/ada.png", got[1].Author.AvatarURL)
		assert.Equal(t, "https://dev.to/ada", got[1].Author.ProfileURL)
	})

	t.Run("Second Collect Same Day Skips Without Creating Duplicates", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		_, itemRepo := newTestStores(t)
		agg := Aggregator{
			items: itemRepo,
		}

		opts := digest.CollectOptions{Sources: []news.Source{news.SourceDevTo}}

		_, err := agg.Collect(t.Context(), opts)
		require.NoError(t, err)

		collStart, collEnd := collectWindow(time.Now())
		first, err := itemRepo.List(t.Context(), news.ItemListOptions{From: &collStart, To: &collEnd})
		require.NoError(t, err)
		require.Len(t, first, 2)

		// Second collect on the same day returns empty (idempotent).
		result, err := agg.Collect(t.Context(), opts)
		require.NoError(t, err)
		assert.Empty(t, result.Sources, "second collect must return no sources when items already exist")
		assert.Empty(t, result.Errors)

		second, err := itemRepo.List(t.Context(), news.ItemListOptions{From: &collStart, To: &collEnd})
		require.NoError(t, err)
		assert.Len(t, second, 2, "second collect must not create duplicate items")
	})

	t.Run("DryRun Does Not Persist", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		_, itemRepo := newTestStores(t)
		agg := Aggregator{
			items: itemRepo,
		}

		_, err := agg.Collect(t.Context(), digest.CollectOptions{
			Sources: []news.Source{news.SourceDevTo},
			DryRun:  true,
		})
		require.NoError(t, err)

		collStart, collEnd := collectWindow(time.Now())
		got, err := itemRepo.List(t.Context(), news.ItemListOptions{From: &collStart, To: &collEnd})
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestCollectWindow(t *testing.T) {
	tuesday := time.Date(2026, 5, 19, 1, 0, 0, 0, time.UTC)

	t.Run("Window always covers yesterday to today", func(t *testing.T) {
		start, end := collectWindow(tuesday)
		assert.Equal(t, "2026-05-18", start.Format("2006-01-02"), "window start should be yesterday")
		assert.Equal(t, "2026-05-19", end.Format("2006-01-02"), "window end should be today")
	})

	t.Run("Monday also covers only yesterday", func(t *testing.T) {
		monday := time.Date(2026, 5, 18, 1, 0, 0, 0, time.UTC)
		start, end := collectWindow(monday)
		assert.Equal(t, "2026-05-17", start.Format("2006-01-02"), "window start should be Sunday")
		assert.Equal(t, "2026-05-18", end.Format("2006-01-02"), "window end should be today (Monday)")
	})
}
