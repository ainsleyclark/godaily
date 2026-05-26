// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/ainsleyclark/godaily/pkg/mocks/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCollect(t *testing.T) {
	tt := map[string]struct {
		mock       func(r *mockdigest.MockService)
		weekend    bool
		wantStatus int
	}{
		"OK": {
			mock: func(r *mockdigest.MockService) {
				r.EXPECT().Collect(gomock.Any(), gomock.Any()).Return(digest.CollectResponse{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"Collect Error": {
			mock: func(r *mockdigest.MockService) {
				r.EXPECT().Collect(gomock.Any(), gomock.Any()).Return(digest.CollectResponse{}, errors.New("boom"))
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
				slackMock := mockslack.NewMockSender(ctrl)
				slackMock.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()
				test.mock(runner)

				h := &Handler{runner: runner, config: &env.Config{}, slack: slackMock}
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/digest/collect", nil)
				invoke(h.Collect, w, r)
				assert.Equal(t, test.wantStatus, w.Code)
			})
		})
	}
}
