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
	"github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

func TestIssues(t *testing.T) {
	t.Parallel()

	log.SetOutput(io.Discard)

	tt := map[string]struct {
		mock       func(issues *mockdigest.MockIssueRepository)
		wantStatus int
		wantHTML   string
	}{
		"Internal Error": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("internal error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"OK No Issues": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return([]digest.Issue{}, nil)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "The complete archive",
		},
		"Find Error": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return([]digest.Issue{{ID: 1, Slug: "2026-04-28"}}, nil)
				issues.EXPECT().
					Find(gomock.Any(), int64(1)).
					Return(digest.Issue{}, errors.New("find error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"OK With Issues": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return([]digest.Issue{
						{ID: 1, Slug: "2026-04-28"},
						{ID: 2, Slug: "2026-04-25"},
					}, nil)
				issues.EXPECT().
					Find(gomock.Any(), int64(1)).
					Return(digest.Issue{ID: 1, Slug: "2026-04-28", Subject: "GoDaily - April 28, 2026", Items: []news.Item{{Title: "foo"}}}, nil)
				issues.EXPECT().
					Find(gomock.Any(), int64(2)).
					Return(digest.Issue{ID: 2, Slug: "2026-04-25", Subject: "GoDaily - April 25, 2026", Items: []news.Item{{Title: "bar"}}}, nil)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "GoDaily - April 28, 2026",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockIssues := mockdigest.NewMockIssueRepository(ctrl)

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
