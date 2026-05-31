// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audience

import (
	"context"
	"time"

	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/pkg/errors"
)

// Subscriber defines a person who has signed up to receive Go Daily.
type Subscriber struct {
	ID               int64      `json:"id"`
	Email            string     `json:"email"`
	UnsubscribeToken string     `json:"unsubscribe_token"`
	ConfirmToken     string     `json:"confirm_token,omitempty"`
	ConfirmedAt      *time.Time `json:"confirmed_at,omitempty"`
	UnsubscribedAt   *time.Time `json:"unsubscribed_at,omitempty"`
	BouncedAt        *time.Time `json:"bounced_at,omitempty"`
	SuppressedAt     *time.Time `json:"suppressed_at,omitempty"`
	// ConfirmationNudgeSentAt records when the one-time reminder to confirm
	// was sent, so an unconfirmed subscriber is never nudged more than once.
	ConfirmationNudgeSentAt *time.Time `json:"confirmation_nudge_sent_at,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
}

// ErrAlreadySubscribed is returned by Subscribe when the email address is
// already registered as an active subscriber.
var ErrAlreadySubscribed = errors.New("already subscribed")

//go:generate go run go.uber.org/mock/mockgen -package=mockaudience -destination=../../mocks/audience/Service.go . SubscriberService

// SubscriberService defines the subscription lifecycle methods used by HTTP handlers
// and the email webhook pipeline.
type SubscriberService interface {
	Subscribe(ctx context.Context, email string) (Subscriber, error)
	Confirm(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
	// SendConfirmationNudges sends a one-time reminder to subscribers who
	// signed up but never confirmed, returning how many were sent and failed.
	SendConfirmationNudges(ctx context.Context) (sent, failed int, err error)
	MarkBounced(ctx context.Context, email string) error
	MarkComplained(ctx context.Context, email string) error
	MarkSuppressed(ctx context.Context, email string) error
}

//go:generate go run go.uber.org/mock/mockgen -package=mockaudience -destination=../../mocks/audience/SubscriberRepository.go . SubscriberRepository

// SubscriberRepository defines the methods for interacting with the
// subscriber store.
type SubscriberRepository interface {
	Find(ctx context.Context, id int64) (Subscriber, error)
	FindByEmail(ctx context.Context, email string) (Subscriber, error)
	FindByUnsubscribeToken(ctx context.Context, token string) (Subscriber, error)
	Create(ctx context.Context, email string) (Subscriber, error)
	Reactivate(ctx context.Context, email string) (Subscriber, error)
	Confirm(ctx context.Context, token string) (Subscriber, error)
	Unsubscribe(ctx context.Context, token string) error
	List(ctx context.Context, opts store.ListOptions) ([]Subscriber, error)
	ListActive(ctx context.Context) ([]Subscriber, error)
	CountAll(ctx context.Context) (int64, error)
	CountActive(ctx context.Context) (int64, error)
	MarkBounced(ctx context.Context, email string) error
	MarkComplained(ctx context.Context, email string) error
	MarkSuppressed(ctx context.Context, email string) error
	// MarkNudgeSent stamps confirmation_nudge_sent_at so the confirmation
	// reminder is only ever sent once per subscriber.
	MarkNudgeSent(ctx context.Context, id int64) error
	CountFiltered(ctx context.Context, search string) (int64, error)
	AdminSetStatus(ctx context.Context, id int64, status string) (Subscriber, error)
}
