package handlers

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	godaily "github.com/ainsleyclark/godaily/internal"
	mocknews "github.com/ainsleyclark/godaily/internal/mocks/news"
	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/store"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

func TestDigest(t *testing.T) {
	t.Parallel()

	log.SetOutput(io.Discard)

	tt := map[string]struct {
		mock       func(issues *mocknews.MockIssueRepository)
		wantStatus int
		wantHTML   string
	}{
		"Not Found": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().
					FindBySlug(gomock.Any(), "issue-1").
					Return(news.Issue{}, store.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		"Internal Error": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().
					FindBySlug(gomock.Any(), "issue-1").
					Return(news.Issue{}, errors.New("internal error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		"OK": {
			mock: func(issues *mocknews.MockIssueRepository) {
				issues.EXPECT().
					FindBySlug(gomock.Any(), "issue-1").
					Return(news.Issue{Slug: "issue-1", Subject: "Go Weekly #1"}, nil)
			},
			wantStatus: http.StatusOK,
			wantHTML:   "Go Weekly #1",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockIssues := mocknews.NewMockIssueRepository(ctrl)

			if test.mock != nil {
				test.mock(mockIssues)
			}

			app := &godaily.App{
				Repository: &godaily.Repository{
					Issues: mockIssues,
				},
			}

			kit := webkit.New()
			kit.Get("/digest/{slug}/", Digest(app))

			req := httptest.NewRequest(http.MethodGet, "/digest/issue-1/", nil)
			rec := httptest.NewRecorder()
			kit.ServeHTTP(rec, req)

			assert.Equal(t, test.wantStatus, rec.Code)
			if test.wantHTML != "" {
				assert.Contains(t, rec.Body.String(), test.wantHTML)
			}
		})
	}
}
