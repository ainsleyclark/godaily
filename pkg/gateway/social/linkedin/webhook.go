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

package linkedin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"
)

// WebhookEvent is the JSON body LinkedIn POSTs to the webhook endpoint.
type WebhookEvent struct {
	Events []struct {
		// EntityUrn is the URN of the post that received the engagement,
		// e.g. "urn:li:share:7234567890" or "urn:li:ugcPost:7234567890".
		EntityUrn string `json:"entityUrn"`
		EventType string `json:"eventType"`
	} `json:"events"`
}

// VerifyWebhook validates the LinkedIn webhook signature.
// LinkedIn computes HMAC-SHA256(clientSecret, rawBody) and base64-encodes it
// in the X-LI-Signature header.
func VerifyWebhook(body []byte, signature, clientSecret string) error {
	mac := hmac.New(sha256.New, []byte(clientSecret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return errors.New("signature mismatch")
	}
	return nil
}

// ParseWebhook decodes a LinkedIn webhook request body.
func ParseWebhook(body []byte) (WebhookEvent, error) {
	var evt WebhookEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		return WebhookEvent{}, errors.Wrap(err, "decoding webhook payload")
	}
	return evt, nil
}
