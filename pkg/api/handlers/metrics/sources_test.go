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

func TestHandleSources(t *testing.T) {
	tt := map[string]struct {
		mock       func(m *mockengagement.MockMetricsRepository)
		query      string
		wantStatus int
	}{
		"OK": {
			mock: func(m *mockengagement.MockMetricsRepository) {
				m.EXPECT().SourceList(gomock.Any(), gomock.Any()).Return([]engagement.SourceMetrics{
					{Source: "hn", Clicks: 220},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"Store error": {
			mock: func(m *mockengagement.MockMetricsRepository) {
				m.EXPECT().SourceList(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"Invalid query params": {
			mock:       func(m *mockengagement.MockMetricsRepository) {},
			query:      "to=bad-date",
			wantStatus: http.StatusBadRequest,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			metricsMock := mockengagement.NewMockMetricsRepository(ctrl)
			test.mock(metricsMock)

			h := &Handler{metricsRepo: metricsMock}

			target := "/metrics/sources"
			if test.query != "" {
				target += "?" + test.query
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, target, nil)
			invoke(h.Sources, w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
