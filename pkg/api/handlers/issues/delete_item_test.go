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
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

func TestDeleteItem(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Issues   *mockdigest.MockIssueRepository
		Items    *mocknews.MockItemRepository
	}

	setup := func(t *testing.T, id, itemID string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		issues := mockdigest.NewMockIssueRepository(ctrl)
		items := mocknews.NewMockItemRepository(ctrl)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/issues/"+id+"/items/"+itemID, nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id)
		rctx.URLParams.Add("itemID", itemID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		return Test{
			Handler:  &Handler{issuesRepo: issues, itemsRepo: items},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Issues:   issues,
			Items:    items,
		}
	}

	t.Run("Unlinks item and returns refreshed issue", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", "7")
		deps.Items.EXPECT().UnlinkFromIssue(gomock.Any(), int64(42), int64(7)).Return(nil)
		deps.Issues.EXPECT().Find(gomock.Any(), int64(42)).Return(digest.Issue{ID: 42, Status: digest.IssueStatusDraft}, nil)

		err := deps.Handler.DeleteItem(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Invalid id returns 400", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "abc", "7")
		_ = deps.Handler.DeleteItem(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Invalid item id returns 400", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", "0")
		_ = deps.Handler.DeleteItem(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Not found returns 404", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", "7")
		deps.Items.EXPECT().UnlinkFromIssue(gomock.Any(), int64(42), int64(7)).Return(store.ErrNotFound)

		_ = deps.Handler.DeleteItem(deps.Context)
		assert.Equal(t, http.StatusNotFound, deps.Recorder.Code)
	})

	t.Run("Non-draft issue returns 409", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", "7")
		deps.Items.EXPECT().UnlinkFromIssue(gomock.Any(), int64(42), int64(7)).Return(digest.ErrIssueNotDraft)

		_ = deps.Handler.DeleteItem(deps.Context)
		assert.Equal(t, http.StatusConflict, deps.Recorder.Code)
	})

	t.Run("Repo error returns 500", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", "7")
		deps.Items.EXPECT().UnlinkFromIssue(gomock.Any(), int64(42), int64(7)).Return(errors.New("db error"))

		_ = deps.Handler.DeleteItem(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})

	t.Run("Refetch failure returns 500", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", "7")
		deps.Items.EXPECT().UnlinkFromIssue(gomock.Any(), int64(42), int64(7)).Return(nil)
		deps.Issues.EXPECT().Find(gomock.Any(), int64(42)).Return(digest.Issue{}, errors.New("db error"))

		_ = deps.Handler.DeleteItem(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
