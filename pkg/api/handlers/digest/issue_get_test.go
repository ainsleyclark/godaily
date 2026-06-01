// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
)

func TestIssueByID(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Issues   *mockdigest.MockIssueRepository
	}

	setup := func(t *testing.T, id string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		issues := mockdigest.NewMockIssueRepository(ctrl)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/digest/issues/"+id, nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		return Test{
			Handler:  &Handler{issuesRepo: issues},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Issues:   issues,
		}
	}

	t.Run("Returns issue on success", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42")
		deps.Issues.EXPECT().Find(gomock.Any(), int64(42)).Return(digest.Issue{ID: 42, Slug: "2026-01-01"}, nil)

		err := deps.Handler.IssueByID(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Not found returns 404", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "99")
		deps.Issues.EXPECT().Find(gomock.Any(), int64(99)).Return(digest.Issue{}, store.ErrNotFound)

		_ = deps.Handler.IssueByID(deps.Context)
		assert.Equal(t, http.StatusNotFound, deps.Recorder.Code)
	})

	t.Run("Invalid id returns 400", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "abc")
		_ = deps.Handler.IssueByID(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Find error returns 500", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "5")
		deps.Issues.EXPECT().Find(gomock.Any(), int64(5)).Return(digest.Issue{}, errors.New("db error"))

		_ = deps.Handler.IssueByID(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
