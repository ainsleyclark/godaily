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
	"strings"
	"testing"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/env"
	mocksubscriber "github.com/ainsleyclark/godaily/pkg/mocks/subscriber"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/subscriber"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleSubscribe(t *testing.T) {
	tt := map[string]struct {
		body       string
		method     string
		mock       func(s *mocksubscriber.MockSubscriber)
		wantStatus int
	}{
		"OK": {
			body:   `{"email":"test@example.com"}`,
			method: http.MethodPost,
			mock: func(s *mocksubscriber.MockSubscriber) {
				s.EXPECT().Subscribe(gomock.Any(), "test@example.com").Return(news.Subscriber{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"Wrong Method": {
			body:       `{"email":"test@example.com"}`,
			method:     http.MethodGet,
			mock:       func(s *mocksubscriber.MockSubscriber) {},
			wantStatus: http.StatusMethodNotAllowed,
		},
		"Missing Email": {
			body:       `{}`,
			method:     http.MethodPost,
			mock:       func(s *mocksubscriber.MockSubscriber) {},
			wantStatus: http.StatusBadRequest,
		},
		"Invalid Email": {
			body:       `{"email":"notanemail"}`,
			method:     http.MethodPost,
			mock:       func(s *mocksubscriber.MockSubscriber) {},
			wantStatus: http.StatusBadRequest,
		},
		"Already Subscribed": {
			body:   `{"email":"dupe@example.com"}`,
			method: http.MethodPost,
			mock: func(s *mocksubscriber.MockSubscriber) {
				s.EXPECT().Subscribe(gomock.Any(), "dupe@example.com").Return(news.Subscriber{}, subscriber.ErrAlreadySubscribed)
			},
			wantStatus: http.StatusConflict,
		},
		"Subscribe Error": {
			body:   `{"email":"err@example.com"}`,
			method: http.MethodPost,
			mock: func(s *mocksubscriber.MockSubscriber) {
				s.EXPECT().Subscribe(gomock.Any(), "err@example.com").Return(news.Subscriber{}, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc := mocksubscriber.NewMockSubscriber(ctrl)
			test.mock(svc)

			api.App = &godaily.App{Subscribers: svc, Config: &env.Config{}}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(test.method, "/api/subscribe", strings.NewReader(test.body))
			HandleSubscribe(w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
