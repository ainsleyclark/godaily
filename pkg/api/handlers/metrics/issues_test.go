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

func TestHandleIssues(t *testing.T) {
	tt := map[string]struct {
		mock       func(m *mockengagement.MockMetricsRepository)
		query      string
		wantStatus int
	}{
		"OK": {
			mock: func(m *mockengagement.MockMetricsRepository) {
				m.EXPECT().IssueList(gomock.Any(), gomock.Any(), "sent_at").Return([]engagement.IssueEngagement{
					{IssueID: 1, Slug: "2026-05-22"},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"OK with sort": {
			mock: func(m *mockengagement.MockMetricsRepository) {
				m.EXPECT().IssueList(gomock.Any(), gomock.Any(), "click_rate").Return([]engagement.IssueEngagement{}, nil)
			},
			query:      "sort=click_rate",
			wantStatus: http.StatusOK,
		},
		"Store error": {
			mock: func(m *mockengagement.MockMetricsRepository) {
				m.EXPECT().IssueList(gomock.Any(), gomock.Any(), "sent_at").Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"Invalid query params": {
			mock:       func(m *mockengagement.MockMetricsRepository) {},
			query:      "from=bad",
			wantStatus: http.StatusBadRequest,
		},
		"Unknown sort": {
			mock:       func(m *mockengagement.MockMetricsRepository) {},
			query:      "sort=bad_sort",
			wantStatus: http.StatusBadRequest,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			metricsMock := mockengagement.NewMockMetricsRepository(ctrl)
			test.mock(metricsMock)

			h := &Handler{metricsRepo: metricsMock}

			target := "/metrics/issues"
			if test.query != "" {
				target += "?" + test.query
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, target, nil)
			invoke(h.Issues, w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
