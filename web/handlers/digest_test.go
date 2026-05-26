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
	"github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

func TestDigest(t *testing.T) {
	t.Parallel()

	log.SetOutput(io.Discard)

	tt := map[string]struct {
		mock       func(issues *mockdigest.MockIssueRepository)
		wantStatus int
		wantHTML   string
	}{
		"Not Found": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().
					FindBySlug(gomock.Any(), "issue-1").
					Return(digest.Issue{}, store.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		"Internal Error": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().
					FindBySlug(gomock.Any(), "issue-1").
					Return(digest.Issue{}, errors.New("internal error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"OK": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().
					FindBySlug(gomock.Any(), "issue-1").
					Return(digest.Issue{Slug: "issue-1", Subject: "Go Weekly #1"}, nil)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "Go Weekly #1",
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
			kit.Get("/issues/{slug}/", Digest(app))

			req := httptest.NewRequest(http.MethodGet, "/issues/issue-1/", nil)
			rec := httptest.NewRecorder()
			kit.ServeHTTP(rec, req)

			assert.Equal(t, test.wantStatus, rec.Code)
			if test.wantHTML != "" {
				assert.Contains(t, rec.Body.String(), test.wantHTML)
			}
		})
	}
}
