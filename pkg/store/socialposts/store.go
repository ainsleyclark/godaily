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
//
// This store is hand-written rather than generated to keep the small,
// social-specific schema isolated from the larger sqlc-managed surface.
package socialposts

import (
	"context"
	"database/sql"
	"time"

	"github.com/ainsleyclark/godaily/pkg/news"
)

// New creates a new social posts Store.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// Store provides methods for interacting with social_posts in the database.
type Store struct {
	db *sql.DB
}

var _ news.SocialPostRepository = (*Store)(nil)

const hasPostedSQL = `
SELECT EXISTS (
    SELECT 1 FROM social_posts
    WHERE issue_id = ? AND platform = ?
)`

// HasPosted reports whether a row exists for the given issue and platform.
func (s Store) HasPosted(ctx context.Context, issueID int64, platform string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, hasPostedSQL, issueID, platform).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

const createSQL = `
INSERT INTO social_posts (issue_id, platform, text, post_url, posted_at)
VALUES (?, ?, ?, ?, ?)
RETURNING id, issue_id, platform, text, post_url, posted_at`

// Create persists a new social post record. When PostedAt is the zero value
// it defaults to time.Now().UTC() so callers don't need to set it.
func (s Store) Create(ctx context.Context, p news.SocialPost) (news.SocialPost, error) {
	postedAt := p.PostedAt
	if postedAt.IsZero() {
		postedAt = time.Now().UTC()
	}

	var postURLArg any
	if p.PostURL == "" {
		postURLArg = nil
	} else {
		postURLArg = p.PostURL
	}

	var (
		out     news.SocialPost
		postURL sql.NullString
	)
	err := s.db.QueryRowContext(ctx, createSQL,
		p.IssueID, p.Platform, p.Text, postURLArg, postedAt,
	).Scan(&out.ID, &out.IssueID, &out.Platform, &out.Text, &postURL, &out.PostedAt)
	if err != nil {
		return news.SocialPost{}, err
	}
	out.PostURL = postURL.String
	return out, nil
}

const listByIssueSQL = `
SELECT id, issue_id, platform, text, post_url, posted_at
FROM social_posts
WHERE issue_id = ?
ORDER BY posted_at ASC`

// ListForIssue returns all posts associated with an issue, oldest first.
func (s Store) ListForIssue(ctx context.Context, issueID int64) ([]news.SocialPost, error) {
	rows, err := s.db.QueryContext(ctx, listByIssueSQL, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]news.SocialPost, 0)
	for rows.Next() {
		var (
			p       news.SocialPost
			postURL sql.NullString
		)
		if err := rows.Scan(&p.ID, &p.IssueID, &p.Platform, &p.Text, &postURL, &p.PostedAt); err != nil {
			return nil, err
		}
		p.PostURL = postURL.String
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
