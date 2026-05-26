// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	"github.com/ainsleyclark/godaily/pkg/mocks/audience"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSubscribers(t *testing.T) {
	tt := map[string]struct {
		mock       func(subs *mockaudience.MockSubscriberRepository)
		query      string
		wantStatus int
	}{
		"OK default pagination": {
			mock: func(subs *mockaudience.MockSubscriberRepository) {
				subs.EXPECT().CountAll(gomock.Any()).Return(int64(3), nil)
				subs.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]audience.Subscriber{
					{ID: 1, Email: "a@example.com"},
					{ID: 2, Email: "b@example.com"},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"OK with explicit page params": {
			mock: func(subs *mockaudience.MockSubscriberRepository) {
				subs.EXPECT().CountAll(gomock.Any()).Return(int64(50), nil)
				subs.EXPECT().List(gomock.Any(), store.ListOptions{Page: 2, PerPage: 5}).Return([]audience.Subscriber{}, nil)
			},
			query:      "?page=2&per_page=5",
			wantStatus: http.StatusOK,
		},
		"CountAll error": {
			mock: func(subs *mockaudience.MockSubscriberRepository) {
				subs.EXPECT().CountAll(gomock.Any()).Return(int64(0), errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"List error": {
			mock: func(subs *mockaudience.MockSubscriberRepository) {
				subs.EXPECT().CountAll(gomock.Any()).Return(int64(1), nil)
				subs.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"Invalid page falls back to default": {
			mock: func(subs *mockaudience.MockSubscriberRepository) {
				subs.EXPECT().CountAll(gomock.Any()).Return(int64(1), nil)
				subs.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]audience.Subscriber{}, nil)
			},
			query:      "?page=abc",
			wantStatus: http.StatusOK,
		},
		"per_page exceeds max falls back to default": {
			mock: func(subs *mockaudience.MockSubscriberRepository) {
				subs.EXPECT().CountAll(gomock.Any()).Return(int64(1), nil)
				subs.EXPECT().List(gomock.Any(), store.ListOptions{Page: 1, PerPage: 20}).Return([]audience.Subscriber{}, nil)
			},
			query:      "?per_page=999",
			wantStatus: http.StatusOK,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			subsMock := mockaudience.NewMockSubscriberRepository(ctrl)
			test.mock(subsMock)

			h := &Handler{subscribersRepo: subsMock}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/digest/subscribers"+test.query, nil)
			invoke(h.Subscribers, w, r)
			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
