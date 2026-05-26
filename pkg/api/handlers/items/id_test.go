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

package items

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestByID(t *testing.T) {
	tt := map[string]struct {
		mock       func(items *mocknews.MockItemRepository)
		id         string
		wantStatus int
	}{
		"OK": {
			mock: func(items *mocknews.MockItemRepository) {
				items.EXPECT().Find(gomock.Any(), int64(42)).Return(news.Item{ID: 42}, nil)
			},
			id:         "42",
			wantStatus: http.StatusOK,
		},
		"Not found": {
			mock: func(items *mocknews.MockItemRepository) {
				items.EXPECT().Find(gomock.Any(), int64(99)).Return(news.Item{}, store.ErrNotFound)
			},
			id:         "99",
			wantStatus: http.StatusNotFound,
		},
		"Missing id": {
			mock:       func(items *mocknews.MockItemRepository) {},
			id:         "",
			wantStatus: http.StatusBadRequest,
		},
		"Non-numeric id": {
			mock:       func(items *mocknews.MockItemRepository) {},
			id:         "abc",
			wantStatus: http.StatusBadRequest,
		},
		"Zero id": {
			mock:       func(items *mocknews.MockItemRepository) {},
			id:         "0",
			wantStatus: http.StatusBadRequest,
		},
		"Store error": {
			mock: func(items *mocknews.MockItemRepository) {
				items.EXPECT().Find(gomock.Any(), int64(1)).Return(news.Item{}, errors.New("db error"))
			},
			id:         "1",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			itemsMock := mocknews.NewMockItemRepository(ctrl)
			test.mock(itemsMock)

			h := &Handler{itemsRepo: itemsMock}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/items/"+test.id, nil)

			switch test.id {
			case "":
				// No chi context — Param returns ""
			default:
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("id", test.id)
				r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
			}

			invoke(h.ByID, w, r)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}

func invoke(h func(*webkit.Context) error, w *httptest.ResponseRecorder, r *http.Request) {
	c := webkit.NewContext(w, r)
	if err := h(c); err != nil {
		var e *webkit.Error
		if errors.As(err, &e) {
			_ = c.JSON(e.Code, map[string]string{"error": e.Message})
		} else {
			_ = c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}
}
