// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package engagement defines the domain model for measured engagement with
// GoDaily: email lifecycle events (opens, clicks, bounces, complaints) and
// the per-issue and per-link aggregates derived from them. It is the data
// foundation the growth loop reads from.
package engagement

import (
	"context"
	"time"
)

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

//go:generate go run go.uber.org/mock/mockgen -package=mockengagement -destination=../../mocks/engagement/EventService.go . EventService

// EventService stores email lifecycle events and applies their
// subscriber-health side effects (bounces, complaints, suppressions). It is
// the interface webhook handlers depend on so they can be tested without
// wiring the event store, subscriber service, and item lookup together.
type EventService interface {
	// Process stores an email event and applies any subscriber-health side
	// effect. Implementations should be idempotent on EmailEvent.EventID so
	// duplicate webhook deliveries do not double-count.
	Process(ctx context.Context, e EmailEvent) error
}

//go:generate go run go.uber.org/mock/mockgen -package=mockengagement -destination=../../mocks/engagement/EmailEventRepository.go . EmailEventRepository

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

	// TopLinks returns the most-clicked links for an issue, most clicks first.
	TopLinks(ctx context.Context, issueID int64, limit int64) ([]LinkClicks, error)
}

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

// LinkClicks counts clicks for a single link within an issue. Title, Tag and
// Source are resolved from the linked news item when the click maps to one;
// they are empty for links that don't (e.g. footer or CTA links).
type LinkClicks struct {
	URL    string `json:"url"`
	Title  string `json:"title,omitempty"`
	Tag    string `json:"tag,omitempty"`
	Source string `json:"source,omitempty"`
	Clicks int64  `json:"clicks"`
}
