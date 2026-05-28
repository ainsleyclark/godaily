// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webhooks

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

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/mocks/audience"
	"github.com/ainsleyclark/godaily/pkg/mocks/engagement"
	svcengagement "github.com/ainsleyclark/godaily/pkg/services/engagement"
)

var webhookSecret = "whsec_" + base64.StdEncoding.EncodeToString([]byte("godaily-handler-test-secret-key!"))

// noopItemFinder satisfies engagement.ItemFinder, resolving no items.
type noopItemFinder struct{}

func (noopItemFinder) FindByURLInIssue(context.Context, int64, string) (int64, bool, error) {
	return 0, false, nil
}

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "examples", "webhooks", "resend", name))
	require.NoError(t, err, "reading fixture %s", name)
	return string(b)
}

func sign(t *testing.T, secret, id, timestamp, payload string) string {
	t.Helper()
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	require.NoError(t, err)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(id + "." + timestamp + "." + payload))
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func newHandler(t *testing.T, secret string) (*Handler, *mockengagement.MockEmailEventRepository, *mockaudience.MockSubscriberService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	events := mockengagement.NewMockEmailEventRepository(ctrl)
	subs := mockaudience.NewMockSubscriberService(ctrl)
	return &Handler{
		emailEvents: svcengagement.NewEvents(events, subs, noopItemFinder{}, ""),
		config:      &env.Config{ResendWebhookSecret: secret},
	}, events, subs
}

func signedPOST(t *testing.T, secret, body string) *http.Request {
	t.Helper()
	id := "msg_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	r := httptest.NewRequest(http.MethodPost, "/webhooks/resend", strings.NewReader(body))
	r.Header.Set("svix-id", id)
	r.Header.Set("svix-timestamp", ts)
	r.Header.Set("svix-signature", sign(t, secret, id, ts, body))
	return r
}

func do(t *testing.T, h *Handler, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	invoke(h.Resend, w, r)
	return w
}

func TestHandleResend(t *testing.T) {
	t.Parallel()

	t.Run("Valid signed event is processed", func(t *testing.T) {
		t.Parallel()

		h, events, _ := newHandler(t, webhookSecret)
		events.EXPECT().ExistsByEventID(gomock.Any(), gomock.Any()).Return(false, nil)
		events.EXPECT().Create(gomock.Any(), gomock.Any()).Return(engagement.EmailEvent{}, nil)

		w := do(t, h, signedPOST(t, webhookSecret, loadFixture(t, "delivered.json")))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Bounced event marks the subscriber", func(t *testing.T) {
		t.Parallel()

		h, events, subs := newHandler(t, webhookSecret)
		events.EXPECT().ExistsByEventID(gomock.Any(), gomock.Any()).Return(false, nil)
		events.EXPECT().Create(gomock.Any(), gomock.Any()).Return(engagement.EmailEvent{}, nil)
		subs.EXPECT().MarkBounced(gomock.Any(), "dead-inbox@example.com").Return(nil)

		w := do(t, h, signedPOST(t, webhookSecret, loadFixture(t, "bounced.json")))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid signature is rejected", func(t *testing.T) {
		t.Parallel()

		h, _, _ := newHandler(t, webhookSecret)
		r := signedPOST(t, webhookSecret, "{}")
		r.Header.Set("svix-signature", "v1,not-a-real-signature")

		w := do(t, h, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Malformed body is rejected", func(t *testing.T) {
		t.Parallel()

		h, _, _ := newHandler(t, webhookSecret)
		w := do(t, h, signedPOST(t, webhookSecret, "{not json"))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Missing secret returns a bad request", func(t *testing.T) {
		t.Parallel()

		h, _, _ := newHandler(t, "")
		w := do(t, h, signedPOST(t, webhookSecret, "{}"))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Untracked event type is acknowledged", func(t *testing.T) {
		t.Parallel()

		h, _, _ := newHandler(t, webhookSecret)
		w := do(t, h, signedPOST(t, webhookSecret, `{"type":"email.sent","data":{}}`))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Duplicate delivery is acknowledged", func(t *testing.T) {
		t.Parallel()

		h, events, _ := newHandler(t, webhookSecret)
		events.EXPECT().ExistsByEventID(gomock.Any(), gomock.Any()).Return(true, nil)

		w := do(t, h, signedPOST(t, webhookSecret, loadFixture(t, "opened.json")))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Processing failure returns a server error", func(t *testing.T) {
		t.Parallel()

		h, events, _ := newHandler(t, webhookSecret)
		events.EXPECT().ExistsByEventID(gomock.Any(), gomock.Any()).Return(false, nil)
		events.EXPECT().Create(gomock.Any(), gomock.Any()).Return(engagement.EmailEvent{}, errors.New("db down"))

		w := do(t, h, signedPOST(t, webhookSecret, loadFixture(t, "opened.json")))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
