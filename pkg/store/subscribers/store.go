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

package subscribers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/pkg/store/internal/sqlc"
)

// New creates a new subscribers Store.
func New(db *sql.DB) *Store {
	return &Store{
		sqlc:       sqlc.New(db),
		db:         db,
		RandReader: rand.Reader,
	}
}

// Store provides methods for interacting with subscriber data
// in the database.
type Store struct {
	sqlc *sqlc.Queries
	db   *sql.DB

	// RandReader is the entropy source for the unsubscribe token.
	// New defaults it to crypto/rand.Reader; tests may swap it.
	RandReader io.Reader
}

var _ news.SubscriberRepository = (*Store)(nil)

func (s Store) Find(ctx context.Context, id int64) (news.Subscriber, error) {
	sub, err := s.sqlc.SubscriberByID(ctx, id)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return news.Subscriber{}, store.ErrNotFound
	} else if err != nil {
		return news.Subscriber{}, err
	}

	return transformSubscriber(sub), nil
}

func (s Store) FindByEmail(ctx context.Context, email string) (news.Subscriber, error) {
	sub, err := s.sqlc.SubscriberByEmail(ctx, email)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return news.Subscriber{}, store.ErrNotFound
	} else if err != nil {
		return news.Subscriber{}, err
	}

	return transformSubscriber(sub), nil
}

func (s Store) FindByUnsubscribeToken(ctx context.Context, token string) (news.Subscriber, error) {
	sub, err := s.sqlc.SubscriberByUnsubscribeToken(ctx, token)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return news.Subscriber{}, store.ErrNotFound
	} else if err != nil {
		return news.Subscriber{}, err
	}

	return transformSubscriber(sub), nil
}

func (s Store) Create(ctx context.Context, email string) (news.Subscriber, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return news.Subscriber{}, errors.New("email is required")
	}

	unsubscribe, err := newToken()
	if err != nil {
		return news.Subscriber{}, err
	}

	confirm, err := newToken()
	if err != nil {
		return news.Subscriber{}, err
	}

	sub, err := s.sqlc.SubscriberCreate(ctx, sqlc.SubscriberCreateParams{
		Email:            email,
		UnsubscribeToken: unsubscribe,
		ConfirmToken:     sql.NullString{String: confirm, Valid: true},
	})
	if err != nil {
		return news.Subscriber{}, err
	}

	return transformSubscriber(sub), nil
}

func (s Store) Reactivate(ctx context.Context, email string) (news.Subscriber, error) {
	unsubscribe, err := newToken()
	if err != nil {
		return news.Subscriber{}, err
	}
	confirm, err := newToken()
	if err != nil {
		return news.Subscriber{}, err
	}
	sub, err := s.sqlc.SubscriberReactivate(ctx, sqlc.SubscriberReactivateParams{
		ConfirmToken:     sql.NullString{String: confirm, Valid: true},
		UnsubscribeToken: unsubscribe,
		Email:            email,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return news.Subscriber{}, store.ErrNotFound
	} else if err != nil {
		return news.Subscriber{}, err
	}
	return transformSubscriber(sub), nil
}

func (s Store) Confirm(ctx context.Context, token string) (news.Subscriber, error) {
	sub, err := s.sqlc.SubscriberConfirm(ctx, sql.NullString{String: token, Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		return news.Subscriber{}, store.ErrNotFound
	} else if err != nil {
		return news.Subscriber{}, err
	}
	return transformSubscriber(sub), nil
}

func (s Store) Unsubscribe(ctx context.Context, token string) error {
	return s.sqlc.SubscriberUnsubscribe(ctx, token)
}

func (s Store) MarkBounced(ctx context.Context, email string) error {
	return s.sqlc.SubscriberMarkBounced(ctx, strings.ToLower(strings.TrimSpace(email)))
}

func (s Store) MarkComplained(ctx context.Context, email string) error {
	return s.sqlc.SubscriberMarkComplained(ctx, strings.ToLower(strings.TrimSpace(email)))
}

func (s Store) CountActive(ctx context.Context) (int64, error) {
	return s.sqlc.SubscriberCountActive(ctx)
}

func (s Store) ListActive(ctx context.Context) ([]news.Subscriber, error) {
	rows, err := s.sqlc.SubscriberListActive(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]news.Subscriber, 0, len(rows))
	for _, r := range rows {
		out = append(out, transformSubscriber(r))
	}
	return out, nil
}

func transformSubscriber(s sqlc.Subscriber) news.Subscriber {
	return news.Subscriber{
		ID:               s.ID,
		Email:            s.Email,
		UnsubscribeToken: s.UnsubscribeToken,
		ConfirmToken:     s.ConfirmToken.String,
		ConfirmedAt:      s.ConfirmedAt,
		UnsubscribedAt:   s.UnsubscribedAt,
		BouncedAt:        s.BouncedAt,
		ComplainedAt:     s.ComplainedAt,
		CreatedAt:        s.CreatedAt,
	}
}

// tokenBytes is the entropy size for confirm/unsubscribe tokens. 32 bytes
// becomes 43 base64url characters — enough to make brute-force discovery
// infeasible while still fitting comfortably in a URL.
const tokenBytes = 32

func newToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
