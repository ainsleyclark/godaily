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

// SocialPost records a single published social media post tied back to the
// digest issue it originated from. Used both as an audit log and as the
// idempotency guard against retried crons.
type SocialPost struct {
	ID       int64     `json:"id"`
	IssueID  int64     `json:"issue_id"`
	Platform string    `json:"platform"`
	Text     string    `json:"text"`
	PostURL  string    `json:"post_url,omitempty"`
	PostedAt time.Time `json:"posted_at"`
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
	// HasPosted reports whether a row exists for the given issue and platform.
	HasPosted(ctx context.Context, issueID int64, platform string) (bool, error)

	// Create persists a new social post record.
	Create(ctx context.Context, p SocialPost) (SocialPost, error)

	// List returns social posts filtered by opts. At least one option must be set.
	List(ctx context.Context, opts SocialPostListOptions) ([]SocialPost, error)
}
