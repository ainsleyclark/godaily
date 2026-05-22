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

// Package engagement defines the domain model for measured engagement with
// GoDaily: email lifecycle events (opens, clicks, bounces, complaints) and
// the per-issue and per-link aggregates derived from them. It is the data
// foundation the growth loop reads from.
package engagement

import (
	"context"
	"time"
)

// EmailEventType identifies an email lifecycle event that GoDaily tracks.
type EmailEventType string

// Event type constants.
const (
	// EmailEventTypeDelivered marks an email accepted by the recipient's
	// server. It is the denominator for open and click rates.
	EmailEventTypeDelivered EmailEventType = "delivered"

	// EmailEventTypeOpened marks an email open. Treated as unreliable — Apple
	// Mail Privacy Protection pre-fetches images and inflates opens.
	EmailEventTypeOpened EmailEventType = "opened"

	// EmailEventTypeClicked marks a link click. This is the primary
	// engagement signal.
	EmailEventTypeClicked EmailEventType = "clicked"

	// EmailEventTypeBounced marks a hard delivery failure.
	EmailEventTypeBounced EmailEventType = "bounced"

	// EmailEventTypeComplained marks a spam complaint.
	EmailEventTypeComplained EmailEventType = "complained"

	// EmailEventTypeSuppressed marks an address on Resend's global suppression
	// list — delivery was refused before it was attempted.
	EmailEventTypeSuppressed EmailEventType = "suppressed"

	// EmailEventTypeDeliveryDelayed marks a temporary delivery delay. Resend
	// retries automatically; no subscriber health action is needed.
	EmailEventTypeDeliveryDelayed EmailEventType = "delivery_delayed"

	// EmailEventTypeFailed marks a permanent send failure (e.g. invalid MX
	// record). Unlike a bounce, this occurs before delivery is attempted.
	EmailEventTypeFailed EmailEventType = "failed"
)

var validEmailEventTypes = map[EmailEventType]bool{
	EmailEventTypeDelivered:       true,
	EmailEventTypeOpened:          true,
	EmailEventTypeClicked:         true,
	EmailEventTypeBounced:         true,
	EmailEventTypeComplained:      true,
	EmailEventTypeSuppressed:      true,
	EmailEventTypeDeliveryDelayed: true,
	EmailEventTypeFailed:          true,
}

// String returns the event type as a string.
func (t EmailEventType) String() string {
	return string(t)
}

// Valid reports whether t is a recognised event type.
func (t EmailEventType) Valid() bool {
	return validEmailEventTypes[t]
}

// EmailEvent is a single email lifecycle event. IssueID, SubscriberID and
// ItemID are optional: events for non-digest mail (such as confirmation
// emails), or for recipients that aren't tracked subscribers, still record —
// with the unknown identifier left nil. ItemID is best-effort: it is set only
// when a click resolves to a known item, and stays nil otherwise.
type EmailEvent struct {
	ID           int64          `json:"id"`
	IssueID      *int64         `json:"issue_id,omitempty"`
	SubscriberID *int64         `json:"subscriber_id,omitempty"`
	ItemID       *int64         `json:"item_id,omitempty"`
	Email        string         `json:"email"`
	Type         EmailEventType `json:"type"`
	URL          string         `json:"url,omitempty"`
	ProviderID   string         `json:"provider_id,omitempty"`
	EventID      string         `json:"event_id"`
	OccurredAt   time.Time      `json:"occurred_at"`
	CreatedAt    time.Time      `json:"created_at"`
}

// IssueStats aggregates email engagement for a single digest issue. OpenRate
// and ClickRate are unique events over delivered; both are zero when nothing
// was delivered.
type IssueStats struct {
	IssueID      int64   `json:"issue_id"`
	Delivered    int64   `json:"delivered"`
	UniqueOpens  int64   `json:"unique_opens"`
	TotalOpens   int64   `json:"total_opens"`
	UniqueClicks int64   `json:"unique_clicks"`
	TotalClicks  int64   `json:"total_clicks"`
	Bounced      int64   `json:"bounced"`
	Complained   int64   `json:"complained"`
	Delayed      int64   `json:"delayed"`
	Failed       int64   `json:"failed"`
	Suppressed   int64   `json:"suppressed"`
	OpenRate     float64 `json:"open_rate"`
	ClickRate    float64 `json:"click_rate"`
}

// LinkClicks counts clicks for a single link within an issue.
type LinkClicks struct {
	URL    string `json:"url"`
	Clicks int64  `json:"clicks"`
}

// ItemStats aggregates click engagement for a single news item across all issues.
type ItemStats struct {
	ItemID int64 `json:"item_id"`
	Clicks int64 `json:"clicks"`
}

//go:generate go run go.uber.org/mock/mockgen -package=mockengagement -destination=../../mocks/domain/engagement/EmailEventRepository.go . EmailEventRepository

// EmailEventRepository persists email events and answers engagement
// aggregates.
type EmailEventRepository interface {
	// Create persists an email event. OccurredAt defaults to now when zero.
	Create(ctx context.Context, e EmailEvent) (EmailEvent, error)

	// ExistsByEventID reports whether an event with the given provider event
	// ID has already been stored — the idempotency guard for webhook retries.
	ExistsByEventID(ctx context.Context, eventID string) (bool, error)

	// IssueStats returns aggregate engagement for a single issue.
	IssueStats(ctx context.Context, issueID int64) (IssueStats, error)

	// ListLinks returns the most-clicked links for an issue, most clicks first.
	ListLinks(ctx context.Context, issueID int64, limit int64) ([]LinkClicks, error)

	// ListIssueStats returns aggregate engagement for all issues that have events,
	// ordered by issue ID descending.
	ListIssueStats(ctx context.Context) ([]IssueStats, error)

	// ListItemStats returns click counts for all items that have been clicked,
	// ordered by clicks descending.
	ListItemStats(ctx context.Context) ([]ItemStats, error)
}
