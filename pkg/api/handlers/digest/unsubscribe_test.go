// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/mocks/audience"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUnsubscribe(t *testing.T) {
	tt := map[string]struct {
		token        string
		mock         func(s *mockaudience.MockSubscriberService)
		wantStatus   int
		wantLocation string
	}{
		"OK": {
			token: "valid-token",
			mock: func(s *mockaudience.MockSubscriberService) {
				s.EXPECT().Unsubscribe(gomock.Any(), "valid-token").Return(nil)
			},
			wantStatus:   http.StatusFound,
			wantLocation: "/unsubscribed/",
		},
		"Missing Token": {
			token:      "",
			mock:       func(s *mockaudience.MockSubscriberService) {},
			wantStatus: http.StatusBadRequest,
		},
		"Unsubscribe Error": {
			token: "bad-token",
			mock: func(s *mockaudience.MockSubscriberService) {
				s.EXPECT().Unsubscribe(gomock.Any(), "bad-token").Return(errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc := mockaudience.NewMockSubscriberService(ctrl)
			test.mock(svc)

			h := &Handler{subscribers: svc}

			url := "/unsubscribe"
			if test.token != "" {
				url += "?token=" + test.token
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, url, nil)
			invoke(h.Unsubscribe, w, r)

			assert.Equal(t, test.wantStatus, w.Code)
			if test.wantLocation != "" {
				assert.Equal(t, test.wantLocation, w.Header().Get("Location"))
			}
		})
	}
}

func TestUnsubscribePost(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mockaudience.NewMockSubscriberService(ctrl)
	svc.EXPECT().Unsubscribe(gomock.Any(), "valid-token").Return(nil)

	h := &Handler{subscribers: svc}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/unsubscribe?token=valid-token", nil)
	invoke(h.Unsubscribe, w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Location"))
}
