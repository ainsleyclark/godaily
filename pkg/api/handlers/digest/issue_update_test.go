// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

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
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
)

func TestUpdateIssue(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Issues   *mockdigest.MockIssueRepository
	}

	setup := func(t *testing.T, id, body string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		issues := mockdigest.NewMockIssueRepository(ctrl)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/digest/issues/"+id, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
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

	t.Run("Updates draft issue on success", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", `{"subject":"New title","summary":"New intro"}`)
		deps.Issues.EXPECT().
			Update(gomock.Any(), digest.Issue{ID: 42, Subject: "New title", Summary: "New intro"}).
			Return(digest.Issue{ID: 42, Subject: "New title", Summary: "New intro", Status: digest.IssueStatusDraft}, nil)

		err := deps.Handler.UpdateIssue(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Trims whitespace from subject and summary", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "7", `{"subject":"  Hello  ","summary":"  there  "}`)
		deps.Issues.EXPECT().
			Update(gomock.Any(), digest.Issue{ID: 7, Subject: "Hello", Summary: "there"}).
			Return(digest.Issue{ID: 7, Subject: "Hello", Summary: "there", Status: digest.IssueStatusDraft}, nil)

		err := deps.Handler.UpdateIssue(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Missing subject returns 400", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", `{"subject":"   ","summary":"x"}`)
		_ = deps.Handler.UpdateIssue(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Invalid JSON returns 400", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "42", `not json`)
		_ = deps.Handler.UpdateIssue(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Invalid id returns 400", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "abc", `{"subject":"x"}`)
		_ = deps.Handler.UpdateIssue(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Not found returns 404", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "99", `{"subject":"x"}`)
		deps.Issues.EXPECT().Update(gomock.Any(), gomock.Any()).Return(digest.Issue{}, store.ErrNotFound)

		_ = deps.Handler.UpdateIssue(deps.Context)
		assert.Equal(t, http.StatusNotFound, deps.Recorder.Code)
	})

	t.Run("Non-draft returns 409", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "5", `{"subject":"x"}`)
		deps.Issues.EXPECT().Update(gomock.Any(), gomock.Any()).Return(digest.Issue{}, digest.ErrIssueNotDraft)

		_ = deps.Handler.UpdateIssue(deps.Context)
		assert.Equal(t, http.StatusConflict, deps.Recorder.Code)
	})

	t.Run("Repo error returns 500", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "5", `{"subject":"x"}`)
		deps.Issues.EXPECT().Update(gomock.Any(), gomock.Any()).Return(digest.Issue{}, errors.New("db error"))

		_ = deps.Handler.UpdateIssue(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
