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
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/env"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	tt := map[string]struct {
		mock       func(issues *mocknews.MockIssueRepository)
		slug       string
		wantStatus int
	}{
		"OK": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().FindBySlug(gomock.Any(), "2026-01-01").Return(news.Issue{
					ID:   1,
					Slug: "2026-01-01",
				}, nil)
			},
			slug:       "2026-01-01",
			wantStatus: http.StatusOK,
		},
		"Not found": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().FindBySlug(gomock.Any(), "unknown").Return(news.Issue{}, store.ErrNotFound)
			},
			slug:       "unknown",
			wantStatus: http.StatusNotFound,
		},
		"Missing slug": {
			mock:       func(issues *mocknews.MockIssueRepository) {},
			slug:       "",
			wantStatus: http.StatusBadRequest,
		},
		"Store error": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().FindBySlug(gomock.Any(), "2026-01-01").Return(news.Issue{}, errors.New("db error"))
			},
			slug:       "2026-01-01",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			issuesMock := mocknews.NewMockIssueRepository(ctrl)
			test.mock(issuesMock)

			api.App = &godaily.App{
				Config: &env.Config{},
				Repository: &godaily.Repository{
					Issues: issuesMock,
				},
			}

			target := "/api/issues/"
			if test.slug != "" {
				target += test.slug + "/"
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, target, nil)
			r.RemoteAddr = "1.2.3.4:1234"

			if test.slug != "" {
				q := r.URL.Query()
				q.Set("slug", test.slug)
				r.URL.RawQuery = q.Encode()
			}

			Handler(w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
