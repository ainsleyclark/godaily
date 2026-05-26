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

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/mocks/digest"
	mockslack "github.com/ainsleyclark/godaily/pkg/mocks/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleSend(t *testing.T) {
	tt := map[string]struct {
		mock       func(r *mockdigest.MockService)
		weekend    bool
		secret     string
		wantStatus int
	}{
		"OK": {
			mock: func(r *mockdigest.MockService) {
				r.EXPECT().SendDigest(gomock.Any(), gomock.Any(), false).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		"Send Digest Error": {
			mock: func(r *mockdigest.MockService) {
				r.EXPECT().SendDigest(gomock.Any(), gomock.Any(), false).Return(errors.New("send failed"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"Unauthorized": {
			mock:       func(r *mockdigest.MockService) {},
			secret:     "supersecret",
			wantStatus: http.StatusUnauthorized,
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
				slack := mockslack.NewMockSender(ctrl)
				slack.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()
				test.mock(runner)

				a := &godaily.App{Runner: runner, Config: &env.Config{APISecret: test.secret}, Slack: slack}
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/digest/send", nil)
				r = r.WithContext(api.WithApp(r.Context(), a))
				HandleSend(w, r)
				assert.Equal(t, test.wantStatus, w.Code)
			})
		})
	}
}
