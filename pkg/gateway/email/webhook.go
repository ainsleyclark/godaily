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

package email

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/resend/resend-go/v3"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

type (
	// WebhookHeaders carries the Svix-style signature headers Resend sends with
	// every webhook request.
	WebhookHeaders struct {
		ID        string
		Timestamp string
		Signature string
	}
	// WebhookEvent is the JSON body Resend POSTs to the webhook endpoint.
	WebhookEvent struct {
		Type      string      `json:"type"`
		CreatedAt string      `json:"created_at"`
		Data      WebhookData `json:"data"`
	}
	// WebhookData is the per-event payload nested inside a WebhookEvent.
	WebhookData struct {
		EmailID string          `json:"email_id"`
		To      []string        `json:"to"`
		Subject string          `json:"subject"`
		Tags    json.RawMessage `json:"tags"`
		Click   *struct {
			Link string `json:"link"`
		} `json:"click"`
	}
)

// webhookEventTypes maps Resend's wire event names to GoDaily's canonical
// event types. Types absent from this map are not tracked.
var webhookEventTypes = map[string]engagement.EmailEventType{
	resend.EventEmailDelivered:  engagement.EmailEventTypeDelivered,
	resend.EventEmailOpened:     engagement.EmailEventTypeOpened,
	resend.EventEmailClicked:    engagement.EmailEventTypeClicked,
	resend.EventEmailBounced:    engagement.EmailEventTypeBounced,
	resend.EventEmailComplained: engagement.EmailEventTypeComplained,
}

// VerifyWebhook checks the Svix-style signature on a Resend webhook request.
// It returns a non-nil error when the signature is missing, malformed,
// expired or does not match the payload.
func VerifyWebhook(payload string, headers WebhookHeaders, secret string) error {
	// Verification is pure HMAC over the payload and secret — the client's
	// API token is not involved, so a tokenless client is sufficient.
	return resend.NewClient("").Webhooks.Verify(&resend.VerifyWebhookOptions{
		Payload: payload,
		Headers: resend.WebhookHeaders{
			Id:        headers.ID,
			Timestamp: headers.Timestamp,
			Signature: headers.Signature,
		},
		WebhookSecret: secret,
	})
}

// ParseWebhook decodes a Resend webhook request body.
func ParseWebhook(body []byte) (WebhookEvent, error) {
	var evt WebhookEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		return WebhookEvent{}, errors.Wrap(err, "decoding webhook payload")
	}
	return evt, nil
}

// ToEmailEvent maps a Resend webhook event to GoDaily's provider-agnostic
// domain event. eventID is the Svix message ID, used as the idempotency key.
// adminEmail is the operator address; events addressed to it or to any
// @godaily.dev address are acknowledged but not stored.
// The returned bool reports whether the event type is one GoDaily tracks;
// when false the event should be acknowledged and ignored.
func ToEmailEvent(evt WebhookEvent, eventID, adminEmail string) (engagement.EmailEvent, bool, error) {
	eventType, tracked := webhookEventTypes[evt.Type]
	if !tracked {
		return engagement.EmailEvent{}, false, nil
	}

	occurredAt, err := time.Parse(time.RFC3339, evt.CreatedAt)
	if err != nil {
		// A missing or malformed timestamp shouldn't drop the event; the
		// store defaults OccurredAt to now.
		occurredAt = time.Time{}
	}

	out := engagement.EmailEvent{
		Type:       eventType,
		EventID:    eventID,
		ProviderID: evt.Data.EmailID,
		OccurredAt: occurredAt,
	}
	if len(evt.Data.To) > 0 {
		out.Email = evt.Data.To[0]
	}
	if isInternalEmail(out.Email, adminEmail) {
		return engagement.EmailEvent{}, false, nil
	}
	if evt.Data.Click != nil {
		out.URL = evt.Data.Click.Link
	}

	tags := parseTags(evt.Data.Tags)
	out.IssueID = tagInt(tags, TagIssueID)
	out.SubscriberID = tagInt(tags, TagSubscriberID)

	// Only track events that are associated with a digest issue. Auxiliary
	// emails (subscribe confirmations, etc.) carry no issue_id tag.
	if out.IssueID == nil {
		return engagement.EmailEvent{}, false, nil
	}

	return out, true, nil
}

// parseTags reads the tags Resend echoes back. Tags are accepted both as an
// object (name → value) and as an array of {name, value} objects, since the
// send API uses the array form and webhook payloads have used either.
func parseTags(raw json.RawMessage) map[string]string {
	if len(raw) == 0 {
		return nil
	}

	var asObject map[string]string
	if err := json.Unmarshal(raw, &asObject); err == nil {
		return asObject
	}

	var asArray []resend.Tag
	if err := json.Unmarshal(raw, &asArray); err == nil {
		out := make(map[string]string, len(asArray))
		for _, t := range asArray {
			out[t.Name] = t.Value
		}
		return out
	}

	return nil
}

// isInternalEmail reports whether addr should be excluded from engagement
// tracking. It matches the configured admin address (case-insensitively) and
// any address in the @godaily.dev domain.
func isInternalEmail(addr, adminEmail string) bool {
	lower := strings.ToLower(strings.TrimSpace(addr))
	return (adminEmail != "" && lower == strings.ToLower(strings.TrimSpace(adminEmail))) ||
		strings.HasSuffix(lower, "@godaily.dev")
}

// tagInt parses a numeric tag value into an optional ID. A missing or
// non-numeric value yields nil rather than an error — correlation is
// best-effort and must never drop an event.
func tagInt(tags map[string]string, name string) *int64 {
	raw, ok := tags[name]
	if !ok {
		return nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &id
}
