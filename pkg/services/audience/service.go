// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audience

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"log/slog"
	texttemplate "text/template"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/pkg/templates"
)

// Service owns the full subscriber lifecycle.
type Service struct {
	repo   audience.SubscriberRepository
	issues digest.IssueRepository
	email  email.Sender

	// now returns the current time. It is a field so tests can pin the clock
	// when asserting the confirmation-nudge eligibility window.
	now func() time.Time
}

var _ audience.SubscriberService = (*Service)(nil)

// New returns a Service wired to the provided dependencies.
func New(repo audience.SubscriberRepository, issues digest.IssueRepository, sender email.Sender) *Service {
	return &Service{
		repo:   repo,
		issues: issues,
		email:  sender,
		now:    time.Now,
	}
}

const (
	// nudgeMinAge is how long an unconfirmed sign-up is left alone before the
	// reminder, giving people time to confirm on their own first.
	nudgeMinAge = 24 * time.Hour
	// nudgeMaxAge bounds how old a sign-up can be to still receive a reminder.
	// It stops the first run from emailing long-dead historical sign-ups.
	nudgeMaxAge = 14 * 24 * time.Hour
)

var (
	htmlTmpl = htmltemplate.Must(htmltemplate.New("confirm-html").Parse(templates.EmailLayout + templates.ConfirmHTML))
	textTmpl = texttemplate.Must(texttemplate.New("confirm-text").Parse(templates.EmailLayoutText + templates.ConfirmText))

	nudgeHTMLTmpl = htmltemplate.Must(htmltemplate.New("nudge-html").Parse(templates.EmailLayout + templates.NudgeHTML))
	nudgeTextTmpl = texttemplate.Must(texttemplate.New("nudge-text").Parse(templates.EmailLayoutText + templates.NudgeText))
)

type confirmData struct {
	ConfirmURL     string
	UnsubscribeURL string
	CanonicalURL   string
}

// Subscribe creates a new subscriber and sends a confirmation email.
// It returns ErrAlreadySubscribed if the email is already registered as active.
// Previously unsubscribed addresses are reactivated with a fresh token.
// Confirmation email failures are logged but do not fail the subscription.
func (s Service) Subscribe(ctx context.Context, emailAddr string) (audience.Subscriber, error) {
	var sub audience.Subscriber

	existing, err := s.repo.FindByEmail(ctx, emailAddr)
	switch {
	case err == nil && existing.UnsubscribedAt == nil:
		return audience.Subscriber{}, audience.ErrAlreadySubscribed
	case err == nil:
		sub, err = s.repo.Reactivate(ctx, emailAddr)
		if err != nil {
			return audience.Subscriber{}, err
		}
	case errors.Is(err, store.ErrNotFound):
		sub, err = s.repo.Create(ctx, emailAddr)
		if err != nil {
			return audience.Subscriber{}, err
		}
	default:
		return audience.Subscriber{}, err
	}

	if sub.ConfirmToken == "" {
		return audience.Subscriber{}, errors.New("subscriber created without confirmation token")
	}
	confirmURL := env.AppURL + "/api/confirm?token=" + sub.ConfirmToken
	unsubscribeURL := env.AppURL + "/api/unsubscribe/?token=" + sub.UnsubscribeToken

	if err = s.sendConfirmation(ctx, sub.Email, confirmURL, unsubscribeURL); err != nil {
		slog.ErrorContext(ctx, "Failed to send confirmation email", "email", sub.Email, "error", err)
	}

	return sub, nil
}

// Confirm verifies a subscriber's email address using their confirmation token.
// Returns store.ErrNotFound if the token is invalid or already used.
func (s Service) Confirm(ctx context.Context, token string) error {
	_, err := s.repo.Confirm(ctx, token)
	return err
}

// Unsubscribe marks a subscriber as unsubscribed using their token.
func (s Service) Unsubscribe(ctx context.Context, token string) error {
	return s.repo.Unsubscribe(ctx, token)
}

// SendConfirmationNudges sends a single, one-time reminder to people who signed
// up but never confirmed. A subscriber is eligible when they are unconfirmed,
// still hold a confirm token, are otherwise healthy (not unsubscribed, bounced
// or suppressed), have never been nudged, and signed up within the
// [nudgeMinAge, nudgeMaxAge] window. It returns how many reminders were sent
// and how many failed; individual failures are logged and do not abort the run.
func (s Service) SendConfirmationNudges(ctx context.Context) (sent, failed int, err error) {
	subs, err := s.repo.List(ctx, store.ListOptions{})
	if err != nil {
		return 0, 0, fmt.Errorf("listing subscribers for confirmation nudge: %w", err)
	}

	now := s.now().UTC()
	for _, sub := range subs {
		if !nudgeEligible(sub, now) {
			continue
		}

		confirmURL := env.AppURL + "/api/confirm?token=" + sub.ConfirmToken
		unsubscribeURL := env.AppURL + "/api/unsubscribe/?token=" + sub.UnsubscribeToken

		if sendErr := s.sendNudge(ctx, sub.Email, confirmURL, unsubscribeURL); sendErr != nil {
			slog.ErrorContext(ctx, "Failed to send confirmation nudge", "email", sub.Email, "error", sendErr)
			failed++
			continue
		}
		sent++

		// The email has gone out; record it so we never nudge twice. A failure
		// here only risks a duplicate on the next run, so it is logged, not fatal.
		if markErr := s.repo.MarkNudgeSent(ctx, sub.ID); markErr != nil {
			slog.ErrorContext(ctx, "Failed to mark confirmation nudge sent", "id", sub.ID, "error", markErr)
		}
	}

	return sent, failed, nil
}

// nudgeEligible reports whether an unconfirmed subscriber should receive the
// one-time confirmation reminder, relative to now.
func nudgeEligible(sub audience.Subscriber, now time.Time) bool {
	if sub.ConfirmedAt != nil || sub.UnsubscribedAt != nil || sub.BouncedAt != nil || sub.SuppressedAt != nil {
		return false
	}
	if sub.ConfirmToken == "" || sub.ConfirmationNudgeSentAt != nil {
		return false
	}
	age := now.Sub(sub.CreatedAt.UTC())
	return age >= nudgeMinAge && age <= nudgeMaxAge
}

// MarkBounced flags a subscriber whose address hard-bounced so the digest is
// no longer sent to it. It is keyed by email because bounce notifications
// identify the recipient by address, not token.
func (s Service) MarkBounced(ctx context.Context, emailAddr string) error {
	return s.repo.MarkBounced(ctx, emailAddr)
}

// MarkComplained unsubscribes a subscriber who reported the digest as spam.
func (s Service) MarkComplained(ctx context.Context, emailAddr string) error {
	return s.repo.MarkComplained(ctx, emailAddr)
}

// MarkSuppressed flags a subscriber whose address is on Resend's global
// suppression list, meaning delivery will be silently refused.
func (s Service) MarkSuppressed(ctx context.Context, emailAddr string) error {
	return s.repo.MarkSuppressed(ctx, emailAddr)
}

func (s Service) sendNudge(ctx context.Context, to, confirmURL, unsubscribeURL string) error {
	data := confirmData{
		ConfirmURL:     confirmURL,
		UnsubscribeURL: unsubscribeURL,
	}

	var htmlBuf bytes.Buffer
	if err := nudgeHTMLTmpl.ExecuteTemplate(&htmlBuf, "email-layout", data); err != nil {
		return fmt.Errorf("rendering nudge html: %w", err)
	}

	var textBuf bytes.Buffer
	if err := nudgeTextTmpl.ExecuteTemplate(&textBuf, "email-layout-text", data); err != nil {
		return fmt.Errorf("rendering nudge text: %w", err)
	}

	return s.email.Send(ctx, email.SendEmailRequest{
		From:    "GoDaily <digest@godaily.dev>",
		To:      []string{to},
		Subject: "Confirm your GoDaily subscription, one click left",
		Html:    htmlBuf.String(),
		Text:    textBuf.String(),
		Headers: map[string]string{
			"List-Unsubscribe":      "<" + unsubscribeURL + ">",
			"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
		},
	})
}

func (s Service) sendConfirmation(ctx context.Context, to, confirmURL, unsubscribeURL string) error {
	data := confirmData{
		ConfirmURL:     confirmURL,
		UnsubscribeURL: unsubscribeURL,
	}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.ExecuteTemplate(&htmlBuf, "email-layout", data); err != nil {
		return fmt.Errorf("rendering confirmation html: %w", err)
	}

	var textBuf bytes.Buffer
	if err := textTmpl.ExecuteTemplate(&textBuf, "email-layout-text", data); err != nil {
		return fmt.Errorf("rendering confirmation text: %w", err)
	}

	return s.email.Send(ctx, email.SendEmailRequest{
		From:    "GoDaily <digest@godaily.dev>",
		To:      []string{to},
		Subject: "Confirm your GoDaily subscription",
		Html:    htmlBuf.String(),
		Text:    textBuf.String(),
		Headers: map[string]string{
			"List-Unsubscribe":      "<" + unsubscribeURL + ">",
			"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
		},
	})
}
