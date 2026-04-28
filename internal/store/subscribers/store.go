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

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/store"
	"github.com/ainsleyclark/godaily/internal/store/internal/sqlc"
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

	// RandReader is the entropy source for confirm/unsubscribe tokens.
	// New defaults it to crypto/rand.Reader; tests may swap it for a
	// failing reader to exercise the token-generation error path.
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

func (s Store) FindByConfirmToken(ctx context.Context, token string) (news.Subscriber, error) {
	sub, err := s.sqlc.SubscriberByConfirmToken(ctx, token)

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

	confirm, err := newToken()
	if err != nil {
		return news.Subscriber{}, err
	}
	unsubscribe, err := newToken()
	if err != nil {
		return news.Subscriber{}, err
	}

	sub, err := s.sqlc.SubscriberCreate(ctx, sqlc.SubscriberCreateParams{
		Email:            email,
		ConfirmToken:     confirm,
		UnsubscribeToken: unsubscribe,
	})
	if err != nil {
		return news.Subscriber{}, err
	}

	return transformSubscriber(sub), nil
}

func (s Store) Confirm(ctx context.Context, token string) error {
	return s.sqlc.SubscriberConfirm(ctx, token)
}

func (s Store) Unsubscribe(ctx context.Context, token string) error {
	return s.sqlc.SubscriberUnsubscribe(ctx, token)
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
		ConfirmToken:     s.ConfirmToken,
		UnsubscribeToken: s.UnsubscribeToken,
		ConfirmedAt:      s.ConfirmedAt,
		UnsubscribedAt:   s.UnsubscribedAt,
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
