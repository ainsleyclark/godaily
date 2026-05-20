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

// Package socialposts persists social media posts that have been published
// for a given digest issue. It provides idempotency for the social cron via
// HasPosted, and an audit log via ListForIssue.
package socialposts

import (
	"context"
	"database/sql"
	"time"

	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/pkg/store/internal/sqlc"
)

// New creates a new social posts Store.
func New(db *sql.DB) *Store {
	return &Store{
		sqlc: sqlc.New(db),
		db:   db,
	}
}

// Store provides methods for interacting with social_posts in the database.
type Store struct {
	sqlc *sqlc.Queries
	db   *sql.DB
}

var _ news.SocialPostRepository = (*Store)(nil)

// HasPosted reports whether a row exists for the given issue and platform.
func (s Store) HasPosted(ctx context.Context, issueID int64, platform string) (bool, error) {
	return s.sqlc.SocialPostExists(ctx, sqlc.SocialPostExistsParams{
		IssueID:  issueID,
		Platform: platform,
	})
}

// Create persists a new social post record. When PostedAt is the zero value
// it defaults to time.Now().UTC() so callers don't need to set it.
func (s Store) Create(ctx context.Context, p news.SocialPost) (news.SocialPost, error) {
	postedAt := p.PostedAt
	if postedAt.IsZero() {
		postedAt = time.Now().UTC()
	}

	row, err := s.sqlc.SocialPostCreate(ctx, sqlc.SocialPostCreateParams{
		IssueID:  p.IssueID,
		Platform: p.Platform,
		Text:     p.Text,
		PostUrl:  store.NullString(p.PostURL),
		PostedAt: postedAt,
	})
	if err != nil {
		return news.SocialPost{}, err
	}
	return transform(row), nil
}

// ListForIssue returns all posts associated with an issue, oldest first.
func (s Store) ListForIssue(ctx context.Context, issueID int64) ([]news.SocialPost, error) {
	rows, err := s.sqlc.SocialPostListByIssue(ctx, issueID)
	if err != nil {
		return nil, err
	}
	out := make([]news.SocialPost, 0, len(rows))
	for _, r := range rows {
		out = append(out, transform(r))
	}
	return out, nil
}

func transform(r sqlc.SocialPost) news.SocialPost {
	return news.SocialPost{
		ID:       r.ID,
		IssueID:  r.IssueID,
		Platform: r.Platform,
		Text:     r.Text,
		PostURL:  r.PostUrl.String,
		PostedAt: r.PostedAt,
	}
}
