// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestIssues(t *testing.T) {
	tt := map[string]struct {
		mock       func(issues *mockdigest.MockIssueRepository)
		query      string
		wantStatus int
	}{
		"OK default pagination": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().Count(gomock.Any()).Return(int64(2), nil)
				issues.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]digest.Issue{
					{ID: 1, Slug: "2026-01-01"},
					{ID: 2, Slug: "2026-01-02"},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"OK with explicit page params": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().Count(gomock.Any()).Return(int64(50), nil)
				issues.EXPECT().List(gomock.Any(), store.ListOptions{Page: 2, PerPage: 10}).Return([]digest.Issue{}, nil)
			},
			query:      "?page=2&per_page=10",
			wantStatus: http.StatusOK,
		},
		"Count error": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().Count(gomock.Any()).Return(int64(0), errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"List error": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().Count(gomock.Any()).Return(int64(1), nil)
				issues.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"Invalid page falls back to default": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().Count(gomock.Any()).Return(int64(1), nil)
				issues.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]digest.Issue{}, nil)
			},
			query:      "?page=abc",
			wantStatus: http.StatusOK,
		},
		"per_page exceeds max falls back to default": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().Count(gomock.Any()).Return(int64(1), nil)
				issues.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]digest.Issue{}, nil)
			},
			query:      "?per_page=999",
			wantStatus: http.StatusOK,
		},
		"OK with status filter": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().CountByStatus(gomock.Any(), digest.IssueStatus("draft")).Return(int64(1), nil)
				issues.EXPECT().ListByStatus(gomock.Any(), digest.IssueStatus("draft"), store.ListOptions{Page: 1, PerPage: 20}).Return([]digest.Issue{
					{ID: 1, Slug: "2026-01-01", Status: "draft"},
				}, nil)
			},
			query:      "?status=draft",
			wantStatus: http.StatusOK,
		},
		"CountByStatus error": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().CountByStatus(gomock.Any(), digest.IssueStatus("draft")).Return(int64(0), errors.New("db error"))
			},
			query:      "?status=draft",
			wantStatus: http.StatusInternalServerError,
		},
		"ListByStatus error": {
			mock: func(issues *mockdigest.MockIssueRepository) {
				issues.EXPECT().CountByStatus(gomock.Any(), digest.IssueStatus("draft")).Return(int64(1), nil)
				issues.EXPECT().ListByStatus(gomock.Any(), digest.IssueStatus("draft"), gomock.Any()).Return(nil, errors.New("db error"))
			},
			query:      "?status=draft",
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
			r := httptest.NewRequest(http.MethodGet, "/digest/issues"+test.query, nil)
			invoke(h.Issues, w, r)
			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
