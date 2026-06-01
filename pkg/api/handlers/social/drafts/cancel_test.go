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
)

func TestCancel(t *testing.T) {
	t.Parallel()

	t.Run("Rejects non-positive id", func(t *testing.T) {
		t.Parallel()
		deps := newTest(t)
		ctx := deps.withRequest(http.MethodPost, "/social/drafts/0/cancel", "", map[string]string{"id": "0"})
		assertNoErr(t, deps.Handler.Cancel(ctx))
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Returns 409 when row already published", func(t *testing.T) {
		t.Parallel()
		deps := newTest(t)
		deps.Posts.EXPECT().Find(gomock.Any(), int64(1)).Return(social.Post{ID: 1, Status: social.PostStatusPublished}, nil)

		ctx := deps.withRequest(http.MethodPost, "/social/drafts/1/cancel", "", map[string]string{"id": "1"})
		assertNoErr(t, deps.Handler.Cancel(ctx))
		assert.Equal(t, http.StatusConflict, deps.Recorder.Code)
	})

	t.Run("Happy path transitions to cancelled", func(t *testing.T) {
		t.Parallel()
		deps := newTest(t)
		deps.Posts.EXPECT().Find(gomock.Any(), int64(7)).Return(social.Post{ID: 7, Status: social.PostStatusDraft}, nil)
		deps.Posts.EXPECT().
			Update(gomock.Any(), int64(7), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, u social.PostUpdate) (social.Post, error) {
				require.NotNil(t, u.Status)
				assert.Equal(t, social.PostStatusCancelled, *u.Status)
				return social.Post{ID: 7, Status: social.PostStatusCancelled}, nil
			})

		ctx := deps.withRequest(http.MethodPost, "/social/drafts/7/cancel", "", map[string]string{"id": "7"})
		assertNoErr(t, deps.Handler.Cancel(ctx))
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})
}
