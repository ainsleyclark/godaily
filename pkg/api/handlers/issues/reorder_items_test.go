// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package issues

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

func TestReorderItems(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Issues   *mockdigest.MockIssueRepository
		Items    *mocknews.MockItemRepository
	}

	setup := func(t *testing.T, id, body string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		issues := mockdigest.NewMockIssueRepository(ctrl)
		items := mocknews.NewMockItemRepository(ctrl)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/issues/"+id+"/items/reorder", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		return Test{
			Handler:  &Handler{issuesRepo: issues, itemsRepo: items},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Issues:   issues,
			Items:    items,
		}
	}

	t.Run("Reorders items and returns refreshed issue", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", `{"item_ids":[3,1,2]}`)
		deps.Items.EXPECT().ReorderInIssue(gomock.Any(), int64(42), []int64{3, 1, 2}).Return(nil)
		deps.Issues.EXPECT().Find(gomock.Any(), int64(42)).Return(digest.Issue{
			ID:     42,
			Status: digest.IssueStatusDraft,
			Items: []news.Item{
				{ID: 3, Position: 0},
				{ID: 1, Position: 1},
				{ID: 2, Position: 2},
			},
		}, nil)

		err := deps.Handler.ReorderItems(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Empty item_ids returns 400", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", `{"item_ids":[]}`)
		_ = deps.Handler.ReorderItems(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Missing body returns 400", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", `not json`)
		_ = deps.Handler.ReorderItems(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Invalid id returns 400", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "abc", `{"item_ids":[1]}`)
		_ = deps.Handler.ReorderItems(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Mismatched ids return 404", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", `{"item_ids":[99]}`)
		deps.Items.EXPECT().ReorderInIssue(gomock.Any(), int64(42), []int64{99}).Return(store.ErrNotFound)

		_ = deps.Handler.ReorderItems(deps.Context)
		assert.Equal(t, http.StatusNotFound, deps.Recorder.Code)
	})

	t.Run("Non-draft returns 409", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", `{"item_ids":[1]}`)
		deps.Items.EXPECT().ReorderInIssue(gomock.Any(), int64(42), []int64{1}).Return(digest.ErrIssueNotDraft)

		_ = deps.Handler.ReorderItems(deps.Context)
		assert.Equal(t, http.StatusConflict, deps.Recorder.Code)
	})

	t.Run("Repo error returns 500", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", `{"item_ids":[1]}`)
		deps.Items.EXPECT().ReorderInIssue(gomock.Any(), int64(42), []int64{1}).Return(errors.New("db error"))

		_ = deps.Handler.ReorderItems(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
