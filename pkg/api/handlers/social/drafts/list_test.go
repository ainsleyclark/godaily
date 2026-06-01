// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drafts

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("Returns drafts filtered by status", func(t *testing.T) {
		t.Parallel()

		deps := newTest(t)
		deps.Posts.EXPECT().
			List(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, opts social.PostListOptions) ([]social.Post, error) {
				require.NotNil(t, opts.Status)
				assert.Equal(t, social.PostStatusDraft, *opts.Status)
				return []social.Post{{ID: 1, Text: "hi", Status: social.PostStatusDraft}}, nil
			})

		ctx := deps.withRequest(http.MethodGet, "/social/drafts", "", nil)
		assertNoErr(t, deps.Handler.List(ctx))
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
		assert.Contains(t, deps.Recorder.Body.String(), `"text":"hi"`)
	})

	t.Run("Repository error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := newTest(t)
		deps.Posts.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("db down"))

		ctx := deps.withRequest(http.MethodGet, "/social/drafts", "", nil)
		_ = deps.Handler.List(ctx)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
