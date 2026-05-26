// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/mocks/engagement"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleRoundup(t *testing.T) {
	tt := map[string]struct {
		mock       func(r *mockengagement.MockMetricsReporter)
		wantStatus int
	}{
		"OK": {
			mock: func(r *mockengagement.MockMetricsReporter) {
				r.EXPECT().Roundup(gomock.Any()).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		"Roundup error": {
			mock: func(r *mockengagement.MockMetricsReporter) {
				r.EXPECT().Roundup(gomock.Any()).Return(errors.New("boom"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			reporter := mockengagement.NewMockMetricsReporter(ctrl)
			test.mock(reporter)

			h := &Handler{metricsReporter: reporter}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/metrics/roundup", nil)
			invoke(h.Roundup, w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
