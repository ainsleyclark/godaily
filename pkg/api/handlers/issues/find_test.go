// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

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

func TestFind(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Issues   *mockdigest.MockIssueRepository
	}

	setup := func(t *testing.T, key string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		issues := mockdigest.NewMockIssueRepository(ctrl)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/issues/"+key, nil)
		if key != "" {
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("key", key)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		}

		return Test{
			Handler:  &Handler{issuesRepo: issues},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Issues:   issues,
		}
	}

	t.Run("Numeric key looks up by ID", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42")
		deps.Issues.EXPECT().Find(gomock.Any(), int64(42)).Return(digest.Issue{ID: 42, Slug: "2026-01-01"}, nil)

		err := deps.Handler.Find(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Non-numeric key looks up by slug", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "2026-01-01")
		deps.Issues.EXPECT().FindBySlug(gomock.Any(), "2026-01-01").Return(digest.Issue{ID: 1, Slug: "2026-01-01"}, nil)

		err := deps.Handler.Find(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Zero is treated as a slug", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "0")
		deps.Issues.EXPECT().FindBySlug(gomock.Any(), "0").Return(digest.Issue{}, store.ErrNotFound)

		_ = deps.Handler.Find(deps.Context)
		assert.Equal(t, http.StatusNotFound, deps.Recorder.Code)
	})

	t.Run("Missing key returns 400", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")

		_ = deps.Handler.Find(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Not found returns 404", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "99")
		deps.Issues.EXPECT().Find(gomock.Any(), int64(99)).Return(digest.Issue{}, store.ErrNotFound)

		_ = deps.Handler.Find(deps.Context)
		assert.Equal(t, http.StatusNotFound, deps.Recorder.Code)
	})

	t.Run("Store error returns 500", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "5")
		deps.Issues.EXPECT().Find(gomock.Any(), int64(5)).Return(digest.Issue{}, errors.New("db error"))

		_ = deps.Handler.Find(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})

	t.Run("Slug store error returns 500", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "2026-01-01")
		deps.Issues.EXPECT().FindBySlug(gomock.Any(), "2026-01-01").Return(digest.Issue{}, errors.New("db error"))

		_ = deps.Handler.Find(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
