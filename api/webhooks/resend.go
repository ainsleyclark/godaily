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
	"io"
	"log/slog"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
)

// Handler is the Vercel serverless function entry point for
// POST /api/webhooks/resend. The endpoint is public but every request is
// verified against the Svix-style signature Resend includes; unsigned or
// tampered requests are rejected.
//
// Status codes are chosen for Resend's retry behaviour: 2xx acknowledges
// (stop retrying), 5xx asks Resend to retry, and 4xx reports a permanent
// rejection.
func Handler(w http.ResponseWriter, r *http.Request) {
	api.Handle(func(ctx context.Context, w http.ResponseWriter, r *http.Request, a *godaily.App) {
		if r.Method != http.MethodPost {
			api.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		secret := a.Config.ResendWebhookSecret
		if secret == "" {
			api.Error(w, http.StatusInternalServerError, "resend webhook secret is not configured")
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			api.Error(w, http.StatusBadRequest, "cannot read request body")
			return
		}

		headers := email.WebhookHeaders{
			ID:        r.Header.Get("svix-id"),
			Timestamp: r.Header.Get("svix-timestamp"),
			Signature: r.Header.Get("svix-signature"),
		}
		if err := email.VerifyWebhook(string(body), headers, secret); err != nil {
			slog.WarnContext(ctx, "Rejected Resend webhook with invalid signature", "err", err)
			api.Error(w, http.StatusUnauthorized, "invalid signature")
			return
		}

		evt, err := email.ParseWebhook(body)
		if err != nil {
			api.Error(w, http.StatusBadRequest, "invalid payload")
			return
		}

		domainEvt, tracked, err := email.ToEmailEvent(evt, headers.ID)
		if err != nil {
			api.Error(w, http.StatusBadRequest, "invalid payload")
			return
		}
		if !tracked {
			// An event type GoDaily does not record — acknowledge it so
			// Resend stops retrying.
			api.OK(w)
			return
		}

		if err := a.EmailEvents.Process(ctx, domainEvt); err != nil {
			slog.ErrorContext(ctx, "Failed to process Resend webhook event", "type", domainEvt.Type, "err", err)
			api.Error(w, http.StatusInternalServerError, "failed to process event")
			return
		}

		api.OK(w)
	})(w, r)
}
