// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drafts

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/store"
)

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("Rejects non-positive id", func(t *testing.T) {
		t.Parallel()
		deps := newTest(t)
		ctx := deps.withRequest(http.MethodPatch, "/social/drafts/0", `{"text":"x"}`, map[string]string{"id": "0"})
		assertNoErr(t, deps.Handler.Update(ctx))
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Rejects empty text", func(t *testing.T) {
		t.Parallel()
		deps := newTest(t)
		ctx := deps.withRequest(http.MethodPatch, "/social/drafts/1", `{"text":"   "}`, map[string]string{"id": "1"})
		assertNoErr(t, deps.Handler.Update(ctx))
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Returns 404 when row missing", func(t *testing.T) {
		t.Parallel()
		deps := newTest(t)
		deps.Posts.EXPECT().Find(gomock.Any(), int64(1)).Return(social.Post{}, store.ErrNotFound)

		ctx := deps.withRequest(http.MethodPatch, "/social/drafts/1", `{"text":"new"}`, map[string]string{"id": "1"})
		assertNoErr(t, deps.Handler.Update(ctx))
		assert.Equal(t, http.StatusNotFound, deps.Recorder.Code)
	})

	t.Run("Returns 409 when row no longer a draft", func(t *testing.T) {
		t.Parallel()
		deps := newTest(t)
		deps.Posts.EXPECT().Find(gomock.Any(), int64(1)).Return(social.Post{ID: 1, Status: social.PostStatusPublished}, nil)

		ctx := deps.withRequest(http.MethodPatch, "/social/drafts/1", `{"text":"new"}`, map[string]string{"id": "1"})
		assertNoErr(t, deps.Handler.Update(ctx))
		assert.Equal(t, http.StatusConflict, deps.Recorder.Code)
	})

	t.Run("Happy path updates text", func(t *testing.T) {
		t.Parallel()
		deps := newTest(t)
		deps.Posts.EXPECT().Find(gomock.Any(), int64(1)).Return(social.Post{ID: 1, Status: social.PostStatusDraft}, nil)
		deps.Posts.EXPECT().
			Update(gomock.Any(), int64(1), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, u social.PostUpdate) (social.Post, error) {
				require.NotNil(t, u.Text)
				assert.Equal(t, "new body", *u.Text)
				assert.Nil(t, u.Status)
				return social.Post{ID: 1, Text: "new body", Status: social.PostStatusDraft}, nil
			})

		ctx := deps.withRequest(http.MethodPatch, "/social/drafts/1", `{"text":"new body"}`, map[string]string{"id": "1"})
		assertNoErr(t, deps.Handler.Update(ctx))
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
		assert.Contains(t, deps.Recorder.Body.String(), `"text":"new body"`)
	})
}
