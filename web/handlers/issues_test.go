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
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

func TestIssues(t *testing.T) {
	t.Parallel()

	log.SetOutput(io.Discard)

	tt := map[string]struct {
		mock       func(issues *mocknews.MockIssueRepository)
		wantStatus int
		wantHTML   string
	}{
		"Internal Error": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("internal error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"OK No Issues": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return([]news.Issue{}, nil)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "The complete archive",
		},
		"Find Error": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return([]news.Issue{{ID: 1, Slug: "2026-04-28"}}, nil)
				issues.EXPECT().
					Find(gomock.Any(), int64(1)).
					Return(news.Issue{}, errors.New("find error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"OK With Issues": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return([]news.Issue{
						{ID: 1, Slug: "2026-04-28"},
						{ID: 2, Slug: "2026-04-25"},
					}, nil)
				issues.EXPECT().
					Find(gomock.Any(), int64(1)).
					Return(news.Issue{ID: 1, Slug: "2026-04-28", Subject: "GoDaily - April 28, 2026", Items: []news.Item{{Title: "foo"}}}, nil)
				issues.EXPECT().
					Find(gomock.Any(), int64(2)).
					Return(news.Issue{ID: 2, Slug: "2026-04-25", Subject: "GoDaily - April 25, 2026", Items: []news.Item{{Title: "bar"}}}, nil)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "GoDaily - April 28, 2026",
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
			kit.Get("/issues/", Issues(app))

			req := httptest.NewRequest(http.MethodGet, "/issues/", nil)
			rec := httptest.NewRecorder()
			kit.ServeHTTP(rec, req)

			assert.Equal(t, test.wantStatus, rec.Code)
			if test.wantHTML != "" {
				assert.Contains(t, rec.Body.String(), test.wantHTML)
			}
		})
	}
}
