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
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

func TestHome(t *testing.T) {
	t.Parallel()

	log.SetOutput(io.Discard)

	tt := map[string]struct {
		url        string
		mock       func(issues *mocknews.MockIssueRepository)
		wantStatus int
		wantHTML   string
	}{
		"Internal Error": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().
					Latest(gomock.Any(), 1).
					Return(nil, errors.New("internal error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"OK No Issues": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().
					Latest(gomock.Any(), 1).
					Return([]news.Issue{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"OK With Issue": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().
					Latest(gomock.Any(), 1).
					Return([]news.Issue{{Slug: "2026-04-28", Subject: "GoDaily - April 28, 2026"}}, nil)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "GoDaily - April 28, 2026",
		},
		"OK Confirmed Flash": {
			url: "/?confirmed=1",
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().
					Latest(gomock.Any(), 1).
					Return([]news.Issue{}, nil)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "You&#39;re confirmed!",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockIssues := mocknews.NewMockIssueRepository(ctrl)

			if test.mock != nil {
				test.mock(mockIssues)
			}

			app := &godaily.App{
				Repository: &godaily.Repository{
					Issues: mockIssues,
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
