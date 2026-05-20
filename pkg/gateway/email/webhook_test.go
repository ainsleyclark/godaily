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

package email_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
)

// webhookSecret is a valid whsec_ secret: a base64-encoded key.
var webhookSecret = "whsec_" + base64.StdEncoding.EncodeToString([]byte("godaily-webhook-test-secret-key!"))

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "examples", "webhooks", "resend", name))
	require.NoError(t, err, "reading fixture %s", name)
	return b
}

// sign produces a Svix-style v1 signature for the given content.
func sign(t *testing.T, secret, id, timestamp, payload string) string {
	t.Helper()
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	require.NoError(t, err)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(id + "." + timestamp + "." + payload))
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestParseWebhook(t *testing.T) {
	t.Parallel()

	t.Run("Decodes every fixture", func(t *testing.T) {
		t.Parallel()
		for _, name := range []string{"delivered.json", "opened.json", "clicked.json", "bounced.json", "complained.json"} {
			evt, err := email.ParseWebhook(loadFixture(t, name))
			require.NoError(t, err, name)
			assert.NotEmpty(t, evt.Type, name)
			assert.NotEmpty(t, evt.Data.EmailID, name)
		}
	})

	t.Run("Rejects invalid JSON", func(t *testing.T) {
		t.Parallel()
		_, err := email.ParseWebhook([]byte("{not json"))
		assert.Error(t, err)
	})
}

func TestToEmailEvent(t *testing.T) {
	t.Parallel()

	t.Run("Maps a clicked event with tags and URL", func(t *testing.T) {
		t.Parallel()

		evt, err := email.ParseWebhook(loadFixture(t, "clicked.json"))
		require.NoError(t, err)

		got, tracked, err := email.ToEmailEvent(evt, "msg_click")
		require.NoError(t, err)
		require.True(t, tracked)
		assert.Equal(t, engagement.EmailEventTypeClicked, got.Type)
		assert.Equal(t, "msg_click", got.EventID)
		assert.Equal(t, "re_delivered_abc123", got.ProviderID)
		assert.Equal(t, "reader@example.com", got.Email)
		assert.Equal(t, "https://go.dev/blog/go1.26", got.URL)
		require.NotNil(t, got.IssueID)
		assert.Equal(t, int64(128), *got.IssueID)
		require.NotNil(t, got.SubscriberID)
		assert.Equal(t, int64(42), *got.SubscriberID)
		assert.False(t, got.OccurredAt.IsZero())
	})

	t.Run("Maps bounced and complained events", func(t *testing.T) {
		t.Parallel()

		bounced, err := email.ParseWebhook(loadFixture(t, "bounced.json"))
		require.NoError(t, err)
		gotBounced, tracked, err := email.ToEmailEvent(bounced, "msg_b")
		require.NoError(t, err)
		require.True(t, tracked)
		assert.Equal(t, engagement.EmailEventTypeBounced, gotBounced.Type)
		assert.Equal(t, "dead-inbox@example.com", gotBounced.Email)

		complained, err := email.ParseWebhook(loadFixture(t, "complained.json"))
		require.NoError(t, err)
		gotComplained, tracked, err := email.ToEmailEvent(complained, "msg_c")
		require.NoError(t, err)
		require.True(t, tracked)
		assert.Equal(t, engagement.EmailEventTypeComplained, gotComplained.Type)
		assert.Equal(t, "unhappy@example.com", gotComplained.Email)
	})

	t.Run("Untracked event type is not tracked", func(t *testing.T) {
		t.Parallel()
		got, tracked, err := email.ToEmailEvent(email.WebhookEvent{Type: "email.sent"}, "msg")
		require.NoError(t, err)
		assert.False(t, tracked)
		assert.Zero(t, got)
	})

	t.Run("Accepts tags as an object", func(t *testing.T) {
		t.Parallel()
		evt := email.WebhookEvent{
			Type: "email.opened",
			Data: email.WebhookData{Tags: json.RawMessage(`{"issue_id":"5","subscriber_id":"9"}`)},
		}
		got, tracked, err := email.ToEmailEvent(evt, "m")
		require.NoError(t, err)
		require.True(t, tracked)
		require.NotNil(t, got.IssueID)
		assert.Equal(t, int64(5), *got.IssueID)
		require.NotNil(t, got.SubscriberID)
		assert.Equal(t, int64(9), *got.SubscriberID)
	})

	t.Run("Missing tags leave IDs nil", func(t *testing.T) {
		t.Parallel()
		got, _, err := email.ToEmailEvent(email.WebhookEvent{Type: "email.opened"}, "m")
		require.NoError(t, err)
		assert.Nil(t, got.IssueID)
		assert.Nil(t, got.SubscriberID)
	})

	t.Run("Non-numeric tag value leaves the ID nil", func(t *testing.T) {
		t.Parallel()
		evt := email.WebhookEvent{
			Type: "email.opened",
			Data: email.WebhookData{Tags: json.RawMessage(`{"issue_id":"not-a-number"}`)},
		}
		got, _, err := email.ToEmailEvent(evt, "m")
		require.NoError(t, err)
		assert.Nil(t, got.IssueID)
	})

	t.Run("Malformed timestamp falls back to zero", func(t *testing.T) {
		t.Parallel()
		got, _, err := email.ToEmailEvent(email.WebhookEvent{Type: "email.opened", CreatedAt: "nonsense"}, "m")
		require.NoError(t, err)
		assert.True(t, got.OccurredAt.IsZero())
	})
}

func TestVerifyWebhook(t *testing.T) {
	t.Parallel()

	const payload = `{"type":"email.opened"}`
	const id = "msg_verify"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	validSig := sign(t, webhookSecret, id, timestamp, payload)

	t.Run("Valid signature passes", func(t *testing.T) {
		t.Parallel()
		err := email.VerifyWebhook(payload, email.WebhookHeaders{ID: id, Timestamp: timestamp, Signature: validSig}, webhookSecret)
		assert.NoError(t, err)
	})

	t.Run("Tampered body fails", func(t *testing.T) {
		t.Parallel()
		err := email.VerifyWebhook(payload+" ", email.WebhookHeaders{ID: id, Timestamp: timestamp, Signature: validSig}, webhookSecret)
		assert.Error(t, err)
	})

	t.Run("Wrong secret fails", func(t *testing.T) {
		t.Parallel()
		other := "whsec_" + base64.StdEncoding.EncodeToString([]byte("a-totally-different-webhook-key!"))
		err := email.VerifyWebhook(payload, email.WebhookHeaders{ID: id, Timestamp: timestamp, Signature: validSig}, other)
		assert.Error(t, err)
	})

	t.Run("Missing signature header fails", func(t *testing.T) {
		t.Parallel()
		err := email.VerifyWebhook(payload, email.WebhookHeaders{ID: id, Timestamp: timestamp}, webhookSecret)
		assert.Error(t, err)
	})

	t.Run("Stale timestamp fails", func(t *testing.T) {
		t.Parallel()
		old := strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10)
		err := email.VerifyWebhook(payload, email.WebhookHeaders{
			ID:        id,
			Timestamp: old,
			Signature: sign(t, webhookSecret, id, old, payload),
		}, webhookSecret)
		assert.Error(t, err)
	})
}
