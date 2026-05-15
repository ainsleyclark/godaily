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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/ainsleyclark/godaily/pkg/news"
)

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
				assert.Equal(t, "to@example.com", agg.adminEmailAddress)
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
				assert.Empty(t, agg.adminEmailAddress)
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

			got, err := New(email.New(""), test.envAddr, nil, nil, nil, nil, nil)
			test.want(t, got, err)
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
