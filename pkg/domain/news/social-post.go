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

package news

import (
	"context"
	"time"
)

// SocialPostKind classifies what flavour of social post a row represents.
// 'featured' rows pair with an issue and come from the daily 11 UTC slot;
// the rotation kinds run from the Tue/Fri rotation slot and use Subject
// for idempotency instead of IssueID.
type SocialPostKind string

const (
	// SocialPostKindFeatured is the daily AI-picked post anchored to an issue.
	SocialPostKindFeatured SocialPostKind = "featured"

	// SocialPostKindNewSource announces that GoDaily started pulling from a
	// new source. Subject is "new_source:<source>".
	SocialPostKindNewSource SocialPostKind = "new_source"

	// SocialPostKindRecap is the Friday weekly top-clicks post.
	SocialPostKindRecap SocialPostKind = "recap"

	// SocialPostKindSpotlight tags and boosts a curated source.
	SocialPostKindSpotlight SocialPostKind = "spotlight"

	// SocialPostKindCTA is a "sign up to GoDaily" rotation post.
	SocialPostKindCTA SocialPostKind = "cta"
)

// SocialPost records a single published social media post. Featured posts
// link back to a digest issue via IssueID; rotation posts (recap, spotlight,
// cta, self_release) leave IssueID nil and use Subject as their idempotency
// key (e.g. "spotlight:ardanlabs", "self_release:v1.4.0", "recap:2026-W21").
type SocialPost struct {
	ID       int64          `json:"id"`
	IssueID  *int64         `json:"issue_id,omitempty"`
	Kind     SocialPostKind `json:"kind"`
	Subject  string         `json:"subject,omitempty"`
	Platform string         `json:"platform"`
	Text     string         `json:"text"`
	PostURL  string         `json:"post_url,omitempty"`
	PostedAt time.Time      `json:"posted_at"`
}

// SocialPostListOptions filters a List query. At least one field must be set.
type SocialPostListOptions struct {
	// IssueID restricts results to posts for a specific issue, oldest first.
	IssueID *int64
	// Since restricts results to posts with posted_at >= Since, newest first.
	Since *time.Time
}

//go:generate go run go.uber.org/mock/mockgen -package=mocknews -destination=../../mocks/domain/news/SocialPostRepository.go . SocialPostRepository

// SocialPostRepository defines the methods for interacting with the
// social_posts store.
type SocialPostRepository interface {
	// HasPosted reports whether a featured row exists for the given issue
	// and platform. Used by the daily featured slot.
	HasPosted(ctx context.Context, issueID int64, platform string) (bool, error)

	// HasPostedBySubject reports whether any row exists with the given
	// subject and platform. Used by rotation candidates that key off a
	// stable subject (release tag, source slug, recap week, cta variant).
	HasPostedBySubject(ctx context.Context, subject, platform string) (bool, error)

	// HasPostedKindSince reports whether any row of the given kind on the
	// given platform was posted at or after `since`. Used to throttle the
	// CTA and recap rotations.
	HasPostedKindSince(ctx context.Context, kind SocialPostKind, platform string, since time.Time) (bool, error)

	// Create persists a new social post record.
	Create(ctx context.Context, p SocialPost) (SocialPost, error)

	// List returns social posts filtered by opts. At least one option must be set.
	List(ctx context.Context, opts SocialPostListOptions) ([]SocialPost, error)
}
