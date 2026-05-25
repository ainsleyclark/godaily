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
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	domengagement "github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

// SubscriberHealth applies the list-health side effects of email events.
// It is satisfied by the subscriber service.
type SubscriberHealth interface {
	MarkBounced(ctx context.Context, email string) error
	MarkComplained(ctx context.Context, email string) error
	MarkSuppressed(ctx context.Context, email string) error
}

// ItemFinder resolves a clicked URL back to the item it points at, scoped to
// an issue. It is satisfied by the items store.
type ItemFinder interface {
	FindByURLInIssue(ctx context.Context, issueID int64, url string) (int64, bool, error)
}

// EventService stores email events and applies their subscriber-health effects.
type EventService struct {
	events      domengagement.EmailEventRepository
	subscribers SubscriberHealth
	items       ItemFinder
	adminEmail  string
}

// NewEvents returns an EventService wired to the event store, subscriber health
// and item lookup. adminEmail is the operator address (EMAIL_SEND_ADDRESS);
// events for it and any @godaily.dev address are silently ignored.
func NewEvents(events domengagement.EmailEventRepository, subscribers SubscriberHealth, items ItemFinder, adminEmail string) *EventService {
	return &EventService{
		events:      events,
		subscribers: subscribers,
		items:       items,
		adminEmail:  adminEmail,
	}
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
var sideEffects = map[domengagement.EmailEventType]func(context.Context, *EventService, string) error{
	domengagement.EmailEventTypeBounced: func(ctx context.Context, s *EventService, addr string) error {
		return s.subscribers.MarkBounced(ctx, addr)
	},
	domengagement.EmailEventTypeComplained: func(ctx context.Context, s *EventService, addr string) error {
		return s.subscribers.MarkComplained(ctx, addr)
	},
	domengagement.EmailEventTypeSuppressed: func(ctx context.Context, s *EventService, addr string) error {
		return s.subscribers.MarkSuppressed(ctx, addr)
	},
}

// Process stores an email event and applies any subscriber-health side
// effect. Events whose EventID has already been stored are treated as
// duplicate webhook deliveries and skipped, making Process idempotent.
// Events addressed to the admin or any @godaily.dev address are silently
// dropped so internal traffic does not skew engagement stats.
func (s *EventService) Process(ctx context.Context, e domengagement.EmailEvent) error {
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

	if e.Type == domengagement.EmailEventTypeClicked && e.IssueID != nil && e.URL != "" {
		if id, ok, err := s.items.FindByURLInIssue(ctx, *e.IssueID, e.URL); err != nil {
			slog.WarnContext(ctx, "Item lookup for click event failed", "url", e.URL, "err", err)
		} else if ok {
			e.ItemID = &id
		}
	}

	if _, err := s.events.Create(ctx, e); err != nil {
		return errors.Wrap(err, "storing email event")
	}

	if effect, ok := sideEffects[e.Type]; ok {
		if err := effect(ctx, s, e.Email); err != nil {
			return errors.Wrap(err, "applying subscriber health change")
		}
	}
	return nil
}
