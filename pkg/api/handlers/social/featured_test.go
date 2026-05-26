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

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/mocks/ai"
	"github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/pkg/mocks/slack"
	"github.com/ainsleyclark/godaily/pkg/mocks/social"
	socialsvc "github.com/ainsleyclark/godaily/pkg/services/social"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newHandlerNoPosters builds a Handler with a real social.Service that has no posters configured.
func newHandlerNoPosters(t *testing.T) *Handler {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	slackMock := mockslack.NewMockSender(ctrl)
	slackMock.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()

	prompter := mockai.NewMockPrompter(ctrl)
	issues := mockdigest.NewMockIssueRepository(ctrl)
	items := mocknews.NewMockItemRepository(ctrl)
	posts := mocksocial.NewMockPostRepository(ctrl)

	svc, err := socialsvc.New(nil, prompter, issues, items, posts, slackMock)
	require.NoError(t, err)

	return &Handler{
		social: svc,
		slack:  slackMock,
		config: &env.Config{},
	}
}

func TestHandleFeatured(t *testing.T) {
	t.Run("No posters configured short-circuits to OK", func(t *testing.T) {
		h := newHandlerNoPosters(t)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/social/featured", nil)
		invoke(h.Featured, w, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandleRotation(t *testing.T) {
	t.Run("No posters configured short-circuits to OK", func(t *testing.T) {
		h := newHandlerNoPosters(t)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/social/rotation", nil)
		invoke(h.Rotation, w, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
