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
	htmltemplate "html/template"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/synth"
)

func TestAggregator_Collect(t *testing.T) {
	day, next := collectWindow(time.Now())
	inWindow := day.Add(time.Hour)
	beforeWindow := day.Add(-time.Hour)
	afterWindow := next.Add(time.Hour)

	tt := map[string]struct {
		registry map[news.Source]news.Fetcher
		opts     CollectOptions
		want     func(t *testing.T, items []news.SourceItems, err error)
	}{
		"DryRun Returns Items Without Persisting": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{
					items: []news.Item{{Title: "in", Published: inWindow}},
				},
			},
			opts: CollectOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, items []news.SourceItems, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.Equal(t, news.SourceDevTo, items[0].Source)
				assert.Len(t, items[0].Items, 1)
			},
		},
		"Default Sources When Empty": {
			registry: allRegistered(),
			opts:     CollectOptions{DryRun: true},
			want: func(t *testing.T, items []news.SourceItems, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, items)
			},
		},
		"Filters Zero Published Items": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{
					items: []news.Item{{Title: "zero"}},
				},
			},
			opts: CollectOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, items []news.SourceItems, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, items)
			},
		},
		"Filters Out Of Window Items": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{
					items: []news.Item{
						{Title: "before", Published: beforeWindow},
						{Title: "after", Published: afterWindow},
					},
				},
			},
			opts: CollectOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, items []news.SourceItems, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, items)
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
			opts: CollectOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, items []news.SourceItems, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 1)
				require.Len(t, items[0].Items, 3)
				assert.Equal(t, "high", items[0].Items[0].Title)
				assert.Equal(t, "mid", items[0].Items[1].Title)
				assert.Equal(t, "low", items[0].Items[2].Title)
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
			opts: CollectOptions{
				DryRun:  true,
				Sources: []news.Source{news.SourceMedium, news.SourceGoBlog, news.SourceReddit},
			},
			want: func(t *testing.T, items []news.SourceItems, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 3)
				assert.Equal(t, news.SourceGoBlog, items[0].Source)
				assert.Equal(t, news.SourceReddit, items[1].Source)
				assert.Equal(t, news.SourceMedium, items[2].Source)
			},
		},
		"Continues On Fetch Error": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{err: errors.New("boom")},
				news.SourceLobsters: mockFetcher{
					items: []news.Item{{Title: "ok", Published: inWindow}},
				},
			},
			opts: CollectOptions{
				DryRun:  true,
				Sources: []news.Source{news.SourceDevTo, news.SourceLobsters},
			},
			want: func(t *testing.T, items []news.SourceItems, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.Equal(t, news.SourceLobsters, items[0].Source)
			},
		},
		"Empty Results No Persist": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{items: []news.Item{}},
			},
			opts: CollectOptions{Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, items []news.SourceItems, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, items)
			},
		},
	}

	// Note: "Render Failure Falls Back Gracefully" is tested separately below
	// because it mutates a package-level template var and cannot run in parallel.

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(news.SwapRegistry(test.registry))

			agg := Aggregator{}
			got, err := agg.Collect(t.Context(), test.opts)
			test.want(t, got, err)
		})
	}
}

func TestAggregator_Collect_RenderFallback(t *testing.T) {
	day, _ := collectWindow(time.Now())
	inWindow := day.Add(time.Hour)

	registry := map[news.Source]news.Fetcher{
		news.SourceDevTo: mockFetcher{
			items: []news.Item{{Title: "in", Published: inWindow}},
		},
	}

	// When renderDigest fails (broken template), Collect logs and returns the
	// raw results without persisting rather than surfacing an error.
	t.Run("Render Failure Falls Back Gracefully", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		orig := htmlTmpl
		htmlTmpl = htmltemplate.Must(htmltemplate.New("digest").Parse(`{{ .Missing.NotAField }}`))
		t.Cleanup(func() { htmlTmpl = orig })

		issueRepo, itemRepo := newTestStores(t)
		agg := Aggregator{issues: issueRepo, items: itemRepo}

		got, err := agg.Collect(t.Context(), CollectOptions{Sources: []news.Source{news.SourceDevTo}})
		require.NoError(t, err)
		require.Len(t, got, 1, "raw items still returned despite render failure")

		count, err := issueRepo.Count(t.Context())
		require.NoError(t, err)
		assert.Equal(t, int64(0), count, "nothing persisted when render fails")
	})
}

func TestAggregator_Collect_Synthesiser(t *testing.T) {
	day, _ := collectWindow(time.Now())
	inWindow := day.Add(time.Hour)

	registry := map[news.Source]news.Fetcher{
		news.SourceDevTo: mockFetcher{
			items: []news.Item{{Title: "in", Published: inWindow}},
		},
	}

	t.Run("Suggester Is Never Called During Collect", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		sg := &mockSuggester{resp: synth.Suggestion{Post: "p"}}
		agg := Aggregator{suggester: sg}

		_, err := agg.Collect(t.Context(), CollectOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}})
		require.NoError(t, err)
		assert.False(t, sg.called, "suggester must not be called during Collect")
	})

	t.Run("DryRun Does Not Call Synthesiser", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		syn := &mockSynthesiser{resp: synth.DigestMeta{Title: "t", Intro: "i"}}
		agg := Aggregator{synthesiser: syn}

		_, err := agg.Collect(t.Context(), CollectOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}})
		require.NoError(t, err)
		assert.False(t, syn.called, "synthesiser must not be called during a dry run")
	})

	t.Run("Synthesiser Populates Subject And Summary On Persist", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		syn := &mockSynthesiser{resp: synth.DigestMeta{Title: "Go 1.24 lands", Intro: "Goroutines got faster."}}
		issueRepo, itemRepo := newTestStores(t)
		agg := Aggregator{synthesiser: syn, issues: issueRepo, items: itemRepo}

		_, err := agg.Collect(t.Context(), CollectOptions{Sources: []news.Source{news.SourceDevTo}})
		require.NoError(t, err)

		stored, err := issueRepo.FindBySlug(t.Context(), day.Format("2006-01-02"))
		require.NoError(t, err)
		assert.True(t, syn.called)
		assert.Equal(t, "Go 1.24 lands", stored.Subject)
		assert.Equal(t, "Goroutines got faster.", stored.Summary)
	})

	t.Run("Synthesiser Error Falls Back To Static Subject", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		syn := &mockSynthesiser{err: errors.New("boom")}
		issueRepo, itemRepo := newTestStores(t)
		agg := Aggregator{synthesiser: syn, issues: issueRepo, items: itemRepo}

		_, err := agg.Collect(t.Context(), CollectOptions{Sources: []news.Source{news.SourceDevTo}})
		require.NoError(t, err)

		stored, err := issueRepo.FindBySlug(t.Context(), day.Format("2006-01-02"))
		require.NoError(t, err)
		assert.Equal(t, "GoDaily - "+day.Format("January 2, 2006"), stored.Subject)
		assert.Empty(t, stored.Summary)
	})

	t.Run("Nil Synthesiser Falls Back To Static Subject", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		issueRepo, itemRepo := newTestStores(t)
		agg := Aggregator{synthesiser: nil, issues: issueRepo, items: itemRepo}

		_, err := agg.Collect(t.Context(), CollectOptions{Sources: []news.Source{news.SourceDevTo}})
		require.NoError(t, err)

		stored, err := issueRepo.FindBySlug(t.Context(), day.Format("2006-01-02"))
		require.NoError(t, err)
		assert.Equal(t, "GoDaily - "+day.Format("January 2, 2006"), stored.Subject)
		assert.Empty(t, stored.Summary)
	})
}

func TestAggregator_Collect_Persistence(t *testing.T) {
	day, _ := collectWindow(time.Now())
	inWindow := day.Add(time.Hour)

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

	t.Run("Persists Issue As Draft With Items", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		issueRepo, itemRepo := newTestStores(t)
		agg := Aggregator{
			issues: issueRepo,
			items:  itemRepo,
		}

		_, err := agg.Collect(t.Context(), CollectOptions{Sources: []news.Source{news.SourceDevTo}})
		require.NoError(t, err)

		stored, err := issueRepo.FindBySlug(t.Context(), day.Format("2006-01-02"))
		require.NoError(t, err)
		assert.Equal(t, news.IssueStatusDraft, stored.Status)

		got, err := itemRepo.ListByIssue(t.Context(), stored.ID)
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

	t.Run("Second Collect Same Day Skips Without Creating Duplicate", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		issueRepo, itemRepo := newTestStores(t)
		agg := Aggregator{
			issues: issueRepo,
			items:  itemRepo,
		}

		opts := CollectOptions{Sources: []news.Source{news.SourceDevTo}}

		_, err := agg.Collect(t.Context(), opts)
		require.NoError(t, err)

		first, err := issueRepo.FindBySlug(t.Context(), day.Format("2006-01-02"))
		require.NoError(t, err)
		require.NotZero(t, first.ID)

		// Second collect on the same day logs a warning and returns nil;
		// the existing issue must not be duplicated.
		_, err = agg.Collect(t.Context(), opts)
		require.NoError(t, err)

		second, err := issueRepo.FindBySlug(t.Context(), day.Format("2006-01-02"))
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID, "second collect must not create a duplicate issue")
	})

	t.Run("DryRun Does Not Persist", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		issueRepo, itemRepo := newTestStores(t)
		agg := Aggregator{
			issues: issueRepo,
			items:  itemRepo,
		}

		_, err := agg.Collect(t.Context(), CollectOptions{
			Sources: []news.Source{news.SourceDevTo},
			DryRun:  true,
		})
		require.NoError(t, err)

		count, err := issueRepo.Count(t.Context())
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}
