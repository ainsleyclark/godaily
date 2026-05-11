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

package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// Handler is the Vercel serverless function entry point for GET /api/items/{id}.
// The id path segment is injected by Vercel as the "id" query parameter.
func Handler(w http.ResponseWriter, r *http.Request) {
	api.Handle(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		raw := r.URL.Query().Get("id")
		if raw == "" {
			api.Error(w, http.StatusBadRequest, "id is required")
			return
		}

		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id < 1 {
			api.Error(w, http.StatusBadRequest, "id must be a positive integer")
			return
		}

		item, err := a.Repository.Items.Find(ctx, id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				api.Error(w, http.StatusNotFound, "item not found")
				return
			}
			api.Error(w, http.StatusInternalServerError, "failed to fetch item")
			return
		}

		api.JSON(w, http.StatusOK, item)
	})(w, r)
}
