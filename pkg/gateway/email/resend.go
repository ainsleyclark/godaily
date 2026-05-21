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
	"context"
	"log/slog"

	"github.com/resend/resend-go/v3"
)

// Sender is satisfied by any type that can dispatch a transactional email.
// Both pkg/digest and pkg/subscriber depend on this interface rather than
// defining their own copies.
type Sender interface {
	Send(ctx context.Context, req SendEmailRequest) error
}

// Client wraps the Resend API client and exposes a minimal surface for
// dispatching transactional emails from godaily.
type Client struct {
	resend *resend.Client
}

// New returns a Client authenticated with the given Resend API token.
func New(token string) *Client {
	return &Client{
		resend: resend.NewClient(token),
	}
}

// SendEmailRequest is the payload accepted by Send.
// Alias for resend.SendEmailRequest.
type SendEmailRequest = resend.SendEmailRequest

// Tag is a custom key/value label attached to an outbound email. Resend
// echoes tags back on the webhook events for that email, so they are how
// GoDaily correlates an event to its issue and subscriber.
// Alias for resend.Tag.
type Tag = resend.Tag

// Tag names attached to outbound digest emails. Resend echoes these back on
// every webhook event for the email, so they are the single contract shared
// by the send path and the webhook reader — define them once, here.
const (
	TagIssueID      = "issue_id"
	TagSubscriberID = "subscriber_id"
)

// Send dispatches req via Resend and logs the resulting message ID on success.
func (c Client) Send(ctx context.Context, req SendEmailRequest) error {
	sent, err := c.resend.Emails.Send(&req)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "Successfully sent email", "id", sent.Id, "subject", req.Subject)
	return nil
}
