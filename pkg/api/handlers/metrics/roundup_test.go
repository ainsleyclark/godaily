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
	"github.com/ainsleyclark/godaily/pkg/env"
	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/engagement"
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

			a := &godaily.App{
				Config:         &env.Config{},
				MetricsService: reporter,
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/metrics/roundup", nil)
			r = r.WithContext(api.WithApp(r.Context(), a))

			HandleRoundup(w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
