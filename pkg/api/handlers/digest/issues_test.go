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
	"net/http"
	"net/http/httptest"
	"testing"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleIssues(t *testing.T) {
	tt := map[string]struct {
		mock       func(issues *mocknews.MockIssueRepository)
		query      string
		wantStatus int
	}{
		"OK default pagination": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().Count(gomock.Any()).Return(int64(2), nil)
				issues.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]news.Issue{
					{ID: 1, Slug: "2026-01-01"},
					{ID: 2, Slug: "2026-01-02"},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"OK with explicit page params": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().Count(gomock.Any()).Return(int64(50), nil)
				issues.EXPECT().List(gomock.Any(), store.ListOptions{Page: 2, PerPage: 10}).Return([]news.Issue{}, nil)
			},
			query:      "?page=2&per_page=10",
			wantStatus: http.StatusOK,
		},
		"Count error": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().Count(gomock.Any()).Return(int64(0), errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"List error": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().Count(gomock.Any()).Return(int64(1), nil)
				issues.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"Invalid page falls back to default": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().Count(gomock.Any()).Return(int64(1), nil)
				issues.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]news.Issue{}, nil)
			},
			query:      "?page=abc",
			wantStatus: http.StatusOK,
		},
		"per_page exceeds max falls back to default": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().Count(gomock.Any()).Return(int64(1), nil)
				issues.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]news.Issue{}, nil)
			},
			query:      "?per_page=999",
			wantStatus: http.StatusOK,
		},
		"OK with status filter": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().CountByStatus(gomock.Any(), news.IssueStatus("draft")).Return(int64(1), nil)
				issues.EXPECT().ListByStatus(gomock.Any(), news.IssueStatus("draft"), store.ListOptions{Page: 1, PerPage: 20}).Return([]news.Issue{
					{ID: 1, Slug: "2026-01-01", Status: "draft"},
				}, nil)
			},
			query:      "?status=draft",
			wantStatus: http.StatusOK,
		},
		"CountByStatus error": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().CountByStatus(gomock.Any(), news.IssueStatus("draft")).Return(int64(0), errors.New("db error"))
			},
			query:      "?status=draft",
			wantStatus: http.StatusInternalServerError,
		},
		"ListByStatus error": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().CountByStatus(gomock.Any(), news.IssueStatus("draft")).Return(int64(1), nil)
				issues.EXPECT().ListByStatus(gomock.Any(), news.IssueStatus("draft"), gomock.Any()).Return(nil, errors.New("db error"))
			},
			query:      "?status=draft",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			issuesMock := mocknews.NewMockIssueRepository(ctrl)
			test.mock(issuesMock)

			a := &godaily.App{
				Config: &env.Config{},
				Repository: &godaily.Repository{
					Issues: issuesMock,
				},
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/digest/issues"+test.query, nil)
			r = r.WithContext(api.WithApp(r.Context(), a))

			HandleIssues(w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
