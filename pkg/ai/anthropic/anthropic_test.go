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

package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAnthropicServer starts an httptest.Server that responds to POST
// /v1/messages with the given status and body, and captures the request.
func fakeAnthropicServer(t *testing.T, status int, body string) (*httptest.Server, *json.RawMessage) {
	t.Helper()
	var captured json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		captured = raw
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &captured
}

func validAnthropicResponse(text string) string {
	envelope := map[string]any{
		"id":            "msg_test",
		"type":          "message",
		"role":          "assistant",
		"model":         "claude-sonnet-4-6",
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"content":       []map[string]any{{"type": "text", "text": text}},
		"usage": map[string]int64{
			"input_tokens":                10,
			"output_tokens":               5,
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens":     0,
		},
	}
	out, _ := json.Marshal(envelope)
	return string(out)
}

func TestClient_Prompt(t *testing.T) {
	t.Parallel()

	t.Run("API Error Wrapped", func(t *testing.T) {
		t.Parallel()

		srv, _ := fakeAnthropicServer(t, http.StatusInternalServerError,
			`{"error":{"type":"api_error","message":"internal error"}}`)

		c := New("test", option.WithBaseURL(srv.URL), option.WithMaxRetries(0))

		_, err := c.Prompt(context.Background(), "system text", "user payload")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "anthropic")
	})

	t.Run("OK Returns Text Bytes", func(t *testing.T) {
		t.Parallel()

		srv, _ := fakeAnthropicServer(t, http.StatusOK, validAnthropicResponse(`{"post":"hello"}`))

		c := New("test", option.WithBaseURL(srv.URL))

		got, err := c.Prompt(context.Background(), "system text", "user payload")
		require.NoError(t, err)
		assert.Equal(t, `{"post":"hello"}`, string(got))
	})

	t.Run("System String Sent As Single Block", func(t *testing.T) {
		t.Parallel()

		srv, captured := fakeAnthropicServer(t, http.StatusOK, validAnthropicResponse("ok"))

		c := New("test", option.WithBaseURL(srv.URL))

		_, err := c.Prompt(context.Background(), "You are a helpful assistant.", "user data")
		require.NoError(t, err)

		var req struct {
			System []struct {
				Text         string `json:"text"`
				CacheControl *struct {
					Type string `json:"type"`
				} `json:"cache_control,omitempty"`
			} `json:"system"`
			Messages []struct {
				Role    string `json:"role"`
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"messages"`
		}
		require.NoError(t, json.Unmarshal(*captured, &req))

		require.Len(t, req.System, 1)
		assert.Equal(t, "You are a helpful assistant.", req.System[0].Text)
		assert.Nil(t, req.System[0].CacheControl)

		require.Len(t, req.Messages, 1)
		assert.Equal(t, "user", req.Messages[0].Role)
		require.Len(t, req.Messages[0].Content, 1)
		assert.Equal(t, "user data", req.Messages[0].Content[0].Text)
	})
}
