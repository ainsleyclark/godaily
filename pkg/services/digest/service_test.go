// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
)

func TestNew(t *testing.T) {
	tt := map[string]struct {
		registry map[news.Source]news.Fetcher
		envAddr  string
		want     func(t *testing.T, agg *Service, err error)
	}{
		"OK": {
			registry: allRegistered(),
			envAddr:  "to@example.com",
			want: func(t *testing.T, agg *Service, err error) {
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
			want: func(t *testing.T, agg *Service, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, agg)
				assert.Empty(t, agg.adminEmailAddress)
			},
		},
		"Empty Registry": {
			registry: map[news.Source]news.Fetcher{},
			envAddr:  "to@example.com",
			want: func(t *testing.T, agg *Service, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, agg)
			},
		},
		"Validate Error": {
			registry: map[news.Source]news.Fetcher{
				news.SourceGoBlog: mockFetcher{},
			},
			envAddr: "to@example.com",
			want: func(t *testing.T, agg *Service, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, agg)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(news.SwapRegistry(test.registry))

			got, err := New(email.New(""), test.envAddr, nil, nil, nil, nil, nil, nil)
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

			agg := Service{}
			got, err := agg.fetchSource(t.Context(), test.source)
			test.want(t, got, err)
		})
	}
}
