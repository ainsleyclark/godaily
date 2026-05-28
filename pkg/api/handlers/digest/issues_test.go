// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
)

func TestIssues(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Issues   *mockdigest.MockIssueRepository
	}

	setup := func(t *testing.T, query string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		issues := mockdigest.NewMockIssueRepository(ctrl)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/digest/issues"+query, nil)

		return Test{
			Handler:  &Handler{issuesRepo: issues},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Issues:   issues,
		}
	}

	t.Run("Returns issues on default pagination", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Issues.EXPECT().Count(gomock.Any()).Return(int64(2), nil)
		deps.Issues.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]digest.Issue{
			{ID: 1, Slug: "2026-01-01"},
			{ID: 2, Slug: "2026-01-02"},
		}, nil)

		err := deps.Handler.Issues(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Returns issues with explicit page params", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "?page=2&per_page=10")
		deps.Issues.EXPECT().Count(gomock.Any()).Return(int64(50), nil)
		deps.Issues.EXPECT().List(gomock.Any(), store.ListOptions{Page: 2, PerPage: 10}).Return([]digest.Issue{}, nil)

		err := deps.Handler.Issues(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Count error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Issues.EXPECT().Count(gomock.Any()).Return(int64(0), errors.New("db error"))

		_ = deps.Handler.Issues(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})

	t.Run("List error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Issues.EXPECT().Count(gomock.Any()).Return(int64(1), nil)
		deps.Issues.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))

		_ = deps.Handler.Issues(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})

	t.Run("Invalid page falls back to default", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "?page=abc")
		deps.Issues.EXPECT().Count(gomock.Any()).Return(int64(1), nil)
		deps.Issues.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]digest.Issue{}, nil)

		err := deps.Handler.Issues(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Per page exceeds max falls back to default", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "?per_page=999")
		deps.Issues.EXPECT().Count(gomock.Any()).Return(int64(1), nil)
		deps.Issues.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]digest.Issue{}, nil)

		err := deps.Handler.Issues(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Returns issues filtered by status", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "?status=draft")
		deps.Issues.EXPECT().CountByStatus(gomock.Any(), digest.IssueStatus("draft")).Return(int64(1), nil)
		deps.Issues.EXPECT().ListByStatus(gomock.Any(), digest.IssueStatus("draft"), store.ListOptions{Page: 1, PerPage: 20}).Return([]digest.Issue{
			{ID: 1, Slug: "2026-01-01", Status: "draft"},
		}, nil)

		err := deps.Handler.Issues(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("CountByStatus error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "?status=draft")
		deps.Issues.EXPECT().CountByStatus(gomock.Any(), digest.IssueStatus("draft")).Return(int64(0), errors.New("db error"))

		_ = deps.Handler.Issues(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})

	t.Run("ListByStatus error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "?status=draft")
		deps.Issues.EXPECT().CountByStatus(gomock.Any(), digest.IssueStatus("draft")).Return(int64(1), nil)
		deps.Issues.EXPECT().ListByStatus(gomock.Any(), digest.IssueStatus("draft"), gomock.Any()).Return(nil, errors.New("db error"))

		_ = deps.Handler.Issues(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
