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

func TestBySlug(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Issues   *mockdigest.MockIssueRepository
	}

	setup := func(t *testing.T, slug string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		issues := mockdigest.NewMockIssueRepository(ctrl)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/issues/"+slug, nil)
		if slug != "" {
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("slug", slug)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		}

		return Test{
			Handler:  &Handler{issuesRepo: issues},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Issues:   issues,
		}
	}

	t.Run("Returns issue on success", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "2026-01-01")
		deps.Issues.EXPECT().FindBySlug(gomock.Any(), "2026-01-01").Return(digest.Issue{ID: 1, Slug: "2026-01-01"}, nil)

		err := deps.Handler.BySlug(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Not found returns 404", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "unknown")
		deps.Issues.EXPECT().FindBySlug(gomock.Any(), "unknown").Return(digest.Issue{}, store.ErrNotFound)

		_ = deps.Handler.BySlug(deps.Context)
		assert.Equal(t, http.StatusNotFound, deps.Recorder.Code)
	})

	t.Run("Missing slug returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")

		_ = deps.Handler.BySlug(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Store error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "2026-01-01")
		deps.Issues.EXPECT().FindBySlug(gomock.Any(), "2026-01-01").Return(digest.Issue{}, errors.New("db error"))

		_ = deps.Handler.BySlug(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
