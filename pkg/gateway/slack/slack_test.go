// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slack

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()
	got := New("token", "#godaily")
	assert.NotNil(t, got.slackSendFunc)
}

func TestClient_Send(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		s := Client{
			slackSendFunc: func(_ context.Context, _ string, _ ...slack.MsgOption) (string, string, error) {
				return "", "", nil
			},
		}

		got := s.Send(t.Context(), "message")
		assert.NoError(t, got)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		s := Client{
			slackSendFunc: func(_ context.Context, _ string, _ ...slack.MsgOption) (string, string, error) {
				return "id", "timestamp", errors.New("error")
			},
		}

		got := s.Send(t.Context(), "message")
		want := "failed to send message to Slack channel 'id' at time 'timestamp': error"
		assert.ErrorContains(t, got, want)
	})
}

func TestClient_MustSend(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		s := Client{
			slackSendFunc: func(_ context.Context, _ string, _ ...slack.MsgOption) (string, string, error) {
				return "", "", nil
			},
		}

		var buf bytes.Buffer
		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

		s.MustSend(t.Context(), "message")
		assert.Equal(t, "", buf.String())
	})

	t.Run("Error", func(t *testing.T) {
		s := Client{
			slackSendFunc: func(_ context.Context, _ string, _ ...slack.MsgOption) (string, string, error) {
				return "id", "timestamp", errors.New("error")
			},
		}

		var buf bytes.Buffer
		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

		s.MustSend(t.Context(), "message")
		assert.Contains(t, buf.String(), "Slack error")
	})
}
