// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package emailevents persists email lifecycle events (delivered, opened,
// clicked, bounced, complained) and answers the per-issue and per-link
// engagement aggregates that feed GoDaily's growth loop.
package emailevents

import (
	"context"
	"database/sql"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/store/internal/dbtypes"
	"github.com/ainsleyclark/godaily/pkg/store/internal/sqlc"
)

// New creates a new email events Store.
func New(db *sql.DB) *Store {
	return &Store{
		sqlc: sqlc.New(db),
		db:   db,
	}
}

// Store provides methods for interacting with email_events in the database.
type Store struct {
	sqlc *sqlc.Queries
	db   *sql.DB
}

var _ engagement.EmailEventRepository = (*Store)(nil)

// Create persists an email event. When OccurredAt is the zero value it
// defaults to time.Now().UTC() so callers don't need to set it.
func (s Store) Create(ctx context.Context, e engagement.EmailEvent) (engagement.EmailEvent, error) {
	occurredAt := e.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	row, err := s.sqlc.EmailEventCreate(ctx, sqlc.EmailEventCreateParams{
		IssueID:      nullInt64(e.IssueID),
		SubscriberID: nullInt64(e.SubscriberID),
		ItemID:       nullInt64(e.ItemID),
		Email:        e.Email,
		EventType:    e.Type.String(),
		Url:          dbtypes.NullString(e.URL),
		ProviderID:   dbtypes.NullString(e.ProviderID),
		EventID:      e.EventID,
		OccurredAt:   occurredAt,
	})
	if err != nil {
		return engagement.EmailEvent{}, err
	}
	return transform(row), nil
}

// ExistsByEventID reports whether an event with the given provider event ID
// has already been stored.
func (s Store) ExistsByEventID(ctx context.Context, eventID string) (bool, error) {
	return s.sqlc.EmailEventExistsByEventID(ctx, eventID)
}

// IssueStats returns aggregate engagement for a single issue. Open and click
// rates are derived in Go to keep the SQL a plain set of counts.
func (s Store) IssueStats(ctx context.Context, issueID int64) (engagement.IssueStats, error) {
	row, err := s.sqlc.EmailEventIssueStats(ctx, sql.NullInt64{Int64: issueID, Valid: true})
	if err != nil {
		return engagement.IssueStats{}, err
	}

	stats := engagement.IssueStats{
		IssueID:      issueID,
		Delivered:    row.Delivered,
		UniqueOpens:  row.UniqueOpens,
		TotalOpens:   row.TotalOpens,
		UniqueClicks: row.UniqueClicks,
		TotalClicks:  row.TotalClicks,
		Bounced:      row.Bounced,
		Complained:   row.Complained,
		Delayed:      row.Delayed,
		Failed:       row.Failed,
		Suppressed:   row.Suppressed,
	}
	if stats.Delivered > 0 {
		stats.OpenRate = float64(stats.UniqueOpens) / float64(stats.Delivered)
		stats.ClickRate = float64(stats.UniqueClicks) / float64(stats.Delivered)
	}
	return stats, nil
}

// TopLinks returns the most-clicked links for an issue, most clicks first.
func (s Store) TopLinks(ctx context.Context, issueID int64, limit int64) ([]engagement.LinkClicks, error) {
	rows, err := s.sqlc.EmailEventTopLinks(ctx, sqlc.EmailEventTopLinksParams{
		IssueID: sql.NullInt64{Int64: issueID, Valid: true},
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}

	out := make([]engagement.LinkClicks, 0, len(rows))
	for _, r := range rows {
		out = append(out, engagement.LinkClicks{
			URL:    r.Url.String,
			Title:  r.Title.String,
			Tag:    r.Tag.String,
			Source: r.Source.String,
			Clicks: r.Clicks,
		})
	}
	return out, nil
}

func transform(r sqlc.EmailEvent) engagement.EmailEvent {
	return engagement.EmailEvent{
		ID:           r.ID,
		IssueID:      int64Ptr(r.IssueID),
		SubscriberID: int64Ptr(r.SubscriberID),
		ItemID:       int64Ptr(r.ItemID),
		Email:        r.Email,
		Type:         engagement.EmailEventType(r.EventType),
		URL:          r.Url.String,
		ProviderID:   r.ProviderID.String,
		EventID:      r.EventID,
		OccurredAt:   r.OccurredAt,
		CreatedAt:    r.CreatedAt,
	}
}

// nullInt64 converts an optional ID to sql.NullInt64.
func nullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

// int64Ptr converts a sql.NullInt64 back to an optional ID.
func int64Ptr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}
