// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/env"
	mockai "github.com/ainsleyclark/godaily/pkg/mocks/ai"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	mockslack "github.com/ainsleyclark/godaily/pkg/mocks/slack"
	mocksocial "github.com/ainsleyclark/godaily/pkg/mocks/social"
	socialsvc "github.com/ainsleyclark/godaily/pkg/services/social"
)

// newHandlerNoPosters builds a Handler with a real social.Service that has no posters configured.
func newHandlerNoPosters(t *testing.T) *Handler {
	t.Helper()

	ctrl := gomock.NewController(t)

	slackMock := mockslack.NewMockSender(ctrl)
	slackMock.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()

	prompter := mockai.NewMockPrompter(ctrl)
	issues := mockdigest.NewMockIssueRepository(ctrl)
	items := mocknews.NewMockItemRepository(ctrl)
	posts := mocksocial.NewMockPostRepository(ctrl)

	svc, err := socialsvc.New(env.Config{}, prompter, issues, items, posts, nil, slackMock)
	require.NoError(t, err)

	return &Handler{
		social: svc,
		slack:  slackMock,
		config: &env.Config{},
	}
}

func TestHandleFeatured(t *testing.T) {
	t.Parallel()

	t.Run("No posters configured short-circuits to OK", func(t *testing.T) {
		t.Parallel()

		h := newHandlerNoPosters(t)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/social/featured", nil)
		ctx := webkit.NewContext(rec, req)

		err := h.Featured(ctx)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandleRotation(t *testing.T) {
	t.Parallel()

	t.Run("No posters configured short-circuits to OK", func(t *testing.T) {
		t.Parallel()

		h := newHandlerNoPosters(t)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/social/rotation", nil)
		ctx := webkit.NewContext(rec, req)

		err := h.Rotation(ctx)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
