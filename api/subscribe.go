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

// Package api contains Vercel serverless function handlers.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/bootstrap"
	"github.com/ainsleyclark/godaily/pkg/email"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// HandleSubscribe is the Vercel serverless function entry point for POST /api/subscribe.
func HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		http.Error(w, `{"error":"email is required"}`, http.StatusBadRequest)
		return
	}

	bootstrap.Handle(w, r, func(app *godaily.App) {
		ctx := r.Context()

		_, err := app.Repository.Subscribers.FindByEmail(ctx, body.Email)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "already subscribed"})
			return
		}
		if !errors.Is(err, store.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sub, err := app.Repository.Subscribers.Create(ctx, body.Email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		confirmURL := godaily.AppURL + "/api/confirm?token=" + sub.ConfirmToken
		if sendErr := email.New().Send(ctx, email.SendEmailRequest{
			From:    "noreply@godaily.dev",
			To:      []string{sub.Email},
			Subject: "Confirm your GoDaily subscription",
			Html:    confirmEmailHTML(confirmURL),
			Text:    fmt.Sprintf("Confirm your GoDaily subscription:\n\n%s\n\nIf you didn't sign up, you can ignore this email.", confirmURL),
		}); sendErr != nil {
			slog.ErrorContext(ctx, "Failed to send confirmation email", "email", sub.Email, "err", sendErr)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
}

func confirmEmailHTML(confirmURL string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Confirm your GoDaily subscription</title></head>
<body style="margin:0;padding:0;background:#f5fbff;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f5fbff;">
    <tr><td align="center" style="padding:40px 16px;">
      <table width="560" cellpadding="0" cellspacing="0" style="max-width:560px;background:#ffffff;border:1px solid #d0e8f5;padding:40px 32px;">
        <tr><td>
          <div style="font-size:15px;font-weight:700;color:#0d2236;margin-bottom:24px;">🐿️ GoDaily</div>
          <h1 style="font-size:20px;color:#0d2236;margin:0 0 12px;">Confirm your subscription</h1>
          <p style="font-size:14px;color:#3a6880;margin:0 0 24px;">Click the button below to confirm your email and start receiving the daily Go digest, weekday mornings.</p>
          <a href="` + confirmURL + `" style="display:inline-block;background:#1a7fa8;color:#ffffff;font-size:14px;font-weight:600;text-decoration:none;padding:12px 28px;border-radius:4px;">Confirm subscription</a>
          <p style="font-size:12px;color:#6b9ab8;margin:24px 0 0;">If you didn't sign up for GoDaily, you can safely ignore this email.</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`
}
