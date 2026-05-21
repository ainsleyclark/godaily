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

// Package emailevent applies email lifecycle events: it persists every event
// and updates subscriber health on bounces and complaints. It is
// provider-agnostic — Resend specifics live in pkg/gateway/email.
package emailevent

import (
	"context"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

// SubscriberHealth applies the list-health side effects of email events.
// It is satisfied by the subscriber service.
type SubscriberHealth interface {
	MarkBounced(ctx context.Context, email string) error
	MarkComplained(ctx context.Context, email string) error
}

// Service stores email events and applies their subscriber-health effects.
type Service struct {
	events      engagement.EmailEventRepository
	subscribers SubscriberHealth
}

// New returns a Service wired to the event store and subscriber health.
func New(events engagement.EmailEventRepository, subscribers SubscriberHealth) *Service {
	return &Service{
		events:      events,
		subscribers: subscribers,
	}
}

// sideEffects maps an event type to the subscriber-health action it triggers.
// Event types without an entry are stored but carry no side effect.
var sideEffects = map[engagement.EmailEventType]func(context.Context, *Service, string) error{
	engagement.EmailEventTypeBounced: func(ctx context.Context, s *Service, addr string) error {
		return s.subscribers.MarkBounced(ctx, addr)
	},
	engagement.EmailEventTypeComplained: func(ctx context.Context, s *Service, addr string) error {
		return s.subscribers.MarkComplained(ctx, addr)
	},
}

// Process stores an email event and applies any subscriber-health side
// effect. Events whose EventID has already been stored are treated as
// duplicate webhook deliveries and skipped, making Process idempotent.
func (s *Service) Process(ctx context.Context, e engagement.EmailEvent) error {
	exists, err := s.events.ExistsByEventID(ctx, e.EventID)
	if err != nil {
		return errors.Wrap(err, "checking for duplicate event")
	}
	if exists {
		return nil
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
