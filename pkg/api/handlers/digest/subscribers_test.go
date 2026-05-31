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

	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	mockaudience "github.com/ainsleyclark/godaily/pkg/mocks/audience"
	"github.com/ainsleyclark/godaily/pkg/store"
)

func TestSubscribers(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler     *Handler
		Context     *webkit.Context
		Recorder    *httptest.ResponseRecorder
		Subscribers *mockaudience.MockSubscriberRepository
	}

	setup := func(t *testing.T, query string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		subs := mockaudience.NewMockSubscriberRepository(ctrl)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/digest/subscribers"+query, nil)

		return Test{
			Handler:     &Handler{subscribersRepo: subs},
			Context:     webkit.NewContext(rec, req),
			Recorder:    rec,
			Subscribers: subs,
		}
	}

	t.Run("Returns subscribers on default pagination", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Subscribers.EXPECT().CountFiltered(gomock.Any(), "").Return(int64(3), nil)
		deps.Subscribers.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]audience.Subscriber{
			{ID: 1, Email: "a@example.com"},
			{ID: 2, Email: "b@example.com"},
		}, nil)

		err := deps.Handler.Subscribers(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Returns subscribers with explicit page params", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "?page=2&per_page=5")
		deps.Subscribers.EXPECT().CountFiltered(gomock.Any(), "").Return(int64(50), nil)
		deps.Subscribers.EXPECT().List(gomock.Any(), store.ListOptions{Page: 2, PerPage: 5}).Return([]audience.Subscriber{}, nil)

		err := deps.Handler.Subscribers(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("CountFiltered error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Subscribers.EXPECT().CountFiltered(gomock.Any(), "").Return(int64(0), errors.New("db error"))

		_ = deps.Handler.Subscribers(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})

	t.Run("List error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Subscribers.EXPECT().CountFiltered(gomock.Any(), "").Return(int64(1), nil)
		deps.Subscribers.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))

		_ = deps.Handler.Subscribers(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})

	t.Run("Invalid page falls back to default", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "?page=abc")
		deps.Subscribers.EXPECT().CountFiltered(gomock.Any(), "").Return(int64(1), nil)
		deps.Subscribers.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]audience.Subscriber{}, nil)

		err := deps.Handler.Subscribers(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Per page exceeds max falls back to default", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "?per_page=999")
		deps.Subscribers.EXPECT().CountFiltered(gomock.Any(), "").Return(int64(1), nil)
		deps.Subscribers.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]audience.Subscriber{}, nil)

		err := deps.Handler.Subscribers(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})
}
