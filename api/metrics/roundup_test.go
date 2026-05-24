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
	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/domain/engagement"
	mockslack "github.com/ainsleyclark/godaily/pkg/mocks/slack"
	metricssvc "github.com/ainsleyclark/godaily/pkg/services/metrics"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleRoundup(t *testing.T) {
	tt := map[string]struct {
		mock       func(m *mockengagement.MockMetricsRepository, s *mockslack.MockSender)
		wantStatus int
	}{
		"OK": {
			mock: func(m *mockengagement.MockMetricsRepository, s *mockslack.MockSender) {
				m.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(engagement.SummaryStats{}, nil).Times(2)
				m.EXPECT().SubscriberGrowth(gomock.Any(), gomock.Any(), "week").Return(engagement.SubscriberData{}, nil).Times(2)
				m.EXPECT().ItemList(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)
				m.EXPECT().TagList(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)
				m.EXPECT().SourceList(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)
				m.EXPECT().IssueList(gomock.Any(), gomock.Any(), "click_rate").Return(nil, nil).Times(2)
				s.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		"Gather error": {
			mock: func(m *mockengagement.MockMetricsRepository, _ *mockslack.MockSender) {
				m.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(engagement.SummaryStats{}, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"Slack error": {
			mock: func(m *mockengagement.MockMetricsRepository, s *mockslack.MockSender) {
				m.EXPECT().Summary(gomock.Any(), gomock.Any()).Return(engagement.SummaryStats{}, nil).Times(2)
				m.EXPECT().SubscriberGrowth(gomock.Any(), gomock.Any(), "week").Return(engagement.SubscriberData{}, nil).Times(2)
				m.EXPECT().ItemList(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)
				m.EXPECT().TagList(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)
				m.EXPECT().SourceList(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)
				m.EXPECT().IssueList(gomock.Any(), gomock.Any(), "click_rate").Return(nil, nil).Times(2)
				s.EXPECT().Send(gomock.Any(), gomock.Any()).Return(errors.New("slack down"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			metricsMock := mockengagement.NewMockMetricsRepository(ctrl)
			slackMock := mockslack.NewMockSender(ctrl)
			test.mock(metricsMock, slackMock)

			a := &godaily.App{
				Config:         &env.Config{},
				Repository:     &godaily.Repository{Metrics: metricsMock},
				MetricsService: metricssvc.New(metricsMock, slackMock),
			}
			api.SetApp(a)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/metrics/roundup", nil)
			r.RemoteAddr = "1.2.3.4:1234"

			HandleRoundup(w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
