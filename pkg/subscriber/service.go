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

// Package subscriber owns the subscription lifecycle: creating subscribers,
// sending welcome emails, and processing unsubscribes.
package subscriber

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"log/slog"
	texttemplate "text/template"

	"github.com/ainsleyclark/godaily/pkg/email"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/pkg/templates"
)

//go:generate go run go.uber.org/mock/mockgen -package=mocksubscriber -destination=../mocks/subscriber/Subscriber.go . Subscriber

// Subscriber defines the subscription lifecycle methods used by HTTP handlers.
type Subscriber interface {
	Subscribe(ctx context.Context, email string) (news.Subscriber, error)
	Unsubscribe(ctx context.Context, token string) error
}

// ErrAlreadySubscribed is returned by Subscribe when the email address is
// already registered as an active subscriber.
var ErrAlreadySubscribed = errors.New("already subscribed")

var (
	htmlTmpl = htmltemplate.Must(htmltemplate.New("welcome-html").Parse(templates.EmailLayout + templates.WelcomeHTML))
	textTmpl = texttemplate.Must(texttemplate.New("welcome-text").Parse(templates.EmailLayoutText + templates.WelcomeText))
)

type welcomeData struct {
	LatestIssueURL   string
	LatestIssueTitle string
	UnsubscribeURL   string
	CanonicalURL     string
}

// Service owns the full subscriber lifecycle.
type Service struct {
	repo   news.SubscriberRepository
	issues news.IssueRepository
	email  email.Sender
}

// New returns a Service wired to the provided dependencies.
func New(repo news.SubscriberRepository, issues news.IssueRepository, sender email.Sender) *Service {
	return &Service{
		repo:   repo,
		issues: issues,
		email:  sender,
	}
}

// Subscribe creates a new subscriber and sends a welcome email.
// It returns ErrAlreadySubscribed if the email is already registered as active.
// Previously unsubscribed addresses are reactivated with a fresh token.
// Welcome email failures are logged but do not fail the subscription.
func (s Service) Subscribe(ctx context.Context, emailAddr string) (news.Subscriber, error) {
	var sub news.Subscriber

	existing, err := s.repo.FindByEmail(ctx, emailAddr)
	switch {
	case err == nil && existing.UnsubscribedAt == nil:
		return news.Subscriber{}, ErrAlreadySubscribed
	case err == nil:
		sub, err = s.repo.Reactivate(ctx, emailAddr)
		if err != nil {
			return news.Subscriber{}, err
		}
	case errors.Is(err, store.ErrNotFound):
		sub, err = s.repo.Create(ctx, emailAddr)
		if err != nil {
			return news.Subscriber{}, err
		}
	default:
		return news.Subscriber{}, err
	}

	if sub.UnsubscribeToken == "" {
		return news.Subscriber{}, errors.New("subscriber created without unsubscribe token")
	}
	unsubURL := env.AppURL + "/api/unsubscribe?token=" + sub.UnsubscribeToken

	var latestIssueURL, latestIssueTitle string
	if latest, err := s.issues.Latest(ctx, 1); err == nil && len(latest) > 0 {
		latestIssueURL = env.AppURL + "/digest/" + latest[0].Slug + "/"
		latestIssueTitle = latest[0].Subject
	}

	if err = s.sendWelcome(ctx, sub.Email, unsubURL, latestIssueURL, latestIssueTitle); err != nil {
		slog.ErrorContext(ctx, "Failed to send welcome email", "email", sub.Email, "error", err)
	}

	return sub, nil
}

// Unsubscribe marks a subscriber as unsubscribed using their token.
func (s Service) Unsubscribe(ctx context.Context, token string) error {
	return s.repo.Unsubscribe(ctx, token)
}

func (s Service) sendWelcome(ctx context.Context, to, unsubURL, latestIssueURL, latestIssueTitle string) error {
	data := welcomeData{
		LatestIssueURL:   latestIssueURL,
		LatestIssueTitle: latestIssueTitle,
		UnsubscribeURL:   unsubURL,
	}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.ExecuteTemplate(&htmlBuf, "email-layout", data); err != nil {
		return fmt.Errorf("rendering welcome html: %w", err)
	}

	var textBuf bytes.Buffer
	if err := textTmpl.ExecuteTemplate(&textBuf, "email-layout-text", data); err != nil {
		return fmt.Errorf("rendering welcome text: %w", err)
	}

	return s.email.Send(ctx, email.SendEmailRequest{
		From:    "GoDaily <noreply@godaily.dev>",
		To:      []string{to},
		Subject: "Welcome to GoDaily!",
		Html:    htmlBuf.String(),
		Text:    textBuf.String(),
		Headers: map[string]string{
			"List-Unsubscribe":      "<" + unsubURL + ">",
			"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
		},
	})
}
