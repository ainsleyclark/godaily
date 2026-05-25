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

// Package subscriber defines the types and repository interface for GoDaily
// newsletter subscribers and their lifecycle.
package subscriber

import (
	"context"
	"time"
)

const defaultPerPage int64 = 20

// ListOptions controls filtering and pagination for List queries.
// A zero value returns all results (no pagination).
type ListOptions struct {
	// Page is 1-based. Zero means no pagination.
	Page int64

	// PerPage is the number of items per page. Zero uses the default (20).
	PerPage int64
}

// Limit returns the SQL LIMIT value for this page.
// Returns a large sentinel (10000) when pagination is disabled.
func (o ListOptions) Limit() int64 {
	if o.Page == 0 {
		return 10000
	}
	if o.PerPage <= 0 {
		return defaultPerPage
	}
	return o.PerPage
}

// Offset returns the SQL OFFSET value for the current page.
func (o ListOptions) Offset() int64 {
	if o.Page <= 1 {
		return 0
	}
	return (o.Page - 1) * o.Limit()
}

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
	CreatedAt        time.Time  `json:"created_at"`
}

//go:generate go run go.uber.org/mock/mockgen -package=mocksubscriber -destination=../../mocks/subscriber/SubscriberRepository.go . SubscriberRepository

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
	List(ctx context.Context, opts ListOptions) ([]Subscriber, error)
	ListActive(ctx context.Context) ([]Subscriber, error)
	CountAll(ctx context.Context) (int64, error)
	CountActive(ctx context.Context) (int64, error)
	MarkBounced(ctx context.Context, email string) error
	MarkComplained(ctx context.Context, email string) error
	MarkSuppressed(ctx context.Context, email string) error
}

//go:generate go run go.uber.org/mock/mockgen -package=mocksubscriber -destination=../../mocks/subscriber/Service.go . Service

// Service defines the subscription lifecycle methods used by HTTP handlers
// and the email webhook pipeline.
type Service interface {
	Subscribe(ctx context.Context, email string) (Subscriber, error)
	Confirm(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
	MarkBounced(ctx context.Context, email string) error
	MarkComplained(ctx context.Context, email string) error
	MarkSuppressed(ctx context.Context, email string) error
}
