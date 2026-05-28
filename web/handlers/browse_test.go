// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handlers

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

func TestBrowse(t *testing.T) {
	t.Parallel()

	log.SetOutput(io.Discard)

	tt := map[string]struct {
		url        string
		mock       func(items *mocknews.MockItemRepository)
		wantStatus int
		wantHTML   string
	}{
		"List error": {
			url: "/browse/",
			mock: func(items *mocknews.MockItemRepository) {
				items.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("boom"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"OK empty": {
			url: "/browse/",
			mock: func(items *mocknews.MockItemRepository) {
				items.EXPECT().List(gomock.Any(), gomock.Any()).Return([]news.Item{}, nil).AnyTimes()
				items.EXPECT().Count(gomock.Any()).Return(int64(0), nil)
				items.EXPECT().SourceCounts(gomock.Any()).Return([]news.SourceCount{}, nil)
				items.EXPECT().TagCounts(gomock.Any()).Return([]news.TagCount{}, nil)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "No stories match those filters",
		},
		"OK with items": {
			url: "/browse/",
			mock: func(items *mocknews.MockItemRepository) {
				items.EXPECT().List(gomock.Any(), gomock.Any()).Return([]news.Item{
					{ID: 1, Title: "Generics in 1.25", Source: news.SourceGoBlog, Tag: news.TagArticle, URL: "https://go.dev/blog/x", InDigest: true},
					{ID: 2, Title: "HN: Some thread", Source: news.SourceHN, Tag: news.TagDiscussion, URL: "https://news.ycombinator.com/y"},
				}, nil).AnyTimes()
				items.EXPECT().Count(gomock.Any()).Return(int64(2), nil)
				items.EXPECT().SourceCounts(gomock.Any()).Return([]news.SourceCount{
					{Source: news.SourceGoBlog, Count: 1},
					{Source: news.SourceHN, Count: 1},
				}, nil)
				items.EXPECT().TagCounts(gomock.Any()).Return([]news.TagCount{
					{Tag: news.TagArticle, Count: 1},
					{Tag: news.TagDiscussion, Count: 1},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "Generics in 1.25",
		},
		"Invalid source ignored": {
			url: "/browse/?source=does-not-exist&sort=top",
			mock: func(items *mocknews.MockItemRepository) {
				items.EXPECT().
					List(gomock.Any(), gomock.AssignableToTypeOf(news.ItemListOptions{})).
					DoAndReturn(func(_ any, opts news.ItemListOptions) ([]news.Item, error) {
						assert.Empty(t, opts.Sources, "invalid source should be dropped")
						assert.Equal(t, news.ItemSortTop, opts.Sort)
						return []news.Item{}, nil
					}).AnyTimes()
				items.EXPECT().Count(gomock.Any()).Return(int64(0), nil)
				items.EXPECT().SourceCounts(gomock.Any()).Return([]news.SourceCount{}, nil)
				items.EXPECT().TagCounts(gomock.Any()).Return([]news.TagCount{}, nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockItems := mocknews.NewMockItemRepository(ctrl)
			if test.mock != nil {
				test.mock(mockItems)
			}

			app := &godaily.App{
				Repository: &godaily.Repository{Items: mockItems},
			}

			kit := webkit.New()
			kit.Get("/browse/", Browse(app))

			req := httptest.NewRequest(http.MethodGet, test.url, nil)
			rec := httptest.NewRecorder()
			kit.ServeHTTP(rec, req)

			assert.Equal(t, test.wantStatus, rec.Code)
			if test.wantHTML != "" {
				assert.Contains(t, rec.Body.String(), test.wantHTML)
			}
		})
	}
}
