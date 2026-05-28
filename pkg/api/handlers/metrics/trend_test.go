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

func TestHandleTrend(t *testing.T) {
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

		target := "/metrics/trend"
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

	t.Run("Returns trend with defaults on success", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Metrics.EXPECT().Trend(gomock.Any(), gomock.Any(), "click_rate", "day").Return(engagement.TrendData{
			Metric: "click_rate",
			Bucket: "day",
			Points: []engagement.TrendPoint{},
		}, nil)

		err := deps.Handler.Trend(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Returns trend with metric and bucket", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "metric=unique_opens&bucket=week")
		deps.Metrics.EXPECT().Trend(gomock.Any(), gomock.Any(), "unique_opens", "week").Return(engagement.TrendData{
			Metric: "unique_opens",
			Bucket: "week",
			Points: []engagement.TrendPoint{},
		}, nil)

		err := deps.Handler.Trend(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Invalid metric returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "metric=bad_metric")

		_ = deps.Handler.Trend(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Invalid bucket returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "bucket=month")

		_ = deps.Handler.Trend(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Invalid query params returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "from=not-a-date")

		_ = deps.Handler.Trend(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Store error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Metrics.EXPECT().Trend(gomock.Any(), gomock.Any(), "click_rate", "day").Return(engagement.TrendData{}, errors.New("db error"))

		_ = deps.Handler.Trend(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
