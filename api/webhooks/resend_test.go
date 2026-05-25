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

package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/mocks/engagement"
	"github.com/ainsleyclark/godaily/pkg/mocks/subscriber"
	svcengagement "github.com/ainsleyclark/godaily/pkg/services/engagement"
)

var webhookSecret = "whsec_" + base64.StdEncoding.EncodeToString([]byte("godaily-handler-test-secret-key!"))

// noopItemFinder satisfies engagement.ItemFinder, resolving no items. The
// webhook fixtures exercised here carry no click events, so it is never hit.
type noopItemFinder struct{}

func (noopItemFinder) FindByURLInIssue(context.Context, int64, string) (int64, bool, error) {
	return 0, false, nil
}

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "examples", "webhooks", "resend", name))
	require.NoError(t, err, "reading fixture %s", name)
	return string(b)
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

func newApp(t *testing.T, secret string) (*godaily.App, *mockengagement.MockEmailEventRepository, *mocksubscriber.MockService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	events := mockengagement.NewMockEmailEventRepository(ctrl)
	subs := mocksubscriber.NewMockService(ctrl)
	return &godaily.App{
		Config:      &env.Config{ResendWebhookSecret: secret},
		EmailEvents: svcengagement.NewEvents(events, subs, noopItemFinder{}, ""),
	}, events, subs
}

// signedPOST builds a Resend-style signed webhook POST request.
func signedPOST(t *testing.T, secret, body string) *http.Request {
	t.Helper()
	id := "msg_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	r := httptest.NewRequest(http.MethodPost, "/api/webhooks/resend", strings.NewReader(body))
	r.Header.Set("svix-id", id)
	r.Header.Set("svix-timestamp", ts)
	r.Header.Set("svix-signature", sign(t, secret, id, ts, body))
	return r
}

func do(t *testing.T, a *godaily.App, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	Handler(w, r.WithContext(api.WithApp(r.Context(), a)))
	return w
}

func TestHandler(t *testing.T) {
	t.Run("Valid signed event is processed", func(t *testing.T) {
		a, events, _ := newApp(t, webhookSecret)
		events.EXPECT().ExistsByEventID(gomock.Any(), gomock.Any()).Return(false, nil)
		events.EXPECT().Create(gomock.Any(), gomock.Any()).Return(engagement.EmailEvent{}, nil)

		w := do(t, a, signedPOST(t, webhookSecret, loadFixture(t, "delivered.json")))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Bounced event marks the subscriber", func(t *testing.T) {
		a, events, subs := newApp(t, webhookSecret)
		events.EXPECT().ExistsByEventID(gomock.Any(), gomock.Any()).Return(false, nil)
		events.EXPECT().Create(gomock.Any(), gomock.Any()).Return(engagement.EmailEvent{}, nil)
		subs.EXPECT().MarkBounced(gomock.Any(), "dead-inbox@example.com").Return(nil)

		w := do(t, a, signedPOST(t, webhookSecret, loadFixture(t, "bounced.json")))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid signature is rejected", func(t *testing.T) {
		a, _, _ := newApp(t, webhookSecret)
		r := signedPOST(t, webhookSecret, "{}")
		r.Header.Set("svix-signature", "v1,not-a-real-signature")

		w := do(t, a, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Malformed body is rejected", func(t *testing.T) {
		a, _, _ := newApp(t, webhookSecret)
		w := do(t, a, signedPOST(t, webhookSecret, "{not json"))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Non-POST is rejected", func(t *testing.T) {
		a, _, _ := newApp(t, webhookSecret)
		r := httptest.NewRequest(http.MethodGet, "/api/webhooks/resend", nil)
		w := do(t, a, r)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("Missing secret returns a server error", func(t *testing.T) {
		a, _, _ := newApp(t, "")
		w := do(t, a, signedPOST(t, webhookSecret, "{}"))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Untracked event type is acknowledged", func(t *testing.T) {
		a, _, _ := newApp(t, webhookSecret)
		w := do(t, a, signedPOST(t, webhookSecret, `{"type":"email.sent","data":{}}`))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Duplicate delivery is acknowledged", func(t *testing.T) {
		a, events, _ := newApp(t, webhookSecret)
		events.EXPECT().ExistsByEventID(gomock.Any(), gomock.Any()).Return(true, nil)

		w := do(t, a, signedPOST(t, webhookSecret, loadFixture(t, "opened.json")))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Processing failure returns a server error", func(t *testing.T) {
		a, events, _ := newApp(t, webhookSecret)
		events.EXPECT().ExistsByEventID(gomock.Any(), gomock.Any()).Return(false, nil)
		events.EXPECT().Create(gomock.Any(), gomock.Any()).Return(engagement.EmailEvent{}, errors.New("db down"))

		w := do(t, a, signedPOST(t, webhookSecret, loadFixture(t, "opened.json")))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
