// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	"github.com/ainsleyclark/godaily/pkg/mocks/audience"
	"github.com/ainsleyclark/godaily/pkg/mocks/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSubscribe(t *testing.T) {
	tt := map[string]struct {
		body       string
		mock       func(s *mockaudience.MockSubscriberService)
		repoMock   func(r *mockaudience.MockSubscriberRepository)
		wantStatus int
	}{
		"OK": {
			body: `{"email":"test@example.com"}`,
			mock: func(s *mockaudience.MockSubscriberService) {
				s.EXPECT().Subscribe(gomock.Any(), "test@example.com").Return(audience.Subscriber{}, nil)
			},
			repoMock: func(r *mockaudience.MockSubscriberRepository) {
				r.EXPECT().CountActive(gomock.Any()).Return(int64(42), nil)
			},
			wantStatus: http.StatusOK,
		},
		"Missing Email": {
			body:       `{}`,
			mock:       func(s *mockaudience.MockSubscriberService) {},
			repoMock:   func(r *mockaudience.MockSubscriberRepository) {},
			wantStatus: http.StatusBadRequest,
		},
		"Invalid Email": {
			body:       `{"email":"notanemail"}`,
			mock:       func(s *mockaudience.MockSubscriberService) {},
			repoMock:   func(r *mockaudience.MockSubscriberRepository) {},
			wantStatus: http.StatusBadRequest,
		},
		"Already Subscribed": {
			body: `{"email":"dupe@example.com"}`,
			mock: func(s *mockaudience.MockSubscriberService) {
				s.EXPECT().Subscribe(gomock.Any(), "dupe@example.com").Return(audience.Subscriber{}, audience.ErrAlreadySubscribed)
			},
			repoMock:   func(r *mockaudience.MockSubscriberRepository) {},
			wantStatus: http.StatusConflict,
		},
		"Subscribe Error": {
			body: `{"email":"err@example.com"}`,
			mock: func(s *mockaudience.MockSubscriberService) {
				s.EXPECT().Subscribe(gomock.Any(), "err@example.com").Return(audience.Subscriber{}, errors.New("db error"))
			},
			repoMock:   func(r *mockaudience.MockSubscriberRepository) {},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc := mockaudience.NewMockSubscriberService(ctrl)
			slackMock := mockslack.NewMockSender(ctrl)
			repo := mockaudience.NewMockSubscriberRepository(ctrl)
			slackMock.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()
			test.mock(svc)
			test.repoMock(repo)

			h := &Handler{
				subscribers:     svc,
				subscribersRepo: repo,
				slack:           slackMock,
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/subscribe", strings.NewReader(test.body))
			invoke(h.Subscribe, w, r)
			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
