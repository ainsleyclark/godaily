// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drafts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	mocksocial "github.com/ainsleyclark/godaily/pkg/mocks/social"
	"github.com/ainsleyclark/godaily/pkg/store"
)

func newHandler(t *testing.T) (*Handler, *mocksocial.MockPostRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	posts := mocksocial.NewMockPostRepository(ctrl)
	return &Handler{socialPosts: posts}, posts
}

// newWebkitContext returns a webkit.Context with the given chi url
// params and JSON body. Path params are wired through chi's
// RouteContext, matching the pattern used by the issues handler tests.
func newWebkitContext(method, path, body string, params map[string]string) (*webkit.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if len(params) > 0 {
		rctx := chi.NewRouteContext()
		for k, v := range params {
			rctx.URLParams.Add(k, v)
		}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	return webkit.NewContext(rec, req), rec
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("Returns drafts filtered by status", func(t *testing.T) {
		t.Parallel()
		h, posts := newHandler(t)

		posts.EXPECT().
			List(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, opts social.PostListOptions) ([]social.Post, error) {
				require.NotNil(t, opts.Status)
				assert.Equal(t, social.PostStatusDraft, *opts.Status)
				return []social.Post{{ID: 1, Text: "hi", Status: social.PostStatusDraft}}, nil
			})

		c, rec := newWebkitContext(http.MethodGet, "/social/drafts", "", nil)
		require.NoError(t, h.List(c))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"text":"hi"`)
	})
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("Rejects non-positive id", func(t *testing.T) {
		t.Parallel()
		h, _ := newHandler(t)
		c, rec := newWebkitContext(http.MethodPatch, "/social/drafts/0", `{"text":"x"}`, map[string]string{"id": "0"})
		require.NoError(t, h.Update(c))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Rejects empty text", func(t *testing.T) {
		t.Parallel()
		h, _ := newHandler(t)
		c, rec := newWebkitContext(http.MethodPatch, "/social/drafts/1", `{"text":"   "}`, map[string]string{"id": "1"})
		require.NoError(t, h.Update(c))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Returns 404 when row missing", func(t *testing.T) {
		t.Parallel()
		h, posts := newHandler(t)
		posts.EXPECT().Find(gomock.Any(), int64(1)).Return(social.Post{}, store.ErrNotFound)

		c, rec := newWebkitContext(http.MethodPatch, "/social/drafts/1", `{"text":"new"}`, map[string]string{"id": "1"})
		require.NoError(t, h.Update(c))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("Returns 409 when row no longer a draft", func(t *testing.T) {
		t.Parallel()
		h, posts := newHandler(t)
		posts.EXPECT().Find(gomock.Any(), int64(1)).Return(social.Post{ID: 1, Status: social.PostStatusPublished}, nil)

		c, rec := newWebkitContext(http.MethodPatch, "/social/drafts/1", `{"text":"new"}`, map[string]string{"id": "1"})
		require.NoError(t, h.Update(c))
		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("Happy path updates text", func(t *testing.T) {
		t.Parallel()
		h, posts := newHandler(t)
		posts.EXPECT().Find(gomock.Any(), int64(1)).Return(social.Post{ID: 1, Status: social.PostStatusDraft}, nil)
		posts.EXPECT().
			Update(gomock.Any(), int64(1), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, u social.PostUpdate) (social.Post, error) {
				require.NotNil(t, u.Text)
				assert.Equal(t, "new body", *u.Text)
				assert.Nil(t, u.Status)
				return social.Post{ID: 1, Text: "new body", Status: social.PostStatusDraft}, nil
			})

		c, rec := newWebkitContext(http.MethodPatch, "/social/drafts/1", `{"text":"new body"}`, map[string]string{"id": "1"})
		require.NoError(t, h.Update(c))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"text":"new body"`)
	})
}

func TestCancel(t *testing.T) {
	t.Parallel()

	t.Run("Returns 409 when row already published", func(t *testing.T) {
		t.Parallel()
		h, posts := newHandler(t)
		posts.EXPECT().Find(gomock.Any(), int64(1)).Return(social.Post{ID: 1, Status: social.PostStatusPublished}, nil)

		c, rec := newWebkitContext(http.MethodPost, "/social/drafts/1/cancel", "", map[string]string{"id": "1"})
		require.NoError(t, h.Cancel(c))
		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("Happy path transitions to cancelled", func(t *testing.T) {
		t.Parallel()
		h, posts := newHandler(t)
		posts.EXPECT().Find(gomock.Any(), int64(7)).Return(social.Post{ID: 7, Status: social.PostStatusDraft}, nil)
		posts.EXPECT().
			Update(gomock.Any(), int64(7), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, u social.PostUpdate) (social.Post, error) {
				require.NotNil(t, u.Status)
				assert.Equal(t, social.PostStatusCancelled, *u.Status)
				return social.Post{ID: 7, Status: social.PostStatusCancelled}, nil
			})

		c, rec := newWebkitContext(http.MethodPost, "/social/drafts/7/cancel", "", map[string]string{"id": "7"})
		require.NoError(t, h.Cancel(c))
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
