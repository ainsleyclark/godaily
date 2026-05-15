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
	"context"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
)

// HandleUnsubscribe is the Vercel serverless function entry point for GET /api/unsubscribe.
func HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	api.Handle(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		token := r.URL.Query().Get("token")
		if token == "" {
			api.Error(w, http.StatusBadRequest, "missing token")
			return
		}

		if err := a.Subscribers.Unsubscribe(ctx, token); err != nil {
			api.Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		// RFC 8058: mail clients send a POST for one-click unsubscribe and
		// expect a 2xx response, not a redirect.
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Redirect(w, r, "/unsubscribed/", http.StatusFound)
	})(w, r)
}
