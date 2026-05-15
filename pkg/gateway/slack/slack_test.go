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
