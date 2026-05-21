package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/env"
	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/domain/engagement"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleEmailStats(t *testing.T) {
	tt := map[string]struct {
		query      string
		mock       func(repo *mockengagement.MockEmailEventRepository)
		wantStatus int
	}{
		"Unauthorized":     {mock: func(_ *mockengagement.MockEmailEventRepository) {}, wantStatus: http.StatusUnauthorized},
		"Missing issue id": {mock: func(_ *mockengagement.MockEmailEventRepository) {}, wantStatus: http.StatusBadRequest},
		"Issue stats error": {
			query: "?issue_id=7",
			mock: func(repo *mockengagement.MockEmailEventRepository) {
				repo.EXPECT().IssueStats(gomock.Any(), int64(7)).Return(engagement.IssueStats{}, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"Top links error": {
			query: "?issue_id=7",
			mock: func(repo *mockengagement.MockEmailEventRepository) {
				repo.EXPECT().IssueStats(gomock.Any(), int64(7)).Return(engagement.IssueStats{IssueID: 7}, nil)
				repo.EXPECT().TopLinks(gomock.Any(), int64(7), int64(10)).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"OK with default link limit": {
			query: "?issue_id=7",
			mock: func(repo *mockengagement.MockEmailEventRepository) {
				repo.EXPECT().IssueStats(gomock.Any(), int64(7)).Return(engagement.IssueStats{IssueID: 7}, nil)
				repo.EXPECT().TopLinks(gomock.Any(), int64(7), int64(10)).Return([]engagement.LinkClicks{{URL: "https://x", Clicks: 1}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"OK with explicit link limit": {
			query: "?issue_id=7&link_limit=5",
			mock: func(repo *mockengagement.MockEmailEventRepository) {
				repo.EXPECT().IssueStats(gomock.Any(), int64(7)).Return(engagement.IssueStats{IssueID: 7}, nil)
				repo.EXPECT().TopLinks(gomock.Any(), int64(7), int64(5)).Return([]engagement.LinkClicks{}, nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mockengagement.NewMockEmailEventRepository(ctrl)
			test.mock(repo)

			a := &godaily.App{Config: &env.Config{APISecret: "topsecret"}, Repository: &godaily.Repository{EmailEvents: repo}}
			api.SetApp(a)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/email/stats"+test.query, nil)
			if name != "Unauthorized" {
				r.Header.Set("Authorization", "Bearer topsecret")
			}
			HandleEmailStats(w, r)
			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
