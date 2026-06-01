// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

// HasPosted reports whether a published featured row exists for the given
// issue and platform. Draft rows are excluded so a re-run of Build before
// publish never collides with itself.
func (s Store) HasPosted(ctx context.Context, issueID int64, platform string) (bool, error) {
	id := issueID
	return s.sqlc.SocialPostExists(ctx, sqlc.SocialPostExistsParams{
		IssueID:  &id,
		Platform: platform,
	})
}

// HasPostedBySubject reports whether any published row exists with the
// given subject and platform.
func (s Store) HasPostedBySubject(ctx context.Context, subject, platform string) (bool, error) {
	return s.sqlc.SocialPostExistsBySubject(ctx, sqlc.SocialPostExistsBySubjectParams{
		Subject:  dbtypes.NullString(subject),
		Platform: platform,
	})
}

// HasPostedKindSince reports whether any published row of the given kind
// on the given platform was posted at or after since.
func (s Store) HasPostedKindSince(ctx context.Context, kind social.PostKind, platform string, since time.Time) (bool, error) {
	return s.sqlc.SocialPostExistsKindSince(ctx, sqlc.SocialPostExistsKindSinceParams{
		Kind:     string(kind),
		Platform: platform,
		PostedAt: since,
	})
}

// Create persists a new social post record. PostedAt defaults to now when
// zero; Status defaults to Published (preserving legacy callers that
// inserted rows in their final state); Kind defaults to Featured.
func (s Store) Create(ctx context.Context, p social.Post) (social.Post, error) {
	postedAt := p.PostedAt
	if postedAt.IsZero() {
		postedAt = time.Now().UTC()
	}
	kind := p.Kind
	if kind == "" {
		kind = social.PostKindFeatured
	}
	status := p.Status
	if status == "" {
		status = social.PostStatusPublished
	}

	row, err := s.sqlc.SocialPostCreate(ctx, sqlc.SocialPostCreateParams{
		IssueID:       p.IssueID,
		Kind:          string(kind),
		Subject:       dbtypes.NullString(p.Subject),
		Platform:      p.Platform,
		Text:          p.Text,
		PostUrl:       dbtypes.NullString(p.PostURL),
		PostedAt:      postedAt,
		Status:        string(status),
		PublishedAt:   p.PublishedAt,
		MentionSource: dbtypes.NullString(p.MentionSource),
	})
	if err != nil {
		return social.Post{}, err
	}
	return transform(row), nil
}

// UpdateStatus transitions a row's status, published_at, and post_url.
// Used by the publish step to mark drafts as published or errored.
func (s Store) UpdateStatus(ctx context.Context, id int64, status social.PostStatus, publishedAt *time.Time, postURL string) (social.Post, error) {
	row, err := s.sqlc.SocialPostUpdateStatus(ctx, sqlc.SocialPostUpdateStatusParams{
		Status:      string(status),
		PublishedAt: publishedAt,
		PostUrl:     dbtypes.NullString(postURL),
		ID:          id,
	})
	if err != nil {
		return social.Post{}, err
	}
	return transform(row), nil
}

// DeleteDraftsByIssue removes any draft rows associated with the issue.
func (s Store) DeleteDraftsByIssue(ctx context.Context, issueID int64) error {
	id := issueID
	return s.sqlc.SocialPostDeleteDraftsByIssue(ctx, &id)
}

// List returns social posts filtered by opts. At least one filter must be set.
func (s Store) List(ctx context.Context, opts social.PostListOptions) ([]social.Post, error) {
	if opts.IssueID == nil && opts.Since == nil && opts.Status == nil && opts.Platform == nil {
		return nil, errors.New("List requires at least one filter")
	}

	params := sqlc.SocialPostListParams{}
	if opts.IssueID != nil {
		params.IssueID = *opts.IssueID
	}
	if opts.Since != nil {
		params.Since = *opts.Since
	}
	if opts.Status != nil {
		params.Status = string(*opts.Status)
	}
	if opts.Platform != nil {
		params.Platform = *opts.Platform
	}

	rows, err := s.sqlc.SocialPostList(ctx, params)
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
		ID:            r.ID,
		IssueID:       r.IssueID,
		Kind:          social.PostKind(r.Kind),
		Subject:       r.Subject.String,
		Platform:      r.Platform,
		Text:          r.Text,
		PostURL:       r.PostUrl.String,
		PostedAt:      r.PostedAt,
		Status:        social.PostStatus(r.Status),
		PublishedAt:   r.PublishedAt,
		MentionSource: r.MentionSource.String,
	}
}
