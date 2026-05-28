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

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/engagement"
)

func TestHandleSummary(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Metrics  *mockengagement.MockMetricsRepository
	}

	setup := func(t *testing.T, query string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		metricsMock := mockengagement.NewMockMetricsRepository(ctrl)
		rec := httptest.NewRecorder()

		target := "/metrics/summary"
		if query != "" {
			target += "?" + query
		}
		req := httptest.NewRequest(http.MethodGet, target, nil)

		return Test{
			Handler:  &Handler{metricsRepo: metricsMock},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Metrics:  metricsMock,
		}
	}

	t.Run("Returns summary on success", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Metrics.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(engagement.SummaryStats{
			IssuesSent: 5,
			Delivered:  100,
		}, nil)

		err := deps.Handler.Summary(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Store error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Metrics.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(engagement.SummaryStats{}, errors.New("db error"))

		_ = deps.Handler.Summary(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})

	t.Run("Invalid query params returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "from=not-a-date")

		_ = deps.Handler.Summary(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})
}
