// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mockaudience "github.com/ainsleyclark/godaily/pkg/mocks/audience"
)

func TestUnsubscribe(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler     *Handler
		Context     *webkit.Context
		Recorder    *httptest.ResponseRecorder
		Subscribers *mockaudience.MockSubscriberService
	}

	setup := func(t *testing.T, method, token string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		svc := mockaudience.NewMockSubscriberService(ctrl)
		rec := httptest.NewRecorder()

		url := "/unsubscribe"
		if token != "" {
			url += "?token=" + token
		}
		req := httptest.NewRequest(method, url, nil)

		return Test{
			Handler:     &Handler{subscribers: svc},
			Context:     webkit.NewContext(rec, req),
			Recorder:    rec,
			Subscribers: svc,
		}
	}

	t.Run("Redirects on successful GET unsubscribe", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, http.MethodGet, "valid-token")
		deps.Subscribers.EXPECT().Unsubscribe(gomock.Any(), "valid-token").Return(nil)

		err := deps.Handler.Unsubscribe(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusFound, deps.Recorder.Code)
		assert.Equal(t, "/unsubscribed/", deps.Recorder.Header().Get("Location"))
	})

	t.Run("Missing token returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, http.MethodGet, "")

		_ = deps.Handler.Unsubscribe(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Unsubscribe error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, http.MethodGet, "bad-token")
		deps.Subscribers.EXPECT().Unsubscribe(gomock.Any(), "bad-token").Return(errors.New("db error"))

		_ = deps.Handler.Unsubscribe(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})

	t.Run("POST unsubscribe returns OK without redirect", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, http.MethodPost, "valid-token")
		deps.Subscribers.EXPECT().Unsubscribe(gomock.Any(), "valid-token").Return(nil)

		err := deps.Handler.Unsubscribe(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
		assert.Empty(t, deps.Recorder.Header().Get("Location"))
	})
}
