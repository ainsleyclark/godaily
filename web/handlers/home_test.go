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

func TestHome(t *testing.T) {
	t.Parallel()

	log.SetOutput(io.Discard)

	emptyFeed := func(items *mocknews.MockItemRepository) {
		items.EXPECT().List(gomock.Any(), gomock.Any()).Return([]news.Item{}, nil).AnyTimes()
		items.EXPECT().Count(gomock.Any()).Return(int64(0), nil).AnyTimes()
	}

	tt := map[string]struct {
		url        string
		mock       func(issues *mockdigest.MockIssueRepository, items *mocknews.MockItemRepository)
		wantStatus int
		wantHTML   string
	}{
		"Internal Error": {
			mock: func(issues *mockdigest.MockIssueRepository, items *mocknews.MockItemRepository) {
				issues.EXPECT().
					Latest(gomock.Any(), 4).
					Return(nil, errors.New("internal error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"OK No Issues": {
			mock: func(issues *mockdigest.MockIssueRepository, items *mocknews.MockItemRepository) {
				issues.EXPECT().
					Latest(gomock.Any(), 4).
					Return([]digest.Issue{}, nil)
				emptyFeed(items)
			},
			wantStatus: http.StatusOK,
		},
		"OK With Issue": {
			mock: func(issues *mockdigest.MockIssueRepository, items *mocknews.MockItemRepository) {
				issues.EXPECT().
					Latest(gomock.Any(), 4).
					Return([]digest.Issue{{Slug: "2026-04-28", Subject: "GoDaily - April 28, 2026"}}, nil)
				emptyFeed(items)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "GoDaily - April 28, 2026",
		},
		"OK With Feed": {
			mock: func(issues *mockdigest.MockIssueRepository, items *mocknews.MockItemRepository) {
				issues.EXPECT().
					Latest(gomock.Any(), 4).
					Return([]digest.Issue{}, nil)
				items.EXPECT().List(gomock.Any(), gomock.Any()).
					Return([]news.Item{{Title: "Go 1.26 released", Tag: news.TagRelease}}, nil)
				items.EXPECT().Count(gomock.Any()).Return(int64(12400), nil)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "Browse all 12,400 stories",
		},
		"OK Feed Error": {
			mock: func(issues *mockdigest.MockIssueRepository, items *mocknews.MockItemRepository) {
				issues.EXPECT().
					Latest(gomock.Any(), 4).
					Return([]digest.Issue{}, nil)
				// The feed is best-effort: a failure renders the empty state.
				items.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("boom")).AnyTimes()
				items.EXPECT().Count(gomock.Any()).Return(int64(0), nil).AnyTimes()
			},
			wantStatus: http.StatusOK,
			wantHTML:   "The first stories land soon",
		},
		"OK Confirmed Flash": {
			url: "/?confirmed=1",
			mock: func(issues *mockdigest.MockIssueRepository, items *mocknews.MockItemRepository) {
				issues.EXPECT().
					Latest(gomock.Any(), 4).
					Return([]digest.Issue{}, nil)
				emptyFeed(items)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "You&#39;re confirmed!",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockIssues := mockdigest.NewMockIssueRepository(ctrl)
			mockItems := mocknews.NewMockItemRepository(ctrl)

			if test.mock != nil {
				test.mock(mockIssues, mockItems)
			}

			app := &godaily.App{
				Repository: &godaily.Repository{
					Issues: mockIssues,
					Items:  mockItems,
				},
			}

			kit := webkit.New()
			kit.Get("/", Home(app))

			url := "/"
			if test.url != "" {
				url = test.url
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			kit.ServeHTTP(rec, req)

			assert.Equal(t, test.wantStatus, rec.Code)
			if test.wantHTML != "" {
				assert.Contains(t, rec.Body.String(), test.wantHTML)
			}
		})
	}
}
