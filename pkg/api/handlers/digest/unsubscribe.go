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

package digest

import (
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Unsubscribe handles /unsubscribe.
// GET serves the link click (redirect to /unsubscribed/),
// POST serves the RFC 8058 one-click unsubscribe (return 200 OK).
func (h *Handler) Unsubscribe(c *webkit.Context) error {
	ctx := c.Context()
	r := c.Request

	token := r.URL.Query().Get("token")
	if token == "" {
		return webkit.NewError(http.StatusBadRequest, "missing token")
	}

	if err := h.subscribers.Unsubscribe(ctx, token); err != nil {
		return webkit.NewError(http.StatusInternalServerError, err.Error())
	}

	// RFC 8058: mail clients send a POST for one-click unsubscribe and
	// expect a 2xx response, not a redirect.
	if r.Method == http.MethodPost {
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/unsubscribed/")
}
