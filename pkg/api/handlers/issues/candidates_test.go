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
	"time"

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

func TestCandidates(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Issues   *mockdigest.MockIssueRepository
		Items    *mocknews.MockItemRepository
	}

	setup := func(t *testing.T, id string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		issues := mockdigest.NewMockIssueRepository(ctrl)
		items := mocknews.NewMockItemRepository(ctrl)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/issues/"+id+"/candidates", nil)
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

	t.Run("Returns unlinked items in the issue window", func(t *testing.T) {
		t.Parallel()

		// Tuesday — a single-day window ending at sent_at.
		sentAt := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
		deps := setup(t, "42")
		deps.Issues.EXPECT().Find(gomock.Any(), int64(42)).
			Return(digest.Issue{ID: 42, Status: digest.IssueStatusDraft, SentAt: sentAt}, nil)
		deps.Items.EXPECT().
			List(gomock.Any(), gomock.AssignableToTypeOf(news.ItemListOptions{})).
			DoAndReturn(func(_ context.Context, opts news.ItemListOptions) ([]news.Item, error) {
				// Only the raw pool (unlinked) within the build window is requested.
				assert.NotNil(t, opts.InDigest)
				assert.False(t, *opts.InDigest)
				assert.NotNil(t, opts.From)
				assert.NotNil(t, opts.To)
				assert.Equal(t, sentAt.AddDate(0, 0, -1), *opts.From)
				assert.Equal(t, sentAt, *opts.To)
				return []news.Item{{ID: 1, Title: "spare"}}, nil
			})

		err := deps.Handler.Candidates(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
		assert.Contains(t, deps.Recorder.Body.String(), "spare")
	})

	t.Run("Invalid id returns 400", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "abc")
		_ = deps.Handler.Candidates(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Issue not found returns 404", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42")
		deps.Issues.EXPECT().Find(gomock.Any(), int64(42)).Return(digest.Issue{}, store.ErrNotFound)

		_ = deps.Handler.Candidates(deps.Context)
		assert.Equal(t, http.StatusNotFound, deps.Recorder.Code)
	})

	t.Run("List error returns 500", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42")
		deps.Issues.EXPECT().Find(gomock.Any(), int64(42)).Return(digest.Issue{ID: 42}, nil)
		deps.Items.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))

		_ = deps.Handler.Candidates(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
