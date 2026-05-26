// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	"time"
)

// Post records a single published social media post. Featured posts link
// back to a digest issue via IssueID; rotation posts (recap, spotlight,
// cta, self_release) leave IssueID nil and use Subject as their idempotency
// key (e.g. "spotlight:ardanlabs", "self_release:v1.4.0", "recap:2026-W21").
type Post struct {
	ID       int64     `json:"id"`
	IssueID  *int64    `json:"issue_id,omitempty"`
	Kind     PostKind  `json:"kind"`
	Subject  string    `json:"subject,omitempty"`
	Platform string    `json:"platform"`
	Text     string    `json:"text"`
	PostURL  string    `json:"post_url,omitempty"`
	PostedAt time.Time `json:"posted_at"`
}

// PostKind classifies what flavour of social post a row represents.
// 'featured' rows pair with an issue and come from the daily 11 UTC slot;
// the rotation kinds run from the Tue/Fri rotation slot and use Subject
// for idempotency instead of IssueID.
type PostKind string

const (
	// PostKindFeatured is the daily AI-picked post anchored to an issue.
	PostKindFeatured PostKind = "featured"

	// PostKindNewSource announces that GoDaily started pulling from a
	// new source. Subject is "new_source:<source>".
	PostKindNewSource PostKind = "new_source"

	// PostKindRecap is the Friday weekly top-clicks post.
	PostKindRecap PostKind = "recap"

	// PostKindSpotlight tags and boosts a curated source.
	PostKindSpotlight PostKind = "spotlight"

	// PostKindCTA is a "sign up to GoDaily" rotation post.
	PostKindCTA PostKind = "cta"

	// PostKindCommunity is the Wednesday community-promo post that tags
	// a Go conference or meetup. Subject is "community:<slug>:<year>".
	PostKindCommunity PostKind = "community"
)

// PostListOptions filters a List query. At least one field must be set.
type PostListOptions struct {
	// IssueID restricts results to posts for a specific issue, oldest first.
	IssueID *int64
	// Since restricts results to posts with posted_at >= Since, newest first.
	Since *time.Time
}

//go:generate go run go.uber.org/mock/mockgen -package=mocksocial -destination=../../mocks/social/PostRepository.go . PostRepository

// PostRepository defines the methods for interacting with the social_posts store.
type PostRepository interface {
	// HasPosted reports whether a featured row exists for the given issue
	// and platform. Used by the daily featured slot.
	HasPosted(ctx context.Context, issueID int64, platform string) (bool, error)

	// HasPostedBySubject reports whether any row exists with the given
	// subject and platform. Used by rotation candidates that key off a
	// stable subject (release tag, source slug, recap week, cta variant).
	HasPostedBySubject(ctx context.Context, subject, platform string) (bool, error)

	// HasPostedKindSince reports whether any row of the given kind on the
	// given platform was posted at or after since. Used to throttle the
	// CTA and recap rotations.
	HasPostedKindSince(ctx context.Context, kind PostKind, platform string, since time.Time) (bool, error)

	// Create persists a new social post record.
	Create(ctx context.Context, p Post) (Post, error)

	// List returns social posts filtered by opts. At least one option must be set.
	List(ctx context.Context, opts PostListOptions) ([]Post, error)
}
