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

func TestAggregator_fetchSource(t *testing.T) {
	tt := map[string]struct {
		registry map[news.Source]news.Fetcher
		source   news.Source
		want     func(t *testing.T, items []news.Item, err error)
	}{
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
