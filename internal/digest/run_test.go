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
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/internal/db"
	"github.com/ainsleyclark/godaily/internal/email"
	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/store/issues"
	"github.com/ainsleyclark/godaily/internal/store/items"
	"github.com/ainsleyclark/godaily/internal/synth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFetcher struct {
	items []news.Item
	err   error
}

func (m mockFetcher) Fetch(_ context.Context) ([]news.Item, error) {
	return m.items, m.err
}

type mockEmail struct {
	called bool
	req    email.SendEmailRequest
	err    error
}

func (m *mockEmail) Send(_ context.Context, req email.SendEmailRequest) error {
	m.called = true
	m.req = req
	return m.err
}

type mockSuggester struct {
	called bool
	resp   synth.Suggestion
	err    error
}

func (m *mockSuggester) Suggest(_ context.Context, _ time.Time, _ []news.SourceItems) (synth.Suggestion, error) {
	m.called = true
	return m.resp, m.err
}

// allRegistered returns a registry populated with mock fetchers for
// every source in news.Sources.
func allRegistered() map[news.Source]news.Fetcher {
	reg := map[news.Source]news.Fetcher{}
	for _, s := range news.Sources {
		reg[s] = mockFetcher{}
	}
	return reg
}

func TestNew(t *testing.T) {
	tt := map[string]struct {
		registry map[news.Source]news.Fetcher
		envAddr  string
		want     func(t *testing.T, agg *Aggregator, err error)
	}{
		"OK": {
			registry: allRegistered(),
			envAddr:  "to@example.com",
			want: func(t *testing.T, agg *Aggregator, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, agg)
				assert.Equal(t, "to@example.com", agg.sendToAddress)
				assert.NotNil(t, agg.email)
			},
		},
		"Missing Send Address": {
			registry: allRegistered(),
			envAddr:  "",
			want: func(t *testing.T, agg *Aggregator, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, agg)
				assert.Empty(t, agg.sendToAddress)
			},
		},
		"Validate Error": {
			registry: map[news.Source]news.Fetcher{},
			envAddr:  "to@example.com",
			want: func(t *testing.T, agg *Aggregator, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, agg)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(news.SwapRegistry(test.registry))
			t.Setenv("EMAIL_SEND_ADDRESS", test.envAddr)

			got, err := New(nil, nil)
			test.want(t, got, err)
		})
	}
}

func TestAggregator_Collect(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	inWindow := yesterday.Add(time.Hour)
	beforeWindow := yesterday.Add(-time.Hour)
	afterWindow := yesterday.Add(25 * time.Hour)

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
		"Returns Items When Not DryRun": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{
					items: []news.Item{{Title: "in", Published: inWindow}},
				},
			},
			opts: CollectOptions{Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, items []news.SourceItems, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 1)
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

func TestAggregator_Collect_NoSynth(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	inWindow := yesterday.Add(time.Hour)

	registry := map[news.Source]news.Fetcher{
		news.SourceDevTo: mockFetcher{
			items: []news.Item{{Title: "in", Published: inWindow}},
		},
	}

	t.Run("Suggester Is Never Called During Collect", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		sg := &mockSuggester{resp: synth.Suggestion{Post: "p"}}
		agg := Aggregator{suggester: sg}

		_, err := agg.Collect(t.Context(), CollectOptions{Sources: []news.Source{news.SourceDevTo}})
		require.NoError(t, err)
		assert.False(t, sg.called, "synth must not be called during Collect")
	})
}

func TestAggregator_Collect_Persistence(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	inWindow := yesterday.Add(time.Hour)

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

		stored, err := issueRepo.FindBySlug(t.Context(), yesterday.Format("2006-01-02"))
		require.NoError(t, err)
		assert.Equal(t, news.IssueStatusDraft, stored.Status)
		assert.NotEmpty(t, stored.HtmlBody)
		assert.NotEmpty(t, stored.TextBody)

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

	t.Run("Second Collect Same Day Is Skipped", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		issueRepo, itemRepo := newTestStores(t)
		agg := Aggregator{
			issues: issueRepo,
			items:  itemRepo,
		}

		opts := CollectOptions{Sources: []news.Source{news.SourceDevTo}}

		_, err := agg.Collect(t.Context(), opts)
		require.NoError(t, err)

		first, err := issueRepo.FindBySlug(t.Context(), yesterday.Format("2006-01-02"))
		require.NoError(t, err)
		require.NotZero(t, first.ID)

		_, err = agg.Collect(t.Context(), opts)
		require.NoError(t, err)

		second, err := issueRepo.FindBySlug(t.Context(), yesterday.Format("2006-01-02"))
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

func TestAggregator_Send(t *testing.T) {
	day := func(s string) time.Time {
		t.Helper()
		d, err := time.Parse("2006-01-02", s)
		require.NoError(t, err)
		return d
	}

	seedDraft := func(t *testing.T, repo *issues.Store, slug string) news.Issue {
		t.Helper()
		stored, err := repo.Create(t.Context(), news.Issue{
			Slug:     slug,
			Subject:  "GoDaily - " + slug,
			HtmlBody: "<p>base</p>",
			TextBody: "base",
			Status:   news.IssueStatusDraft,
			SentAt:   time.Now().UTC(),
		})
		require.NoError(t, err)
		return stored
	}

	seedItem := func(t *testing.T, repo *items.Store, issueID int64, source news.Source) {
		t.Helper()
		_, err := repo.Create(t.Context(), issueID, 1, news.Item{
			Source:    source,
			Title:     "item",
			URL:       "https://example.com/x",
			Published: time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(time.Hour),
		})
		require.NoError(t, err)
	}

	t.Run("Sends Email And Updates Status To Sent", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-04-26")
		stored := seedDraft(t, issueRepo, "2026-04-26")

		m := &mockEmail{}
		agg := Aggregator{email: m, sendToAddress: "to@example.com", issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.Send(t.Context(), date))

		assert.True(t, m.called)
		assert.Contains(t, m.req.Subject, "2026-04-26")

		updated, err := issueRepo.Find(t.Context(), stored.ID)
		require.NoError(t, err)
		assert.Equal(t, news.IssueStatusSent, updated.Status)
	})

	t.Run("Email Error Updates Status To Error", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-04-27")
		stored := seedDraft(t, issueRepo, "2026-04-27")

		m := &mockEmail{err: errors.New("send boom")}
		agg := Aggregator{email: m, sendToAddress: "to@example.com", issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.Send(t.Context(), date))

		updated, err := issueRepo.Find(t.Context(), stored.ID)
		require.NoError(t, err)
		assert.Equal(t, news.IssueStatusError, updated.Status)
	})

	t.Run("No Send Address Skips Email And Status Update", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-04-28")
		stored := seedDraft(t, issueRepo, "2026-04-28")

		m := &mockEmail{}
		agg := Aggregator{email: m, sendToAddress: "", issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.Send(t.Context(), date))
		assert.False(t, m.called)

		unchanged, err := issueRepo.Find(t.Context(), stored.ID)
		require.NoError(t, err)
		assert.Equal(t, news.IssueStatusDraft, unchanged.Status)
	})

	t.Run("Returns Error When Repos Are Nil", func(t *testing.T) {
		agg := Aggregator{email: &mockEmail{}, sendToAddress: "to@example.com"}
		err := agg.Send(t.Context(), day("2026-04-29"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "persistence")
	})

	t.Run("Returns Error When Issue Not Found", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		agg := Aggregator{email: &mockEmail{}, sendToAddress: "to@example.com", issues: issueRepo, items: itemRepo}

		err := agg.Send(t.Context(), day("1999-01-01"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no digest found")
	})

	t.Run("Returns Error When Status Not Draft", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		stored, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:     "2026-04-30",
			Subject:  "GoDaily - 2026-04-30",
			HtmlBody: "<p>x</p>",
			TextBody: "x",
			Status:   news.IssueStatusSent,
			SentAt:   time.Now().UTC(),
		})
		require.NoError(t, err)
		_ = stored

		agg := Aggregator{email: &mockEmail{}, sendToAddress: "to@example.com", issues: issueRepo, items: itemRepo}
		sendErr := agg.Send(t.Context(), day("2026-04-30"))
		require.Error(t, sendErr)
		assert.Contains(t, sendErr.Error(), "expected")
	})

	t.Run("Synth Called With Sections And Appended To Bodies", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-01")
		stored := seedDraft(t, issueRepo, "2026-05-01")
		seedItem(t, itemRepo, stored.ID, news.SourceDevTo)

		m := &mockEmail{}
		sg := &mockSuggester{resp: synth.Suggestion{Post: "punchy-post"}}
		agg := Aggregator{email: m, sendToAddress: "to@example.com", suggester: sg, issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.Send(t.Context(), date))

		assert.True(t, sg.called)
		assert.True(t, m.called)
		assert.Contains(t, m.req.Html, "punchy-post")
		assert.Contains(t, m.req.Text, "punchy-post")

		// Stored body must not be modified.
		reloaded, err := issueRepo.Find(t.Context(), stored.ID)
		require.NoError(t, err)
		assert.NotContains(t, reloaded.HtmlBody, "Suggested post")
	})

	t.Run("Synth Error Logged And Email Sent Without Suggestion", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-02")
		stored := seedDraft(t, issueRepo, "2026-05-02")
		seedItem(t, itemRepo, stored.ID, news.SourceDevTo)

		m := &mockEmail{}
		sg := &mockSuggester{err: errors.New("api boom")}
		agg := Aggregator{email: m, sendToAddress: "to@example.com", suggester: sg, issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.Send(t.Context(), date))

		assert.True(t, sg.called)
		assert.True(t, m.called, "email still sent when synth fails")
		assert.NotContains(t, m.req.Html, "Suggested post")
	})

	t.Run("No Items Skips Synth", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-03")
		seedDraft(t, issueRepo, "2026-05-03")

		m := &mockEmail{}
		sg := &mockSuggester{resp: synth.Suggestion{Post: "p"}}
		agg := Aggregator{email: m, sendToAddress: "to@example.com", suggester: sg, issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.Send(t.Context(), date))

		assert.False(t, sg.called)
		assert.True(t, m.called)
	})
}

func newTestStores(t *testing.T) (*issues.Store, *items.Store) {
	t.Helper()

	url := "file:" + filepath.Join(t.TempDir(), "godaily.db")
	conn, err := db.New(t.Context(), url, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, db.Up(t.Context(), conn))
	return issues.New(conn), items.New(conn)
}

func TestAggregator_FetchSource(t *testing.T) {
	tt := map[string]struct {
		registry map[news.Source]news.Fetcher
		source   news.Source
		want     func(t *testing.T, items []news.Item, err error)
	}{
		"Unregistered Source": {
			registry: map[news.Source]news.Fetcher{},
			source:   news.SourceDevTo,
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				assert.Nil(t, items)
				assert.ErrorContains(t, err, "getting fetcher")
			},
		},
		"Fetch Error": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{err: errors.New("boom")},
			},
			source: news.SourceDevTo,
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				assert.Nil(t, items)
				assert.ErrorContains(t, err, "fetching")
			},
		},
		"OK": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{
					items: []news.Item{{Title: "ok"}},
				},
			},
			source: news.SourceDevTo,
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Len(t, items, 1)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(news.SwapRegistry(test.registry))

			agg := Aggregator{}
			got, err := agg.fetchSource(t.Context(), test.source)
			test.want(t, got, err)
		})
	}
}
