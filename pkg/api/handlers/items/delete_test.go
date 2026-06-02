// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package items

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

	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

func TestDelete(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Items    *mocknews.MockItemRepository
	}

	setup := func(t *testing.T, id string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		items := mocknews.NewMockItemRepository(ctrl)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/items/"+id, nil)
		if id != "" {
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		}

		return Test{
			Handler:  &Handler{itemsRepo: items},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Items:    items,
		}
	}

	t.Run("Deletes item on success", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42")
		deps.Items.EXPECT().Delete(gomock.Any(), int64(42)).Return(nil)

		err := deps.Handler.Delete(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Not found returns 404", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "99")
		deps.Items.EXPECT().Delete(gomock.Any(), int64(99)).Return(store.ErrNotFound)

		_ = deps.Handler.Delete(deps.Context)
		assert.Equal(t, http.StatusNotFound, deps.Recorder.Code)
	})

	t.Run("Missing id returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")

		_ = deps.Handler.Delete(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Non-numeric id returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "abc")

		_ = deps.Handler.Delete(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Zero id returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "0")

		_ = deps.Handler.Delete(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Store error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "1")
		deps.Items.EXPECT().Delete(gomock.Any(), int64(1)).Return(errors.New("db error"))

		_ = deps.Handler.Delete(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
