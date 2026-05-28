// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/env"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	mockslack "github.com/ainsleyclark/godaily/pkg/mocks/slack"
)

func TestCollect(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Runner   *mockdigest.MockService
		Slack    *mockslack.MockSender
	}

	setup := func(t *testing.T, req *http.Request) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		runner := mockdigest.NewMockService(ctrl)
		slackMock := mockslack.NewMockSender(ctrl)
		slackMock.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()
		rec := httptest.NewRecorder()

		return Test{
			Handler:  &Handler{runner: runner, config: &env.Config{}, slack: slackMock},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Runner:   runner,
			Slack:    slackMock,
		}
	}

	t.Run("Collects successfully on a weekday", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			// Fake clock starts on Saturday 2000-01-01; advance to Monday.
			time.Sleep(48 * time.Hour)

			req := httptest.NewRequest(http.MethodGet, "/digest/collect", nil)
			deps := setup(t, req)
			deps.Runner.EXPECT().Collect(gomock.Any(), gomock.Any()).Return(digest.CollectResponse{}, nil)

			err := deps.Handler.Collect(deps.Context)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, deps.Recorder.Code)
		})
	})

	t.Run("Collect error returns internal server error", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			time.Sleep(48 * time.Hour)

			req := httptest.NewRequest(http.MethodGet, "/digest/collect", nil)
			deps := setup(t, req)
			deps.Runner.EXPECT().Collect(gomock.Any(), gomock.Any()).Return(digest.CollectResponse{}, errors.New("boom"))

			_ = deps.Handler.Collect(deps.Context)

			assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
		})
	})

	t.Run("Skips collect on weekend", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/digest/collect", nil)
			deps := setup(t, req)

			err := deps.Handler.Collect(deps.Context)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, deps.Recorder.Code)
		})
	})
}
