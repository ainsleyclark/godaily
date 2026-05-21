package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/domain/news"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleItems(t *testing.T) {
	tt := map[string]struct {
		query      string
		mock       func(items *mocknews.MockItemRepository)
		wantStatus int
	}{
		"Unauthorized": {
			query:      "",
			mock:       func(_ *mocknews.MockItemRepository) {},
			wantStatus: http.StatusUnauthorized,
		},
		"OK": {
			query: "",
			mock: func(items *mocknews.MockItemRepository) {
				items.EXPECT().List(gomock.Any(), news.ItemListOptions{}).Return([]news.Item{{ID: 1}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"Issue filter": {
			query: "?issue_id=42",
			mock: func(items *mocknews.MockItemRepository) {
				issueID := int64(42)
				items.EXPECT().List(gomock.Any(), news.ItemListOptions{IssueID: &issueID}).Return([]news.Item{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"Date range filters": {
			query: "?from=2026-01-01&to=2026-01-31",
			mock: func(items *mocknews.MockItemRepository) {
				from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
				items.EXPECT().List(gomock.Any(), news.ItemListOptions{From: &from, To: &to}).Return([]news.Item{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		"Bad from": {
			query:      "?from=01-01-2026",
			mock:       func(_ *mocknews.MockItemRepository) {},
			wantStatus: http.StatusBadRequest,
		},
		"Store error": {
			query: "",
			mock: func(items *mocknews.MockItemRepository) {
				items.EXPECT().List(gomock.Any(), news.ItemListOptions{}).Return(nil, errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			items := mocknews.NewMockItemRepository(ctrl)
			test.mock(items)

			a := &godaily.App{
				Config: &env.Config{APISecret: "topsecret"},
				Repository: &godaily.Repository{
					Items: items,
				},
			}
			api.SetApp(a)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/items"+test.query, nil)
			r.RemoteAddr = "1.2.3.4:1234"
			if name != "Unauthorized" {
				r.Header.Set("Authorization", "Bearer topsecret")
			}

			HandleItems(w, r)
			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
