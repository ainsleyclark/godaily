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

func TestHandleItems(t *testing.T) {
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

		target := "/metrics/items"
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

	t.Run("Returns item metrics on success", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Metrics.EXPECT().ItemList(gomock.Any(), gomock.Any()).Return([]engagement.ItemMetrics{
			{ItemID: 42, Title: "Go 1.24", Clicks: 18},
		}, nil)

		err := deps.Handler.Items(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Store error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Metrics.EXPECT().ItemList(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))

		_ = deps.Handler.Items(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})

	t.Run("Invalid from date returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "from=bad")

		_ = deps.Handler.Items(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Limit exceeds max returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "limit=999")

		_ = deps.Handler.Items(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})
}
