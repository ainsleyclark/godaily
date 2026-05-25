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

package issues

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/engagement"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleBySlug(t *testing.T) {
	tt := map[string]struct {
		mockIssues      func(m *mocknews.MockIssueRepository)
		mockEmailEvents func(m *mockengagement.MockEmailEventRepository)
		slug            string
		wantStatus      int
	}{
		"OK": {
			mockIssues: func(m *mocknews.MockIssueRepository) {
				m.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(news.Issue{ID: 7, Slug: "2026-05-22"}, nil)
			},
			mockEmailEvents: func(m *mockengagement.MockEmailEventRepository) {
				m.EXPECT().IssueStats(gomock.Any(), int64(7)).Return(engagement.IssueStats{IssueID: 7, Delivered: 312}, nil)
				m.EXPECT().TopLinks(gomock.Any(), int64(7), int64(10)).Return([]engagement.LinkClicks{
					{URL: "https://go.dev", Clicks: 18},
				}, nil)
			},
			slug:       "2026-05-22",
			wantStatus: http.StatusOK,
		},
		"Missing slug": {
			mockIssues:      func(m *mocknews.MockIssueRepository) {},
			mockEmailEvents: func(m *mockengagement.MockEmailEventRepository) {},
			slug:            "",
			wantStatus:      http.StatusBadRequest,
		},
		"Issue not found": {
			mockIssues: func(m *mocknews.MockIssueRepository) {
				m.EXPECT().FindBySlug(gomock.Any(), "unknown").Return(news.Issue{}, store.ErrNotFound)
			},
			mockEmailEvents: func(m *mockengagement.MockEmailEventRepository) {},
			slug:            "unknown",
			wantStatus:      http.StatusNotFound,
		},
		"Issue store error": {
			mockIssues: func(m *mocknews.MockIssueRepository) {
				m.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(news.Issue{}, errors.New("db error"))
			},
			mockEmailEvents: func(m *mockengagement.MockEmailEventRepository) {},
			slug:            "2026-05-22",
			wantStatus:      http.StatusInternalServerError,
		},
		"Stats store error": {
			mockIssues: func(m *mocknews.MockIssueRepository) {
				m.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(news.Issue{ID: 7}, nil)
			},
			mockEmailEvents: func(m *mockengagement.MockEmailEventRepository) {
				m.EXPECT().IssueStats(gomock.Any(), int64(7)).Return(engagement.IssueStats{}, errors.New("db error"))
			},
			slug:       "2026-05-22",
			wantStatus: http.StatusInternalServerError,
		},
		"TopLinks store error": {
			mockIssues: func(m *mocknews.MockIssueRepository) {
				m.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(news.Issue{ID: 7}, nil)
			},
			mockEmailEvents: func(m *mockengagement.MockEmailEventRepository) {
				m.EXPECT().IssueStats(gomock.Any(), int64(7)).Return(engagement.IssueStats{IssueID: 7}, nil)
				m.EXPECT().TopLinks(gomock.Any(), int64(7), int64(10)).Return(nil, errors.New("db error"))
			},
			slug:       "2026-05-22",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			issuesMock := mocknews.NewMockIssueRepository(ctrl)
			emailEventsMock := mockengagement.NewMockEmailEventRepository(ctrl)
			test.mockIssues(issuesMock)
			test.mockEmailEvents(emailEventsMock)

			a := &godaily.App{
				Config: &env.Config{},
				Repository: &godaily.Repository{
					Issues:      issuesMock,
					EmailEvents: emailEventsMock,
				},
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/metrics/issues/"+test.slug, nil)
			r = r.WithContext(api.WithApp(r.Context(), a))
			if test.slug != "" {
				r.SetPathValue("slug", test.slug)
			}

			HandleBySlug(w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
