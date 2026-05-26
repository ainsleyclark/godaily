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
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestBySlug(t *testing.T) {
	tt := map[string]struct {
		mock       func(issues *mockdigest.MockIssueRepository)
		slug       string
		wantStatus int
	}{
		"OK": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().FindBySlug(gomock.Any(), "2026-01-01").Return(digest.Issue{ID: 1, Slug: "2026-01-01"}, nil)
			},
			slug:       "2026-01-01",
			wantStatus: http.StatusOK,
		},
		"Not found": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().FindBySlug(gomock.Any(), "unknown").Return(digest.Issue{}, store.ErrNotFound)
			},
			slug:       "unknown",
			wantStatus: http.StatusNotFound,
		},
		"Missing slug": {
			mock:       func(issues *mockdigest.MockIssueRepository) {},
			slug:       "",
			wantStatus: http.StatusBadRequest,
		},
		"Store error": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().FindBySlug(gomock.Any(), "2026-01-01").Return(digest.Issue{}, errors.New("db error"))
			},
			slug:       "2026-01-01",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			issuesMock := mockdigest.NewMockIssueRepository(ctrl)
			test.mock(issuesMock)

			h := &Handler{issuesRepo: issuesMock}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/issues/"+test.slug, nil)
			if test.slug != "" {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("slug", test.slug)
				r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
			}

			invoke(h.BySlug, w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}

func invoke(h func(*webkit.Context) error, w *httptest.ResponseRecorder, r *http.Request) {
	c := webkit.NewContext(w, r)
	if err := h(c); err != nil {
		var e *webkit.Error
		if errors.As(err, &e) {
			_ = c.JSON(e.Code, map[string]string{"error": e.Message})
		} else {
			_ = c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}
}
