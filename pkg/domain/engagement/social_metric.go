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

package engagement

import (
	"context"
	"time"
)

// SocialMetric holds the latest engagement counts for a single social post.
// One row per (social_post_id, platform) — upserted on each fetch.
type SocialMetric struct {
	ID           int64     `json:"id"`
	SocialPostID int64     `json:"social_post_id"`
	Platform     string    `json:"platform"`
	Likes        int64     `json:"likes"`
	Reposts      int64     `json:"reposts"`
	Comments     int64     `json:"comments"`
	Impressions  int64     `json:"impressions"`
	FetchedAt    time.Time `json:"fetched_at"`
}

//go:generate go run go.uber.org/mock/mockgen -package=mockengagement -destination=../../mocks/domain/engagement/SocialMetricRepository.go . SocialMetricRepository

// SocialMetricRepository persists social engagement metrics.
type SocialMetricRepository interface {
	// Upsert inserts or replaces the metrics for a (social_post_id, platform) pair.
	Upsert(ctx context.Context, m SocialMetric) error

	// ListBySocialPostID returns all metric rows for a given social post.
	ListBySocialPostID(ctx context.Context, socialPostID int64) ([]SocialMetric, error)
}
