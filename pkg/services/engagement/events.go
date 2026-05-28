// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package engagement

import (
	"context"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	eng "github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

var _ eng.EventService = (*EventService)(nil)

// EventService stores email events and applies their subscriber-health effects.
type EventService struct {
	events      eng.EmailEventRepository
	subscribers subscriberHealth
	items       itemFinder
	adminEmail  string
}

// NewEvents returns an EventService wired to the event store, subscriber health
// and item lookup. adminEmail is the operator address (EMAIL_SEND_ADDRESS);
// events for it and any @godaily.dev address are silently ignored.
func NewEvents(events eng.EmailEventRepository, subscribers subscriberHealth, items itemFinder, adminEmail string) *EventService {
	return &EventService{
		events:      events,
		subscribers: subscribers,
		items:       items,
		adminEmail:  adminEmail,
	}
}

// subscriberHealth applies the list-health side effects of email events.
// It is satisfied by the subscriber service.
type subscriberHealth interface {
	MarkBounced(ctx context.Context, email string) error
	MarkComplained(ctx context.Context, email string) error
	MarkSuppressed(ctx context.Context, email string) error
}

// itemFinder resolves a clicked URL back to the item it points at, scoped to
// an issue. It is satisfied by the items store.
type itemFinder interface {
	FindByURLInIssue(ctx context.Context, issueID int64, url string) (int64, bool, error)
}

// Process stores an email event and applies any subscriber-health side
// effect. Events whose EventID has already been stored are treated as
// duplicate webhook deliveries and skipped, making Process idempotent.
// Events addressed to the admin or any @godaily.dev address are silently
// dropped so internal traffic does not skew engagement stats.
func (s *EventService) Process(ctx context.Context, e eng.EmailEvent) error {
	if s.isInternalEmail(e.Email) {
		return nil
	}

	exists, err := s.events.ExistsByEventID(ctx, e.EventID)
	if err != nil {
		return errors.Wrap(err, "checking for duplicate event")
	}
	if exists {
		return nil
	}

	if e.Type == eng.EmailEventTypeClicked && e.IssueID != nil && e.URL != "" {
		if id, ok, err := s.items.FindByURLInIssue(ctx, *e.IssueID, e.URL); err != nil {
			slog.WarnContext(ctx, "Item lookup for click event failed", "url", e.URL, "err", err)
		} else if ok {
			e.ItemID = &id
		}
	}

	if _, err = s.events.Create(ctx, e); err != nil {
		return errors.Wrap(err, "storing email event")
	}

	if effect, ok := sideEffects[e.Type]; ok {
		if err = effect(ctx, s, e.Email); err != nil {
			return errors.Wrap(err, "applying subscriber health change")
		}
	}

	return nil
}

// isInternalEmail reports whether addr belongs to the operator or the
// @godaily.dev domain and should be excluded from engagement tracking.
func (s *EventService) isInternalEmail(addr string) bool {
	lower := strings.ToLower(strings.TrimSpace(addr))
	return (s.adminEmail != "" && lower == strings.ToLower(strings.TrimSpace(s.adminEmail))) ||
		strings.HasSuffix(lower, "@godaily.dev")
}

// sideEffects maps an event type to the subscriber-health action it triggers.
// Event types without an entry are stored but carry no side effect.
//
// email.failed is intentionally absent: it fires when Resend cannot attempt
// delivery at all (quota, API key, domain config) rather than when the
// recipient's address is bad. Recipient-specific permanent failures produce
// email.bounced instead. Calling MarkBounced on failed events would silently
// deactivate valid subscribers during send-side outages.
var sideEffects = map[eng.EmailEventType]func(context.Context, *EventService, string) error{
	eng.EmailEventTypeBounced: func(ctx context.Context, s *EventService, addr string) error {
		return s.subscribers.MarkBounced(ctx, addr)
	},
	eng.EmailEventTypeComplained: func(ctx context.Context, s *EventService, addr string) error {
		return s.subscribers.MarkComplained(ctx, addr)
	},
	eng.EmailEventTypeSuppressed: func(ctx context.Context, s *EventService, addr string) error {
		return s.subscribers.MarkSuppressed(ctx, addr)
	},
}
