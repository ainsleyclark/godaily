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

// Package bootstrap provides shared bootstrap wiring for api/ serverless handlers.
package bootstrap

import (
	"log/slog"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/digest"
)

// Handle bootstraps the app and calls fn with the runner. It writes a 500 and
// returns early if Bootstrap fails, so fn is only called on success.
func Handle(w http.ResponseWriter, r *http.Request, fn func(digest.Runner)) {
	ctx := r.Context()

	app, teardown, err := godaily.Bootstrap(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Bootstrapping app", "error", err)
		http.Error(w, "failed to bootstrap app: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer teardown()

	fn(app.Runner)
}
