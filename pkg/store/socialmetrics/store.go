// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package socialmetrics persists per-post social engagement counts (likes,
// reposts, comments, impressions) for every platform GoDaily posts to.
package socialmetrics

import (
	"context"
	"database/sql"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/store/internal/sqlc"
)

var _ engagement.SocialMetricRepository = (*Store)(nil)

// New creates a new social metrics Store.
func New(db *sql.DB) *Store {
	return &Store{
		sqlc: sqlc.New(db),
	}
}

// Store provides methods for interacting with social_metrics in the database.
type Store struct {
	sqlc *sqlc.Queries
}

// Upsert inserts or replaces the metrics for a (social_post_id, platform) pair.
// FetchedAt defaults to now when zero.
func (s Store) Upsert(ctx context.Context, m engagement.SocialMetric) error {
	fetchedAt := m.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	return s.sqlc.SocialMetricUpsert(ctx, sqlc.SocialMetricUpsertParams{
		SocialPostID: m.SocialPostID,
		Platform:     m.Platform,
		Likes:        m.Likes,
		Reposts:      m.Reposts,
		Comments:     m.Comments,
		Impressions:  m.Impressions,
		FetchedAt:    fetchedAt,
	})
}

// ListBySocialPostID returns all metric rows for a given social post.
func (s Store) ListBySocialPostID(ctx context.Context, socialPostID int64) ([]engagement.SocialMetric, error) {
	rows, err := s.sqlc.SocialMetricListBySocialPostID(ctx, socialPostID)
	if err != nil {
		return nil, err
	}
	out := make([]engagement.SocialMetric, 0, len(rows))
	for _, r := range rows {
		out = append(out, transformMetric(r))
	}
	return out, nil
}

// List returns social posts joined with their latest engagement counts,
// filtered by the date range in f.
func (s Store) List(ctx context.Context, f engagement.MetricsFilter) ([]engagement.SocialPostEngagement, error) {
	var fromArg, toArg interface{}
	if f.From != nil {
		fromArg = *f.From
	}
	if f.To != nil {
		// Bump by one day so the half-open bound covers the full selected end date.
		// parseDateWindow parses YYYY-MM-DD as midnight; without this, posts made
		// later that day would be excluded.
		exclusive := f.To.AddDate(0, 0, 1)
		toArg = exclusive
	}
	rows, err := s.sqlc.SocialPostsWithMetrics(ctx, sqlc.SocialPostsWithMetricsParams{
		From: fromArg,
		To:   toArg,
	})
	if err != nil {
		return nil, err
	}
	out := make([]engagement.SocialPostEngagement, 0, len(rows))
	for _, r := range rows {
		out = append(out, engagement.SocialPostEngagement{
			ID:          r.ID,
			IssueID:     r.IssueID,
			Kind:        r.Kind,
			Subject:     r.Subject.String,
			Platform:    r.Platform,
			Text:        r.Text,
			PostURL:     r.PostUrl.String,
			PostedAt:    r.PostedAt,
			Likes:       r.Likes,
			Reposts:     r.Reposts,
			Comments:    r.Comments,
			Impressions: r.Impressions,
		})
	}
	return out, nil
}

func transformMetric(r sqlc.SocialMetric) engagement.SocialMetric {
	return engagement.SocialMetric{
		ID:           r.ID,
		SocialPostID: r.SocialPostID,
		Platform:     r.Platform,
		Likes:        r.Likes,
		Reposts:      r.Reposts,
		Comments:     r.Comments,
		Impressions:  r.Impressions,
		FetchedAt:    r.FetchedAt,
	}
}
