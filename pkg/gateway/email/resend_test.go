// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package email

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	c := New("test-token")

	require.NotNil(t, c)
	require.NotNil(t, c.resend)
	assert.Equal(t, "test-token", c.resend.ApiKey)
}

func TestClient_Send(t *testing.T) {
	t.Parallel()

	newClient := func(t *testing.T, stub http.HandlerFunc) *Client {
		t.Helper()
		srv := httptest.NewServer(stub)
		t.Cleanup(srv.Close)

		c := New("test-token")
		base, err := url.Parse(srv.URL + "/")
		require.NoError(t, err)
		c.resend.BaseURL = base
		return c
	}

	req := SendEmailRequest{
		From:    "from@example.com",
		To:      []string{"to@example.com"},
		Subject: "hello",
		Text:    "world",
	}

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		c := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"abc-123"}`))
		})

		err := c.Send(t.Context(), req)
		assert.NoError(t, err)
	})

	t.Run("API Error", func(t *testing.T) {
		t.Parallel()

		c := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"invalid sender"}`))
		})

		err := c.Send(t.Context(), req)
		assert.ErrorContains(t, err, "invalid sender")
	})
}
