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

func TestHandleSummary(t *testing.T) {
	tt := map[string]struct {
		mock       func(m *mockengagement.MockMetricsRepository)
		query      string
		wantStatus int
	}{
		"OK": {
			mock: func(m *mockengagement.MockMetricsRepository) {
				m.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(engagement.SummaryStats{
					IssuesSent: 5,
					Delivered:  100,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"Store error": {
			mock: func(m *mockengagement.MockMetricsRepository) {
				m.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(engagement.SummaryStats{}, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"Invalid query params": {
			mock:       func(m *mockengagement.MockMetricsRepository) {},
			query:      "from=not-a-date",
			wantStatus: http.StatusBadRequest,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			metricsMock := mockengagement.NewMockMetricsRepository(ctrl)
			test.mock(metricsMock)

			h := &Handler{metricsRepo: metricsMock}

			target := "/metrics/summary"
			if test.query != "" {
				target += "?" + test.query
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, target, nil)
			invoke(h.Summary, w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
