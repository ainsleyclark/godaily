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

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/env"
	mockaudience "github.com/ainsleyclark/godaily/pkg/mocks/audience"
	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/engagement"
	engagementsvc "github.com/ainsleyclark/godaily/pkg/services/engagement"
)

var webhookSecret = "whsec_" + base64.StdEncoding.EncodeToString([]byte("godaily-handler-test-secret-key!"))

// noopItemFinder satisfies engagement.ItemFinder, resolving no items.
type noopItemFinder struct{}

func (noopItemFinder) FindByURLInIssue(context.Context, int64, string) (int64, bool, error) {
	return 0, false, nil
}

func loadFixture(t *testing.T, name string) string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("testdata", name))
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

func TestHandleResend(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler     *Handler
		Context     *webkit.Context
		Recorder    *httptest.ResponseRecorder
		EmailEvents *mockengagement.MockEmailEventRepository
		Subscriber  *mockaudience.MockSubscriberService
	}

	setup := func(t *testing.T, secret string, req *http.Request) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		events := mockengagement.NewMockEmailEventRepository(ctrl)
		subs := mockaudience.NewMockSubscriberService(ctrl)
		rec := httptest.NewRecorder()

		return Test{
			Handler: &Handler{
				emailEvents: engagementsvc.NewEvents(events, subs, noopItemFinder{}, ""),
				config:      &env.Config{ResendWebhookSecret: secret},
			},
			Recorder:    rec,
			Context:     webkit.NewContext(rec, req),
			EmailEvents: events,
			Subscriber:  subs,
		}
	}

	t.Run("Valid signed event is processed", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, webhookSecret, signedPOST(t, webhookSecret, loadFixture(t, "delivered.json")))
		deps.EmailEvents.EXPECT().ExistsByEventID(gomock.Any(), gomock.Any()).Return(false, nil)
		deps.EmailEvents.EXPECT().Create(gomock.Any(), gomock.Any()).Return(engagement.EmailEvent{}, nil)
		err := deps.Handler.Resend(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Bounced event marks the subscriber", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, webhookSecret, signedPOST(t, webhookSecret, loadFixture(t, "bounced.json")))
		deps.EmailEvents.EXPECT().ExistsByEventID(gomock.Any(), gomock.Any()).Return(false, nil)
		deps.EmailEvents.EXPECT().Create(gomock.Any(), gomock.Any()).Return(engagement.EmailEvent{}, nil)
		deps.Subscriber.EXPECT().MarkBounced(gomock.Any(), "dead-inbox@example.com").Return(nil)

		err := deps.Handler.Resend(deps.Context)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Invalid signature is rejected", func(t *testing.T) {
		t.Parallel()

		req := signedPOST(t, webhookSecret, "{}")
		req.Header.Set("svix-signature", "v1,not-a-real-signature")
		deps := setup(t, webhookSecret, req)

		_ = deps.Handler.Resend(deps.Context)
		assert.Equal(t, http.StatusUnauthorized, deps.Recorder.Code)
	})

	t.Run("Malformed body is rejected", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, webhookSecret, signedPOST(t, webhookSecret, "{not json"))

		_ = deps.Handler.Resend(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Missing secret returns a bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "", signedPOST(t, webhookSecret, "{}"))

		_ = deps.Handler.Resend(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Untracked event type is acknowledged", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, webhookSecret, signedPOST(t, webhookSecret, `{"type":"email.sent","data":{}}`))
		err := deps.Handler.Resend(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Duplicate delivery is acknowledged", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, webhookSecret, signedPOST(t, webhookSecret, loadFixture(t, "opened.json")))
		deps.EmailEvents.EXPECT().ExistsByEventID(gomock.Any(), gomock.Any()).Return(true, nil)

		err := deps.Handler.Resend(deps.Context)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Processing failure returns a server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, webhookSecret, signedPOST(t, webhookSecret, loadFixture(t, "opened.json")))
		deps.EmailEvents.EXPECT().ExistsByEventID(gomock.Any(), gomock.Any()).Return(false, nil)
		deps.EmailEvents.EXPECT().Create(gomock.Any(), gomock.Any()).Return(engagement.EmailEvent{}, errors.New("db down"))

		_ = deps.Handler.Resend(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
