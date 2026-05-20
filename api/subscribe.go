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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/services/subscriber"
)

// HandleSubscribe is the Vercel serverless function entry point for POST /api/subscribe.
func HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	api.Handle(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
			api.Error(w, http.StatusBadRequest, "email is required")
			return
		}

		if _, err := mail.ParseAddress(body.Email); err != nil {
			api.Error(w, http.StatusBadRequest, "invalid email address")
			return
		}

		if _, err := a.Subscribers.Subscribe(ctx, body.Email); err != nil {
			if errors.Is(err, subscriber.ErrAlreadySubscribed) {
				api.Error(w, http.StatusConflict, "already subscribed")
				return
			}
			api.Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		msg := "New subscriber: " + body.Email
		if count, err := a.Repository.Subscribers.CountActive(ctx); err == nil {
			msg += fmt.Sprintf(" | Total subscribers: %d", count)
		}
		a.Slack.MustSend(ctx, msg)
		api.OK(w)
	})(w, r)
}
