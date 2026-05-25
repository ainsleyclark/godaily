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

// Package socialposts persists social media posts and their idempotency
// keys. Featured posts (kind='featured') are scoped per (issue_id, platform);
// rotation posts (recap, spotlight, cta, self_release) leave issue_id null
// and use a free-text Subject as their idempotency key.
package socialposts

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/store/internal/dbtypes"
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

var _ social.PostRepository = (*Store)(nil)

// HasPosted reports whether a featured row exists for the given issue and platform.
func (s Store) HasPosted(ctx context.Context, issueID int64, platform string) (bool, error) {
	id := issueID
	return s.sqlc.SocialPostExists(ctx, sqlc.SocialPostExistsParams{
		IssueID:  &id,
		Platform: platform,
	})
}

// HasPostedBySubject reports whether any row exists with the given subject and platform.
func (s Store) HasPostedBySubject(ctx context.Context, subject, platform string) (bool, error) {
	return s.sqlc.SocialPostExistsBySubject(ctx, sqlc.SocialPostExistsBySubjectParams{
		Subject:  dbtypes.NullString(subject),
		Platform: platform,
	})
}

// HasPostedKindSince reports whether any row of the given kind on the given
// platform was posted at or after since.
func (s Store) HasPostedKindSince(ctx context.Context, kind social.PostKind, platform string, since time.Time) (bool, error) {
	return s.sqlc.SocialPostExistsKindSince(ctx, sqlc.SocialPostExistsKindSinceParams{
		Kind:     string(kind),
		Platform: platform,
		PostedAt: since,
	})
}

// Create persists a new social post record. When PostedAt is the zero value
// it defaults to time.Now().UTC() so callers don't need to set it. Kind
// defaults to SocialPostKindFeatured to preserve the historical row shape.
func (s Store) Create(ctx context.Context, p social.Post) (social.Post, error) {
	postedAt := p.PostedAt
	if postedAt.IsZero() {
		postedAt = time.Now().UTC()
	}
	kind := p.Kind
	if kind == "" {
		kind = social.PostKindFeatured
	}

	row, err := s.sqlc.SocialPostCreate(ctx, sqlc.SocialPostCreateParams{
		IssueID:  p.IssueID,
		Kind:     string(kind),
		Subject:  dbtypes.NullString(p.Subject),
		Platform: p.Platform,
		Text:     p.Text,
		PostUrl:  dbtypes.NullString(p.PostURL),
		PostedAt: postedAt,
	})
	if err != nil {
		return social.Post{}, err
	}
	return transform(row), nil
}

// List returns social posts filtered by opts.
func (s Store) List(ctx context.Context, opts social.PostListOptions) ([]social.Post, error) {
	var rows []sqlc.SocialPost
	var err error
	switch {
	case opts.IssueID != nil:
		id := *opts.IssueID
		rows, err = s.sqlc.SocialPostListByIssue(ctx, &id)
	case opts.Since != nil:
		rows, err = s.sqlc.SocialPostListSince(ctx, *opts.Since)
	default:
		return nil, errors.New("List requires at least one option (IssueID or Since)")
	}
	if err != nil {
		return nil, err
	}
	out := make([]social.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, transform(r))
	}
	return out, nil
}

func transform(r sqlc.SocialPost) social.Post {
	return social.Post{
		ID:       r.ID,
		IssueID:  r.IssueID,
		Kind:     social.PostKind(r.Kind),
		Subject:  r.Subject.String,
		Platform: r.Platform,
		Text:     r.Text,
		PostURL:  r.PostUrl.String,
		PostedAt: r.PostedAt,
	}
}
