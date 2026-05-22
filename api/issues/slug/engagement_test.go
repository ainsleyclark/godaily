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

package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/domain/engagement"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	tt := map[string]struct {
		mock       func(issues *mocknews.MockIssueRepository, events *mockengagement.MockEmailEventRepository)
		slug       string
		limit      string
		secret     string
		authHeader string
		wantStatus int
	}{
		"OK": {
			mock: func(issues *mocknews.MockIssueRepository, events *mockengagement.MockEmailEventRepository) {
				issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(news.Issue{ID: 1, Slug: "2026-05-22"}, nil)
				events.EXPECT().IssueStats(gomock.Any(), int64(1)).Return(engagement.IssueStats{IssueID: 1, Delivered: 100}, nil)
				events.EXPECT().TopItems(gomock.Any(), int64(1), int64(10)).Return([]engagement.ItemClicks{{ItemID: 42, Clicks: 5}}, nil)
			},
			slug:       "2026-05-22",
			wantStatus: http.StatusOK,
		},
		"Custom limit is passed through": {
			mock: func(issues *mocknews.MockIssueRepository, events *mockengagement.MockEmailEventRepository) {
				issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(news.Issue{ID: 1}, nil)
				events.EXPECT().IssueStats(gomock.Any(), int64(1)).Return(engagement.IssueStats{}, nil)
				events.EXPECT().TopItems(gomock.Any(), int64(1), int64(3)).Return(nil, nil)
			},
			slug:       "2026-05-22",
			limit:      "3",
			wantStatus: http.StatusOK,
		},
		"Non-positive limit falls back to the default": {
			mock: func(issues *mocknews.MockIssueRepository, events *mockengagement.MockEmailEventRepository) {
				issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(news.Issue{ID: 1}, nil)
				events.EXPECT().IssueStats(gomock.Any(), int64(1)).Return(engagement.IssueStats{}, nil)
				events.EXPECT().TopItems(gomock.Any(), int64(1), int64(10)).Return(nil, nil)
			},
			slug:       "2026-05-22",
			limit:      "-5",
			wantStatus: http.StatusOK,
		},
		"Issue not found": {
			mock: func(issues *mocknews.MockIssueRepository, _ *mockengagement.MockEmailEventRepository) {
				issues.EXPECT().FindBySlug(gomock.Any(), "unknown").Return(news.Issue{}, store.ErrNotFound)
			},
			slug:       "unknown",
			wantStatus: http.StatusNotFound,
		},
		"Missing slug": {
			mock:       func(*mocknews.MockIssueRepository, *mockengagement.MockEmailEventRepository) {},
			slug:       "",
			wantStatus: http.StatusBadRequest,
		},
		"Issue store error": {
			mock: func(issues *mocknews.MockIssueRepository, _ *mockengagement.MockEmailEventRepository) {
				issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(news.Issue{}, errors.New("db error"))
			},
			slug:       "2026-05-22",
			wantStatus: http.StatusInternalServerError,
		},
		"Stats store error": {
			mock: func(issues *mocknews.MockIssueRepository, events *mockengagement.MockEmailEventRepository) {
				issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(news.Issue{ID: 1}, nil)
				events.EXPECT().IssueStats(gomock.Any(), int64(1)).Return(engagement.IssueStats{}, errors.New("db error"))
			},
			slug:       "2026-05-22",
			wantStatus: http.StatusInternalServerError,
		},
		"Top items store error": {
			mock: func(issues *mocknews.MockIssueRepository, events *mockengagement.MockEmailEventRepository) {
				issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(news.Issue{ID: 1}, nil)
				events.EXPECT().IssueStats(gomock.Any(), int64(1)).Return(engagement.IssueStats{}, nil)
				events.EXPECT().TopItems(gomock.Any(), int64(1), int64(10)).Return(nil, errors.New("db error"))
			},
			slug:       "2026-05-22",
			wantStatus: http.StatusInternalServerError,
		},
		"Unauthorized when the API secret is missing": {
			mock:       func(*mocknews.MockIssueRepository, *mockengagement.MockEmailEventRepository) {},
			slug:       "2026-05-22",
			secret:     "topsecret",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			issuesMock := mocknews.NewMockIssueRepository(ctrl)
			eventsMock := mockengagement.NewMockEmailEventRepository(ctrl)
			test.mock(issuesMock, eventsMock)

			a := &godaily.App{
				Config: &env.Config{APISecret: test.secret},
				Repository: &godaily.Repository{
					Issues:      issuesMock,
					EmailEvents: eventsMock,
				},
			}

			target := "/api/issues/slug/engagement"
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, target, nil)

			q := r.URL.Query()
			if test.slug != "" {
				q.Set("slug", test.slug)
			}
			if test.limit != "" {
				q.Set("limit", test.limit)
			}
			r.URL.RawQuery = q.Encode()
			if test.authHeader != "" {
				r.Header.Set("Authorization", test.authHeader)
			}

			Handler(w, r.WithContext(api.WithApp(r.Context(), a)))

			assert.Equal(t, test.wantStatus, w.Code)

			if name == "OK" {
				var got engagementResponse
				require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
				assert.Equal(t, int64(100), got.Stats.Delivered)
				require.Len(t, got.TopItems, 1)
				assert.Equal(t, int64(42), got.TopItems[0].ItemID)
			}
		})
	}
}
