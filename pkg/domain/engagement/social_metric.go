// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

// SocialPostEngagement joins a social post with its latest engagement counts.
type SocialPostEngagement struct {
	ID          int64     `json:"id"`
	IssueID     *int64    `json:"issue_id,omitempty"`
	Kind        string    `json:"kind"`
	Subject     string    `json:"subject,omitempty"`
	Platform    string    `json:"platform"`
	Text        string    `json:"text"`
	PostURL     string    `json:"post_url,omitempty"`
	PostedAt    time.Time `json:"posted_at"`
	Likes       int64     `json:"likes"`
	Reposts     int64     `json:"reposts"`
	Comments    int64     `json:"comments"`
	Impressions int64     `json:"impressions"`
}

//go:generate go run go.uber.org/mock/mockgen -package=mockengagement -destination=../../mocks/engagement/SocialMetricRepository.go . SocialMetricRepository

// SocialMetricRepository persists social engagement metrics.
type SocialMetricRepository interface {
	// Upsert inserts or replaces the metrics for a (social_post_id, platform) pair.
	Upsert(ctx context.Context, m SocialMetric) error

	// ListBySocialPostID returns all metric rows for a given social post.
	ListBySocialPostID(ctx context.Context, socialPostID int64) ([]SocialMetric, error)

	// List returns social posts joined with their latest engagement counts,
	// filtered by the date range in f.
	List(ctx context.Context, f MetricsFilter) ([]SocialPostEngagement, error)
}
