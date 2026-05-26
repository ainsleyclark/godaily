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

package digest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestBuild(t *testing.T) {
	tt := map[string]struct {
		mock       func(r *mockdigest.MockService)
		weekend    bool
		wantStatus int
	}{
		"OK": {
			mock: func(r *mockdigest.MockService) {
				r.EXPECT().Build(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		"Build Error": {
			mock: func(r *mockdigest.MockService) {
				r.EXPECT().Build(gomock.Any(), gomock.Any()).Return(errors.New("boom"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"Weekend": {
			mock:       func(r *mockdigest.MockService) {},
			weekend:    true,
			wantStatus: http.StatusOK,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				if !test.weekend {
					// Fake clock starts on Saturday 2000-01-01; advance to Monday.
					time.Sleep(48 * time.Hour)
				}

				ctrl := gomock.NewController(t)
				runner := mockdigest.NewMockService(ctrl)
				test.mock(runner)

				h := &Handler{runner: runner, config: &env.Config{}}
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/digest/build", nil)
				invoke(h.Build, w, r)
				assert.Equal(t, test.wantStatus, w.Code)
			})
		})
	}
}
