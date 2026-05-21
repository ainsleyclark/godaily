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
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/domain/news"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleSubscribers(t *testing.T) {
	tt := map[string]struct {
		mock       func(subs *mocknews.MockSubscriberRepository)
		query      string
		wantStatus int
	}{
		"OK default pagination": {
			mock: func(subs *mocknews.MockSubscriberRepository) {
				subs.EXPECT().CountAll(gomock.Any()).Return(int64(3), nil)
				subs.EXPECT().List(gomock.Any(), news.ListOptions{Page: 1, PerPage: 20}).Return([]news.Subscriber{
					{ID: 1, Email: "a@example.com"},
					{ID: 2, Email: "b@example.com"},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"OK with explicit page params": {
			mock: func(subs *mocknews.MockSubscriberRepository) {
				subs.EXPECT().CountAll(gomock.Any()).Return(int64(50), nil)
				subs.EXPECT().List(gomock.Any(), news.ListOptions{Page: 2, PerPage: 5}).Return([]news.Subscriber{}, nil)
			},
			query:      "?page=2&per_page=5",
			wantStatus: http.StatusOK,
		},
		"CountAll error": {
			mock: func(subs *mocknews.MockSubscriberRepository) {
				subs.EXPECT().CountAll(gomock.Any()).Return(int64(0), errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"List error": {
			mock: func(subs *mocknews.MockSubscriberRepository) {
				subs.EXPECT().CountAll(gomock.Any()).Return(int64(1), nil)
				subs.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"Invalid page falls back to default": {
			mock: func(subs *mocknews.MockSubscriberRepository) {
				subs.EXPECT().CountAll(gomock.Any()).Return(int64(1), nil)
				subs.EXPECT().List(gomock.Any(), news.ListOptions{Page: 1, PerPage: 20}).Return([]news.Subscriber{}, nil)
			},
			query:      "?page=abc",
			wantStatus: http.StatusOK,
		},
		"per_page exceeds max falls back to default": {
			mock: func(subs *mocknews.MockSubscriberRepository) {
				subs.EXPECT().CountAll(gomock.Any()).Return(int64(1), nil)
				subs.EXPECT().List(gomock.Any(), news.ListOptions{Page: 1, PerPage: 20}).Return([]news.Subscriber{}, nil)
			},
			query:      "?per_page=999",
			wantStatus: http.StatusOK,
		},
		"Unauthorized": {
			mock:       func(_ *mocknews.MockSubscriberRepository) {},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			subsMock := mocknews.NewMockSubscriberRepository(ctrl)
			test.mock(subsMock)

			a := &godaily.App{
				Config: &env.Config{APISecret: "test-secret"},
				Repository: &godaily.Repository{
					Subscribers: subsMock,
				},
			}
			api.SetApp(a)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/subscribers"+test.query, nil)
			r.RemoteAddr = "1.2.3.4:1234"

			if name != "Unauthorized" {
				r.Header.Set("Authorization", "Bearer test-secret")
			}

			HandleSubscribers(w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
