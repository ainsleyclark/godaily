// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/engagement"
)

func TestHandleRoundup(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Metrics  *mockengagement.MockMetricsService
	}

	setup := func(t *testing.T) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		reporter := mockengagement.NewMockMetricsService(ctrl)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/metrics/roundup", nil)

		return Test{
			Handler:  &Handler{metricsService: reporter},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Metrics:  reporter,
		}
	}

	t.Run("Runs roundup on success", func(t *testing.T) {
		t.Parallel()

		deps := setup(t)
		deps.Metrics.EXPECT().Roundup(gomock.Any()).Return(nil)

		err := deps.Handler.Roundup(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Roundup error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t)
		deps.Metrics.EXPECT().Roundup(gomock.Any()).Return(errors.New("boom"))

		_ = deps.Handler.Roundup(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
