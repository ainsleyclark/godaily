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
	"strings"
	"testing"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	"github.com/ainsleyclark/godaily/pkg/env"
	mockaudience "github.com/ainsleyclark/godaily/pkg/mocks/audience"
	mockslack "github.com/ainsleyclark/godaily/pkg/mocks/slack"
	audiencesvc "github.com/ainsleyclark/godaily/pkg/services/audience"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleSubscribe(t *testing.T) {
	tt := map[string]struct {
		body       string
		method     string
		mock       func(s *mockaudience.MockService)
		repoMock   func(r *mockaudience.MockSubscriberRepository)
		wantStatus int
	}{
		"OK": {
			body:   `{"email":"test@example.com"}`,
			method: http.MethodPost,
			mock: func(s *mockaudience.MockService) {
				s.EXPECT().Subscribe(gomock.Any(), "test@example.com").Return(audience.Subscriber{}, nil)
			},
			repoMock: func(r *mockaudience.MockSubscriberRepository) {
				r.EXPECT().CountActive(gomock.Any()).Return(int64(42), nil)
			},
			wantStatus: http.StatusOK,
		},
		"Wrong Method": {
			body:       `{"email":"test@example.com"}`,
			method:     http.MethodGet,
			mock:       func(s *mockaudience.MockService) {},
			repoMock:   func(r *mockaudience.MockSubscriberRepository) {},
			wantStatus: http.StatusMethodNotAllowed,
		},
		"Missing Email": {
			body:       `{}`,
			method:     http.MethodPost,
			mock:       func(s *mockaudience.MockService) {},
			repoMock:   func(r *mockaudience.MockSubscriberRepository) {},
			wantStatus: http.StatusBadRequest,
		},
		"Invalid Email": {
			body:       `{"email":"notanemail"}`,
			method:     http.MethodPost,
			mock:       func(s *mockaudience.MockService) {},
			repoMock:   func(r *mockaudience.MockSubscriberRepository) {},
			wantStatus: http.StatusBadRequest,
		},
		"Already Subscribed": {
			body:   `{"email":"dupe@example.com"}`,
			method: http.MethodPost,
			mock: func(s *mockaudience.MockService) {
				s.EXPECT().Subscribe(gomock.Any(), "dupe@example.com").Return(audience.Subscriber{}, audiencesvc.ErrAlreadySubscribed)
			},
			repoMock:   func(r *mockaudience.MockSubscriberRepository) {},
			wantStatus: http.StatusConflict,
		},
		"Subscribe Error": {
			body:   `{"email":"err@example.com"}`,
			method: http.MethodPost,
			mock: func(s *mockaudience.MockService) {
				s.EXPECT().Subscribe(gomock.Any(), "err@example.com").Return(audience.Subscriber{}, errors.New("db error"))
			},
			repoMock:   func(r *mockaudience.MockSubscriberRepository) {},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc := mockaudience.NewMockService(ctrl)
			slack := mockslack.NewMockSender(ctrl)
			repo := mockaudience.NewMockSubscriberRepository(ctrl)
			slack.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()
			test.mock(svc)
			test.repoMock(repo)

			a := &godaily.App{
				Subscribers: svc,
				Config:      &env.Config{},
				Slack:       slack,
				Repository:  &godaily.Repository{Subscribers: repo},
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(test.method, "/digest/subscribe", strings.NewReader(test.body))
			r = r.WithContext(api.WithApp(r.Context(), a))

			HandleSubscribe(w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
