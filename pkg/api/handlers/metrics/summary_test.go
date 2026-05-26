// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package metrics

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/env"
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

			a := &godaily.App{
				Config: &env.Config{},
				Repository: &godaily.Repository{
					Metrics: metricsMock,
				},
			}

			target := "/metrics/summary"
			if test.query != "" {
				target += "?" + test.query
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, target, nil)
			r = r.WithContext(api.WithApp(r.Context(), a))

			HandleSummary(w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
