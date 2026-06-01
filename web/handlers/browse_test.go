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
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

func TestBrowse(t *testing.T) {
	t.Parallel()

	log.SetOutput(io.Discard)

	tt := map[string]struct {
		url        string
		mock       func(items *mocknews.MockItemRepository, issues *mockdigest.MockIssueRepository)
		wantStatus int
		wantHTML   string
	}{
		"List error": {
			url: "/browse/",
			mock: func(items *mocknews.MockItemRepository, issues *mockdigest.MockIssueRepository) {
				// Queries now run concurrently, so the siblings may fire before
				// the failing List cancels the group. Allow them all.
				items.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("boom")).AnyTimes()
				items.EXPECT().Count(gomock.Any()).Return(int64(0), nil).AnyTimes()
				items.EXPECT().CountMatching(gomock.Any(), gomock.Any()).Return(int64(0), nil).AnyTimes()
				items.EXPECT().SourceCounts(gomock.Any()).Return([]news.SourceCount{}, nil).AnyTimes()
				items.EXPECT().TagCounts(gomock.Any()).Return([]news.TagCount{}, nil).AnyTimes()
			},
			wantStatus: http.StatusInternalServerError,
		},
		"OK empty": {
			url: "/browse/",
			mock: func(items *mocknews.MockItemRepository, issues *mockdigest.MockIssueRepository) {
				items.EXPECT().List(gomock.Any(), gomock.Any()).Return([]news.Item{}, nil).AnyTimes()
				items.EXPECT().Count(gomock.Any()).Return(int64(0), nil)
				items.EXPECT().CountMatching(gomock.Any(), gomock.Any()).Return(int64(0), nil).AnyTimes()
				items.EXPECT().SourceCounts(gomock.Any()).Return([]news.SourceCount{}, nil)
				items.EXPECT().TagCounts(gomock.Any()).Return([]news.TagCount{}, nil)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "No stories match those filters",
		},
		"OK with items": {
			url: "/browse/",
			mock: func(items *mocknews.MockItemRepository, issues *mockdigest.MockIssueRepository) {
				items.EXPECT().List(gomock.Any(), gomock.Any()).Return([]news.Item{
					{ID: 1, Title: "Generics in 1.25", Source: news.SourceGoBlog, Tag: news.TagArticle, URL: "https://go.dev/blog/x", InDigest: true},
					{ID: 2, Title: "HN: Some thread", Source: news.SourceHN, Tag: news.TagDiscussion, URL: "https://news.ycombinator.com/y"},
				}, nil).AnyTimes()
				items.EXPECT().Count(gomock.Any()).Return(int64(2), nil)
				items.EXPECT().CountMatching(gomock.Any(), gomock.Any()).Return(int64(2), nil).AnyTimes()
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
			mock: func(items *mocknews.MockItemRepository, issues *mockdigest.MockIssueRepository) {
				items.EXPECT().
					List(gomock.Any(), gomock.AssignableToTypeOf(news.ItemListOptions{})).
					DoAndReturn(func(_ any, opts news.ItemListOptions) ([]news.Item, error) {
						assert.Empty(t, opts.Sources, "invalid source should be dropped")
						assert.Equal(t, news.ItemSortTop, opts.Sort)
						return []news.Item{}, nil
					}).AnyTimes()
				items.EXPECT().Count(gomock.Any()).Return(int64(0), nil)
				items.EXPECT().CountMatching(gomock.Any(), gomock.AssignableToTypeOf(news.ItemListOptions{})).
					DoAndReturn(func(_ any, opts news.ItemListOptions) (int64, error) {
						assert.Empty(t, opts.Sources, "invalid source should be dropped")
						assert.Equal(t, news.ItemSortTop, opts.Sort)
						return 0, nil
					}).AnyTimes()
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
			mockIssues := mockdigest.NewMockIssueRepository(ctrl)
			mockIssues.EXPECT().Latest(gomock.Any(), gomock.Any()).Return([]digest.Issue{}, nil).AnyTimes()
			if test.mock != nil {
				test.mock(mockItems, mockIssues)
			}

			app := &godaily.App{
				Repository: &godaily.Repository{Items: mockItems, Issues: mockIssues},
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

func TestBrowse_Redirect(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		url          string
		wantStatus   int
		wantLocation string
	}{
		"Clean tab redirects to path form": {
			url:          "/browse/?tab=article",
			wantStatus:   http.StatusMovedPermanently,
			wantLocation: "/browse/article/",
		},
		"Tab with extra params does not redirect": {
			url:        "/browse/?tab=article&sort=top",
			wantStatus: http.StatusOK,
		},
		"No tab does not redirect": {
			url:        "/browse/",
			wantStatus: http.StatusOK,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockItems := mocknews.NewMockItemRepository(ctrl)
			mockIssues := mockdigest.NewMockIssueRepository(ctrl)
			mockIssues.EXPECT().Latest(gomock.Any(), gomock.Any()).Return([]digest.Issue{}, nil).AnyTimes()
			mockItems.EXPECT().List(gomock.Any(), gomock.Any()).Return([]news.Item{}, nil).AnyTimes()
			mockItems.EXPECT().Count(gomock.Any()).Return(int64(0), nil).AnyTimes()
			mockItems.EXPECT().CountMatching(gomock.Any(), gomock.Any()).Return(int64(0), nil).AnyTimes()
			mockItems.EXPECT().SourceCounts(gomock.Any()).Return([]news.SourceCount{}, nil).AnyTimes()
			mockItems.EXPECT().TagCounts(gomock.Any()).Return([]news.TagCount{}, nil).AnyTimes()

			app := &godaily.App{
				Repository: &godaily.Repository{Items: mockItems, Issues: mockIssues},
			}

			kit := webkit.New()
			kit.Get("/browse/", Browse(app))

			req := httptest.NewRequest(http.MethodGet, test.url, nil)
			rec := httptest.NewRecorder()
			kit.ServeHTTP(rec, req)

			assert.Equal(t, test.wantStatus, rec.Code)
			if test.wantLocation != "" {
				assert.Equal(t, test.wantLocation, rec.Header().Get("Location"))
			}
		})
	}
}

func TestBrowseTag(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		url        string
		mock       func(items *mocknews.MockItemRepository, issues *mockdigest.MockIssueRepository)
		wantStatus int
		wantHTML   string
	}{
		"Valid tag renders page": {
			url: "/browse/article/",
			mock: func(items *mocknews.MockItemRepository, issues *mockdigest.MockIssueRepository) {
				items.EXPECT().List(gomock.Any(), gomock.Any()).Return([]news.Item{}, nil).AnyTimes()
				items.EXPECT().Count(gomock.Any()).Return(int64(0), nil)
				items.EXPECT().CountMatching(gomock.Any(), gomock.Any()).Return(int64(0), nil).AnyTimes()
				items.EXPECT().SourceCounts(gomock.Any()).Return([]news.SourceCount{}, nil)
				items.EXPECT().TagCounts(gomock.Any()).Return([]news.TagCount{}, nil)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "Articles",
		},
		"Unknown tag returns 404": {
			url:        "/browse/nonsense/",
			wantStatus: http.StatusNotFound,
		},
		"Repository error returns 500": {
			url: "/browse/release/",
			mock: func(items *mocknews.MockItemRepository, issues *mockdigest.MockIssueRepository) {
				items.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error")).AnyTimes()
				items.EXPECT().Count(gomock.Any()).Return(int64(0), nil).AnyTimes()
				items.EXPECT().CountMatching(gomock.Any(), gomock.Any()).Return(int64(0), nil).AnyTimes()
				items.EXPECT().SourceCounts(gomock.Any()).Return([]news.SourceCount{}, nil).AnyTimes()
				items.EXPECT().TagCounts(gomock.Any()).Return([]news.TagCount{}, nil).AnyTimes()
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockItems := mocknews.NewMockItemRepository(ctrl)
			mockIssues := mockdigest.NewMockIssueRepository(ctrl)
			mockIssues.EXPECT().Latest(gomock.Any(), gomock.Any()).Return([]digest.Issue{}, nil).AnyTimes()
			if test.mock != nil {
				test.mock(mockItems, mockIssues)
			}

			app := &godaily.App{
				Repository: &godaily.Repository{Items: mockItems, Issues: mockIssues},
			}

			kit := webkit.New()
			kit.Get("/browse/{tag}/", BrowseTag(app))

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
