// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package social

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/env"
	mockai "github.com/ainsleyclark/godaily/pkg/mocks/ai"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	mocksocial "github.com/ainsleyclark/godaily/pkg/mocks/social"
	mockslack "github.com/ainsleyclark/godaily/pkg/mocks/slack"
	socialsvc "github.com/ainsleyclark/godaily/pkg/services/social"
)

// newAppNoPosters builds a real social.Service with no posters configured.
// This is the most stable path to test through the featured handler
// without standing up a full integration harness.
func newAppNoPosters(t *testing.T, secret string) *godaily.App {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	slack := mockslack.NewMockSender(ctrl)
	slack.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()

	prompter := mockai.NewMockPrompter(ctrl)
	issues := mocknews.NewMockIssueRepository(ctrl)
	items := mocknews.NewMockItemRepository(ctrl)
	posts := mocksocial.NewMockPostRepository(ctrl)

	svc, err := socialsvc.New(nil, prompter, issues, items, posts, slack)
	require.NoError(t, err)

	return &godaily.App{
		Config: &env.Config{APISecret: secret},
		Slack:  slack,
		Social: svc,
	}
}

func TestHandleFeatured(t *testing.T) {
	t.Run("Unauthorized request is rejected", func(t *testing.T) {
		a := newAppNoPosters(t, "supersecret")
		api.SetApp(a)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/social/featured", nil)
		r.Header.Set("Authorization", "Bearer wrong")
		HandleFeatured(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("No posters configured short-circuits to OK", func(t *testing.T) {
		// HasPosters is false, so the handler must return OK without
		// invoking Post. Real wall clock is fine — both the weekend
		// path and the no-posters path return OK, and the slot check is
		// bypassed by the no-posters branch.
		a := newAppNoPosters(t, "")
		api.SetApp(a)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/social/featured", nil)
		HandleFeatured(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandleRotation(t *testing.T) {
	t.Run("Unauthorized request is rejected", func(t *testing.T) {
		a := newAppNoPosters(t, "supersecret")
		api.SetApp(a)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/social/rotation", nil)
		r.Header.Set("Authorization", "Bearer wrong")
		HandleRotation(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("No posters configured short-circuits to OK", func(t *testing.T) {
		a := newAppNoPosters(t, "")
		api.SetApp(a)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/social/rotation", nil)
		HandleRotation(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
