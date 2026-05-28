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

	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	mockaudience "github.com/ainsleyclark/godaily/pkg/mocks/audience"
	mockslack "github.com/ainsleyclark/godaily/pkg/mocks/slack"
)

func TestSubscribe(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler     *Handler
		Context     *webkit.Context
		Recorder    *httptest.ResponseRecorder
		Subscribers *mockaudience.MockSubscriberService
		Repo        *mockaudience.MockSubscriberRepository
		Slack       *mockslack.MockSender
	}

	setup := func(t *testing.T, body string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		svc := mockaudience.NewMockSubscriberService(ctrl)
		repo := mockaudience.NewMockSubscriberRepository(ctrl)
		slackMock := mockslack.NewMockSender(ctrl)
		slackMock.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/subscribe", strings.NewReader(body))

		return Test{
			Handler: &Handler{
				subscribers:     svc,
				subscribersRepo: repo,
				slack:           slackMock,
			},
			Context:     webkit.NewContext(rec, req),
			Recorder:    rec,
			Subscribers: svc,
			Repo:        repo,
			Slack:       slackMock,
		}
	}

	t.Run("Subscribes successfully", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, `{"email":"test@example.com"}`)
		deps.Subscribers.EXPECT().Subscribe(gomock.Any(), "test@example.com").Return(audience.Subscriber{}, nil)
		deps.Repo.EXPECT().CountActive(gomock.Any()).Return(int64(42), nil)

		err := deps.Handler.Subscribe(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Missing email returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, `{}`)

		_ = deps.Handler.Subscribe(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Invalid email returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, `{"email":"notanemail"}`)

		_ = deps.Handler.Subscribe(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Already subscribed returns conflict", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, `{"email":"dupe@example.com"}`)
		deps.Subscribers.EXPECT().Subscribe(gomock.Any(), "dupe@example.com").Return(audience.Subscriber{}, audience.ErrAlreadySubscribed)

		_ = deps.Handler.Subscribe(deps.Context)
		assert.Equal(t, http.StatusConflict, deps.Recorder.Code)
	})

	t.Run("Subscribe error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, `{"email":"err@example.com"}`)
		deps.Subscribers.EXPECT().Subscribe(gomock.Any(), "err@example.com").Return(audience.Subscriber{}, errors.New("db error"))

		_ = deps.Handler.Subscribe(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
