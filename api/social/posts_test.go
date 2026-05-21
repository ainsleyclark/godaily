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

func TestHandleSocialPosts(t *testing.T) {
	tt := map[string]struct {
		query      string
		mock       func(posts *mocknews.MockSocialPostRepository)
		wantStatus int
	}{
		"Unauthorized":     {mock: func(_ *mocknews.MockSocialPostRepository) {}, wantStatus: http.StatusUnauthorized},
		"Missing issue id": {query: "", mock: func(_ *mocknews.MockSocialPostRepository) {}, wantStatus: http.StatusBadRequest},
		"OK": {
			query: "?issue_id=8",
			mock: func(posts *mocknews.MockSocialPostRepository) {
				posts.EXPECT().ListForIssue(gomock.Any(), int64(8)).Return([]news.SocialPost{{ID: 1}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"Store error": {
			query: "?issue_id=8",
			mock: func(posts *mocknews.MockSocialPostRepository) {
				posts.EXPECT().ListForIssue(gomock.Any(), int64(8)).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			posts := mocknews.NewMockSocialPostRepository(ctrl)
			test.mock(posts)

			a := &godaily.App{Config: &env.Config{APISecret: "topsecret"}, Repository: &godaily.Repository{SocialPosts: posts}}
			api.SetApp(a)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/social/posts"+test.query, nil)
			if name != "Unauthorized" {
				r.Header.Set("Authorization", "Bearer topsecret")
			}
			HandleSocialPosts(w, r)
			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
