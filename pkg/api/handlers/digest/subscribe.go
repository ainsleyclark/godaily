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
	"errors"
	"fmt"
	"net/http"
	"net/mail"

	"github.com/ainsleyclark/godaily/pkg/services/audience"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Subscribe handles POST /subscribe.
func (h *Handler) Subscribe(c *webkit.Context) error {
	ctx := c.Context()

	var body struct {
		Email string `json:"email"`
	}
	if err := c.BindJSON(&body); err != nil || body.Email == "" {
		return webkit.NewError(http.StatusBadRequest, "email is required")
	}

	if _, err := mail.ParseAddress(body.Email); err != nil {
		return webkit.NewError(http.StatusBadRequest, "invalid email address")
	}

	if _, err := h.subscribers.Subscribe(ctx, body.Email); err != nil {
		if errors.Is(err, audience.ErrAlreadySubscribed) {
			return webkit.NewError(http.StatusConflict, "already subscribed")
		}
		return webkit.NewError(http.StatusInternalServerError, err.Error())
	}

	msg := "New subscriber: " + body.Email
	if count, err := h.subscribersRepo.CountActive(ctx); err == nil {
		msg += fmt.Sprintf(" | Total subscribers: %d", count)
	}
	h.slack.MustSend(ctx, msg)

	return c.NoContent(http.StatusOK)
}
