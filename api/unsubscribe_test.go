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

package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/env"
	mocksubscriber "github.com/ainsleyclark/godaily/pkg/mocks/subscriber"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleUnsubscribe(t *testing.T) {
	tt := map[string]struct {
		token        string
		mock         func(s *mocksubscriber.MockSubscriber)
		wantStatus   int
		wantLocation string
	}{
		"OK": {
			token: "valid-token",
			mock: func(s *mocksubscriber.MockSubscriber) {
				s.EXPECT().Unsubscribe(gomock.Any(), "valid-token").Return(nil)
			},
			wantStatus:   http.StatusFound,
			wantLocation: "/unsubscribed/",
		},
		"Missing Token": {
			token:      "",
			mock:       func(s *mocksubscriber.MockSubscriber) {},
			wantStatus: http.StatusBadRequest,
		},
		"Unsubscribe Error": {
			token: "bad-token",
			mock: func(s *mocksubscriber.MockSubscriber) {
				s.EXPECT().Unsubscribe(gomock.Any(), "bad-token").Return(errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc := mocksubscriber.NewMockSubscriber(ctrl)
			test.mock(svc)

			a := &godaily.App{Subscribers: svc, Config: &env.Config{}}
			api.SetApp(a)

			url := "/api/unsubscribe"
			if test.token != "" {
				url += "?token=" + test.token
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, url, nil)

			HandleUnsubscribe(w, r)

			assert.Equal(t, test.wantStatus, w.Code)
			if test.wantLocation != "" {
				assert.Equal(t, test.wantLocation, w.Header().Get("Location"))
			}
		})
	}
}

func TestHandleUnsubscribePost(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocksubscriber.NewMockSubscriber(ctrl)
	svc.EXPECT().Unsubscribe(gomock.Any(), "valid-token").Return(nil)

	a := &godaily.App{Subscribers: svc, Config: &env.Config{}}
	api.SetApp(a)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/unsubscribe?token=valid-token", nil)

	HandleUnsubscribe(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Location"))
}
