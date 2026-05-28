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

	"github.com/ainsleyclark/godaily/pkg/env"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
)

func TestBuild(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Runner   *mockdigest.MockService
	}

	setup := func(t *testing.T, req *http.Request) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		runner := mockdigest.NewMockService(ctrl)
		rec := httptest.NewRecorder()

		return Test{
			Handler:  &Handler{runner: runner, config: &env.Config{}},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Runner:   runner,
		}
	}

	t.Run("Builds successfully on a weekday", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			// Fake clock starts on Saturday 2000-01-01; advance to Monday.
			time.Sleep(48 * time.Hour)

			req := httptest.NewRequest(http.MethodGet, "/digest/build", nil)
			deps := setup(t, req)
			deps.Runner.EXPECT().Build(gomock.Any(), gomock.Any()).Return(nil)

			err := deps.Handler.Build(deps.Context)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, deps.Recorder.Code)
		})
	})

	t.Run("Build error returns internal server error", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			time.Sleep(48 * time.Hour)

			req := httptest.NewRequest(http.MethodGet, "/digest/build", nil)
			deps := setup(t, req)
			deps.Runner.EXPECT().Build(gomock.Any(), gomock.Any()).Return(errors.New("boom"))

			_ = deps.Handler.Build(deps.Context)

			assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
		})
	})

	t.Run("Skips build on weekend", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/digest/build", nil)
			deps := setup(t, req)

			err := deps.Handler.Build(deps.Context)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, deps.Recorder.Code)
		})
	})
}
