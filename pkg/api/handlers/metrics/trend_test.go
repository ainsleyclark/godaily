// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/mocks/engagement"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleTrend(t *testing.T) {
	tt := map[string]struct {
		mock       func(m *mockengagement.MockMetricsRepository)
		query      string
		wantStatus int
	}{
		"OK defaults": {
			mock: func(m *mockengagement.MockMetricsRepository) {
				m.EXPECT().Trend(gomock.Any(), gomock.Any(), "click_rate", "day").Return(engagement.TrendData{
					Metric: "click_rate",
					Bucket: "day",
					Points: []engagement.TrendPoint{},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"OK with metric and bucket": {
			mock: func(m *mockengagement.MockMetricsRepository) {
				m.EXPECT().Trend(gomock.Any(), gomock.Any(), "unique_opens", "week").Return(engagement.TrendData{
					Metric: "unique_opens",
					Bucket: "week",
					Points: []engagement.TrendPoint{},
				}, nil)
			},
			query:      "metric=unique_opens&bucket=week",
			wantStatus: http.StatusOK,
		},
		"Invalid metric": {
			mock:       func(m *mockengagement.MockMetricsRepository) {},
			query:      "metric=bad_metric",
			wantStatus: http.StatusBadRequest,
		},
		"Invalid bucket": {
			mock:       func(m *mockengagement.MockMetricsRepository) {},
			query:      "bucket=month",
			wantStatus: http.StatusBadRequest,
		},
		"Invalid query params": {
			mock:       func(m *mockengagement.MockMetricsRepository) {},
			query:      "from=not-a-date",
			wantStatus: http.StatusBadRequest,
		},
		"Store error": {
			mock: func(m *mockengagement.MockMetricsRepository) {
				m.EXPECT().Trend(gomock.Any(), gomock.Any(), "click_rate", "day").Return(engagement.TrendData{}, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			metricsMock := mockengagement.NewMockMetricsRepository(ctrl)
			test.mock(metricsMock)

			h := &Handler{metricsRepo: metricsMock}

			target := "/metrics/trend"
			if test.query != "" {
				target += "?" + test.query
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, target, nil)
			invoke(h.Trend, w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
