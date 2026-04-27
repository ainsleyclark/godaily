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

package cron

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/internal/email"
	"github.com/ainsleyclark/godaily/internal/news"
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

			got, err := New()
			test.want(t, got, err)
		})
	}
}

func TestAggregator_Run(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	inWindow := yesterday.Add(time.Hour)
	beforeWindow := yesterday.Add(-time.Hour)
	afterWindow := yesterday.Add(25 * time.Hour)

	tt := map[string]struct {
		registry map[news.Source]news.Fetcher
		opts     RunOptions
		emailErr error
		want     func(t *testing.T, items []news.SourceItems, m *mockEmail, err error)
	}{
		"DryRun Skips Email": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{
					items: []news.Item{{Title: "in", Published: inWindow}},
				},
			},
			opts: RunOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, items []news.SourceItems, m *mockEmail, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.Equal(t, news.SourceDevTo, items[0].Source)
				assert.Len(t, items[0].Items, 1)
				assert.False(t, m.called)
			},
		},
		"Sends Digest When Not DryRun": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{
					items: []news.Item{{Title: "in", Published: inWindow}},
				},
			},
			opts: RunOptions{Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, items []news.SourceItems, m *mockEmail, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.True(t, m.called)
				assert.Contains(t, m.req.Subject, "GoDaily")
			},
		},
		"Send Error Is Logged Not Returned": {
			registry: map[news.Source]news.Fetcher{
				news.SourceDevTo: mockFetcher{
					items: []news.Item{{Title: "in", Published: inWindow}},
				},
			},
			opts:     RunOptions{Sources: []news.Source{news.SourceDevTo}},
			emailErr: errors.New("send boom"),
			want: func(t *testing.T, items []news.SourceItems, m *mockEmail, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.True(t, m.called)
			},
		},
		"Default Sources When Empty": {
			registry: allRegistered(),
			opts:     RunOptions{DryRun: true},
			want: func(t *testing.T, items []news.SourceItems, _ *mockEmail, err error) {
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
			opts: RunOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, items []news.SourceItems, _ *mockEmail, err error) {
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
			opts: RunOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, items []news.SourceItems, _ *mockEmail, err error) {
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
			opts: RunOptions{DryRun: true, Sources: []news.Source{news.SourceDevTo}},
			want: func(t *testing.T, items []news.SourceItems, _ *mockEmail, err error) {
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
			opts: RunOptions{
				DryRun:  true,
				Sources: []news.Source{news.SourceMedium, news.SourceGoBlog, news.SourceReddit},
			},
			want: func(t *testing.T, items []news.SourceItems, _ *mockEmail, err error) {
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
			opts: RunOptions{
				DryRun:  true,
				Sources: []news.Source{news.SourceDevTo, news.SourceLobsters},
			},
			want: func(t *testing.T, items []news.SourceItems, _ *mockEmail, err error) {
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

			m := &mockEmail{err: test.emailErr}
			agg := Aggregator{email: m, sendToAddress: "to@example.com"}

			got, err := agg.Run(t.Context(), test.opts)
			test.want(t, got, m, err)
		})
	}
}

func TestAggregator_Run_Synth(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	inWindow := yesterday.Add(time.Hour)

	registry := map[news.Source]news.Fetcher{
		news.SourceDevTo: mockFetcher{
			items: []news.Item{{Title: "in", Published: inWindow}},
		},
	}

	tt := map[string]struct {
		opts      RunOptions
		suggester *mockSuggester
		want      func(t *testing.T, m *mockEmail, sg *mockSuggester)
	}{
		"Disabled By Default": {
			opts:      RunOptions{Sources: []news.Source{news.SourceDevTo}},
			suggester: &mockSuggester{resp: synth.Suggestion{Twitter: "t", LinkedIn: "l"}},
			want: func(t *testing.T, m *mockEmail, sg *mockSuggester) {
				t.Helper()
				assert.False(t, sg.called, "synth must not be called without IncludeSynth")
				assert.True(t, m.called)
			},
		},
		"Enabled Calls Suggester And Includes In Email": {
			opts: RunOptions{
				Sources:      []news.Source{news.SourceDevTo},
				IncludeSynth: true,
			},
			suggester: &mockSuggester{resp: synth.Suggestion{Twitter: "tweetytweet", LinkedIn: "linky"}},
			want: func(t *testing.T, m *mockEmail, sg *mockSuggester) {
				t.Helper()
				assert.True(t, sg.called)
				assert.True(t, m.called)
				assert.Contains(t, m.req.Html, "tweetytweet")
				assert.Contains(t, m.req.Text, "linky")
			},
		},
		"Suggester Error Logged Not Returned": {
			opts: RunOptions{
				Sources:      []news.Source{news.SourceDevTo},
				IncludeSynth: true,
			},
			suggester: &mockSuggester{err: errors.New("api boom")},
			want: func(t *testing.T, m *mockEmail, sg *mockSuggester) {
				t.Helper()
				assert.True(t, sg.called)
				assert.True(t, m.called, "digest still ships when synth fails")
				assert.NotContains(t, m.req.Html, "Suggested posts")
			},
		},
		"Suggester ErrNoItems Skipped Silently": {
			opts: RunOptions{
				Sources:      []news.Source{news.SourceDevTo},
				IncludeSynth: true,
			},
			suggester: &mockSuggester{err: synth.ErrNoItems},
			want: func(t *testing.T, m *mockEmail, sg *mockSuggester) {
				t.Helper()
				assert.True(t, sg.called)
				assert.True(t, m.called)
				assert.NotContains(t, m.req.Html, "Suggested posts")
			},
		},
		"DryRun Skips Suggester": {
			opts: RunOptions{
				Sources:      []news.Source{news.SourceDevTo},
				IncludeSynth: true,
				DryRun:       true,
			},
			suggester: &mockSuggester{resp: synth.Suggestion{Twitter: "t", LinkedIn: "l"}},
			want: func(t *testing.T, m *mockEmail, sg *mockSuggester) {
				t.Helper()
				assert.False(t, sg.called, "synth must not be called when DryRun is set")
				assert.False(t, m.called)
			},
		},
		"Nil Suggester Tolerated": {
			opts: RunOptions{
				Sources:      []news.Source{news.SourceDevTo},
				IncludeSynth: true,
			},
			suggester: nil,
			want: func(t *testing.T, m *mockEmail, _ *mockSuggester) {
				t.Helper()
				assert.True(t, m.called)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(news.SwapRegistry(registry))

			m := &mockEmail{}
			agg := Aggregator{email: m, sendToAddress: "to@example.com"}
			if test.suggester != nil {
				agg.suggester = test.suggester
			}

			_, err := agg.Run(t.Context(), test.opts)
			require.NoError(t, err)
			test.want(t, m, test.suggester)
		})
	}
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
