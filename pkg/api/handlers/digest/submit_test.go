// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	mockslack "github.com/ainsleyclark/godaily/pkg/mocks/slack"
)

func TestSubmitReddit(t *testing.T) {
	t.Parallel()

	// A minimal but valid r/golang listing payload.
	const redditJSON = `{"data":{"children":[
		{"data":{"title":"A Go post","url":"https://example.com/post","author":"gopher","score":42,"num_comments":3,"created_utc":1748000000,"permalink":"/r/golang/comments/x/a_go_post/"}}
	]}}`

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Runner   *mockdigest.MockService
	}

	setup := func(t *testing.T, body string) Test {
		t.Helper()
		ctrl := gomock.NewController(t)
		runner := mockdigest.NewMockService(ctrl)
		slackMock := mockslack.NewMockSender(ctrl)
		slackMock.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/digest/submit-reddit", strings.NewReader(body))
		return Test{
			Handler:  &Handler{runner: runner, config: &env.Config{}, slack: slackMock},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Runner:   runner,
		}
	}

	t.Run("Submits valid payload", func(t *testing.T) {
		t.Parallel()
		deps := setup(t, redditJSON)
		deps.Runner.EXPECT().
			Submit(gomock.Any(), news.SourceReddit, gomock.Any()).
			Return(digest.SubmitResponse{Received: 1, Persisted: 1}, nil)

		err := deps.Handler.SubmitReddit(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
		assert.Contains(t, deps.Recorder.Body.String(), "Successfully submitted")
	})

	t.Run("Empty body is rejected", func(t *testing.T) {
		t.Parallel()
		deps := setup(t, "")

		err := deps.Handler.SubmitReddit(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Invalid JSON is rejected", func(t *testing.T) {
		t.Parallel()
		deps := setup(t, "not json")

		err := deps.Handler.SubmitReddit(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Skipped submission reports skipped", func(t *testing.T) {
		t.Parallel()
		deps := setup(t, redditJSON)
		deps.Runner.EXPECT().
			Submit(gomock.Any(), news.SourceReddit, gomock.Any()).
			Return(digest.SubmitResponse{Received: 1, Skipped: true}, nil)

		err := deps.Handler.SubmitReddit(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
		assert.Contains(t, deps.Recorder.Body.String(), "skipped")
	})

	t.Run("Service error returns internal server error", func(t *testing.T) {
		t.Parallel()
		deps := setup(t, redditJSON)
		deps.Runner.EXPECT().
			Submit(gomock.Any(), news.SourceReddit, gomock.Any()).
			Return(digest.SubmitResponse{}, errors.New("boom"))

		_ = deps.Handler.SubmitReddit(deps.Context)

		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
