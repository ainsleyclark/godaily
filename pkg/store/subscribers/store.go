// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package subscribers implements domain/audience.SubscriberRepository backed by a SQL database.
package subscribers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"github.com/ainsleyclark/godaily/pkg/domain/audience"
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

var _ audience.SubscriberRepository = (*Store)(nil)

func (s Store) Find(ctx context.Context, id int64) (audience.Subscriber, error) {
	sub, err := s.sqlc.SubscriberByID(ctx, id)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return audience.Subscriber{}, store.ErrNotFound
	} else if err != nil {
		return audience.Subscriber{}, err
	}

	return transformSubscriber(sub), nil
}

func (s Store) FindByEmail(ctx context.Context, email string) (audience.Subscriber, error) {
	sub, err := s.sqlc.SubscriberByEmail(ctx, email)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return audience.Subscriber{}, store.ErrNotFound
	} else if err != nil {
		return audience.Subscriber{}, err
	}

	return transformSubscriber(sub), nil
}

func (s Store) FindByUnsubscribeToken(ctx context.Context, token string) (audience.Subscriber, error) {
	sub, err := s.sqlc.SubscriberByUnsubscribeToken(ctx, token)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return audience.Subscriber{}, store.ErrNotFound
	} else if err != nil {
		return audience.Subscriber{}, err
	}

	return transformSubscriber(sub), nil
}

func (s Store) Create(ctx context.Context, email string) (audience.Subscriber, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return audience.Subscriber{}, errors.New("email is required")
	}

	unsubscribe, err := newToken()
	if err != nil {
		return audience.Subscriber{}, err
	}

	confirm, err := newToken()
	if err != nil {
		return audience.Subscriber{}, err
	}

	sub, err := s.sqlc.SubscriberCreate(ctx, sqlc.SubscriberCreateParams{
		Email:            email,
		UnsubscribeToken: unsubscribe,
		ConfirmToken:     sql.NullString{String: confirm, Valid: true},
	})
	if err != nil {
		return audience.Subscriber{}, err
	}

	return transformSubscriber(sub), nil
}

func (s Store) Reactivate(ctx context.Context, email string) (audience.Subscriber, error) {
	unsubscribe, err := newToken()
	if err != nil {
		return audience.Subscriber{}, err
	}
	confirm, err := newToken()
	if err != nil {
		return audience.Subscriber{}, err
	}
	sub, err := s.sqlc.SubscriberReactivate(ctx, sqlc.SubscriberReactivateParams{
		ConfirmToken:     sql.NullString{String: confirm, Valid: true},
		UnsubscribeToken: unsubscribe,
		Email:            email,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return audience.Subscriber{}, store.ErrNotFound
	} else if err != nil {
		return audience.Subscriber{}, err
	}
	return transformSubscriber(sub), nil
}

func (s Store) Confirm(ctx context.Context, token string) (audience.Subscriber, error) {
	sub, err := s.sqlc.SubscriberConfirm(ctx, sql.NullString{String: token, Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		return audience.Subscriber{}, store.ErrNotFound
	} else if err != nil {
		return audience.Subscriber{}, err
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

func (s Store) MarkSuppressed(ctx context.Context, email string) error {
	return s.sqlc.SubscriberMarkSuppressed(ctx, strings.ToLower(strings.TrimSpace(email)))
}

func (s Store) MarkNudgeSent(ctx context.Context, id int64) error {
	return s.sqlc.SubscriberMarkNudgeSent(ctx, id)
}

func (s Store) CountActive(ctx context.Context) (int64, error) {
	return s.sqlc.SubscriberCountActive(ctx)
}

func (s Store) CountAll(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM subscribers").Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s Store) CountFiltered(ctx context.Context, search string) (int64, error) {
	var count int64
	if search == "" {
		return s.CountAll(ctx)
	}
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM subscribers WHERE email LIKE ?", "%"+search+"%").Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s Store) List(ctx context.Context, opts store.ListOptions) ([]audience.Subscriber, error) {
	query := "SELECT id, email, unsubscribe_token, COALESCE(confirm_token,''), confirmed_at, unsubscribed_at, bounced_at, suppressed_at, confirmation_nudge_sent_at, created_at FROM subscribers"
	args := make([]any, 0, 3)
	if opts.Search != "" {
		query += " WHERE email LIKE ?"
		args = append(args, "%"+opts.Search+"%")
	}
	query += " ORDER BY id ASC LIMIT ? OFFSET ?"
	args = append(args, opts.Limit(), opts.Offset())
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []audience.Subscriber
	for rows.Next() {
		var (
			sub                                    audience.Subscriber
			confirmedAt, unsubscribedAt, bouncedAt sql.NullTime
			suppressedAt, nudgeSentAt              sql.NullTime
		)
		if err := rows.Scan(
			&sub.ID, &sub.Email, &sub.UnsubscribeToken, &sub.ConfirmToken,
			&confirmedAt, &unsubscribedAt, &bouncedAt, &suppressedAt, &nudgeSentAt, &sub.CreatedAt,
		); err != nil {
			return nil, err
		}
		if confirmedAt.Valid {
			sub.ConfirmedAt = &confirmedAt.Time
		}
		if unsubscribedAt.Valid {
			sub.UnsubscribedAt = &unsubscribedAt.Time
		}
		if bouncedAt.Valid {
			sub.BouncedAt = &bouncedAt.Time
		}
		if suppressedAt.Valid {
			sub.SuppressedAt = &suppressedAt.Time
		}
		if nudgeSentAt.Valid {
			sub.ConfirmationNudgeSentAt = &nudgeSentAt.Time
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

func (s Store) ListActive(ctx context.Context) ([]audience.Subscriber, error) {
	rows, err := s.sqlc.SubscriberListActive(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]audience.Subscriber, 0, len(rows))
	for _, r := range rows {
		out = append(out, transformSubscriber(r))
	}
	return out, nil
}

func transformSubscriber(s sqlc.Subscriber) audience.Subscriber {
	return audience.Subscriber{
		ID:                      s.ID,
		Email:                   s.Email,
		UnsubscribeToken:        s.UnsubscribeToken,
		ConfirmToken:            s.ConfirmToken.String,
		ConfirmedAt:             s.ConfirmedAt,
		UnsubscribedAt:          s.UnsubscribedAt,
		BouncedAt:               s.BouncedAt,
		SuppressedAt:            s.SuppressedAt,
		ConfirmationNudgeSentAt: s.ConfirmationNudgeSentAt,
		CreatedAt:               s.CreatedAt,
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
