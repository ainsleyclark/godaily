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

// Package email defines the domain model for email lifecycle events —
// opens, clicks, bounces and complaints — received from the email provider,
// together with the engagement aggregates derived from them.
package email

import (
	"context"
	"time"
)

// EventType identifies an email lifecycle event that GoDaily tracks.
type EventType string

const (
	// EventTypeDelivered marks an email accepted by the recipient's server.
	// It is the denominator for open and click rates.
	EventTypeDelivered EventType = "delivered"
	// EventTypeOpened marks an email open. Treated as unreliable — Apple Mail
	// Privacy Protection pre-fetches images and inflates opens.
	EventTypeOpened EventType = "opened"
	// EventTypeClicked marks a link click. This is the primary engagement
	// signal.
	EventTypeClicked EventType = "clicked"
	// EventTypeBounced marks a hard delivery failure.
	EventTypeBounced EventType = "bounced"
	// EventTypeComplained marks a spam complaint.
	EventTypeComplained EventType = "complained"
)

var validEventTypes = map[EventType]bool{
	EventTypeDelivered:  true,
	EventTypeOpened:     true,
	EventTypeClicked:    true,
	EventTypeBounced:    true,
	EventTypeComplained: true,
}

// String returns the event type as a string.
func (t EventType) String() string {
	return string(t)
}

// Valid reports whether t is a recognised event type.
func (t EventType) Valid() bool {
	return validEventTypes[t]
}

// Event is a single email lifecycle event. IssueID and SubscriberID are
// optional: events for non-digest mail (such as confirmation emails), or for
// recipients that aren't tracked subscribers, still record — with the unknown
// identifier left nil.
type Event struct {
	ID           int64     `json:"id"`
	IssueID      *int64    `json:"issue_id,omitempty"`
	SubscriberID *int64    `json:"subscriber_id,omitempty"`
	Email        string    `json:"email"`
	Type         EventType `json:"type"`
	URL          string    `json:"url,omitempty"`
	ProviderID   string    `json:"provider_id,omitempty"`
	EventID      string    `json:"event_id"`
	OccurredAt   time.Time `json:"occurred_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// IssueStats aggregates engagement for a single digest issue. OpenRate and
// ClickRate are unique events over delivered; both are zero when nothing was
// delivered.
type IssueStats struct {
	IssueID      int64   `json:"issue_id"`
	Delivered    int64   `json:"delivered"`
	UniqueOpens  int64   `json:"unique_opens"`
	TotalOpens   int64   `json:"total_opens"`
	UniqueClicks int64   `json:"unique_clicks"`
	TotalClicks  int64   `json:"total_clicks"`
	Bounced      int64   `json:"bounced"`
	Complained   int64   `json:"complained"`
	OpenRate     float64 `json:"open_rate"`
	ClickRate    float64 `json:"click_rate"`
}

// LinkClicks counts clicks for a single link within an issue.
type LinkClicks struct {
	URL    string `json:"url"`
	Clicks int64  `json:"clicks"`
}

//go:generate go run go.uber.org/mock/mockgen -package=mockemail -destination=../../mocks/domain/email/EventRepository.go . EventRepository

// EventRepository persists email events and answers engagement aggregates.
type EventRepository interface {
	// Create persists an email event. OccurredAt defaults to now when zero.
	Create(ctx context.Context, e Event) (Event, error)

	// ExistsByEventID reports whether an event with the given provider event
	// ID has already been stored — the idempotency guard for webhook retries.
	ExistsByEventID(ctx context.Context, eventID string) (bool, error)

	// IssueStats returns aggregate engagement for a single issue.
	IssueStats(ctx context.Context, issueID int64) (IssueStats, error)

	// TopLinks returns the most-clicked links for an issue, most clicks first.
	TopLinks(ctx context.Context, issueID int64, limit int64) ([]LinkClicks, error)
}
