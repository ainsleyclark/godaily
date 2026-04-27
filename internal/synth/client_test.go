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

package synth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/internal/news"
)

// captured holds the parsed body of the most recent /v1/messages call,
// so tests can assert on what we sent.
type captured struct {
	System []struct {
		Text         string `json:"text"`
		CacheControl *struct {
			Type string `json:"type"`
		} `json:"cache_control,omitempty"`
	} `json:"system"`
	Messages []struct {
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"messages"`
	Model string `json:"model"`
}

// fakeServer responds to POST /v1/messages with `body` (status code
// `status`) and records the request body for inspection.
func fakeServer(t *testing.T, status int, body string) (*httptest.Server, *captured) {
	t.Helper()
	got := &captured{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, got)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

func sampleSections() []news.SourceItems {
	return []news.SourceItems{{
		Source: news.SourceGoBlog,
		Items: []news.Item{{
			Source: news.SourceGoBlog,
			Title:  "Go 1.24 ships",
			URL:    "https://go.dev/blog/go1.24",
			Score:  0.95,
		}},
	}}
}

// validResponse mimics the JSON shape Anthropic returns for a Messages
// call, with a single text block that contains the strict-JSON payload
// our parser expects.
func validResponse(twitter, linkedin string) string {
	inner, _ := json.Marshal(map[string]any{
		"twitter":  twitter,
		"linkedin": linkedin,
		"references": []map[string]string{{
			"title":  "Go 1.24 ships",
			"url":    "https://go.dev/blog/go1.24",
			"source": "go_blog",
		}},
	})
	envelope := map[string]any{
		"id":            "msg_test",
		"type":          "message",
		"role":          "assistant",
		"model":         "claude-sonnet-4-6",
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"content":       []map[string]any{{"type": "text", "text": string(inner)}},
		"usage": map[string]int64{
			"input_tokens":                123,
			"output_tokens":               45,
			"cache_creation_input_tokens": 600,
			"cache_read_input_tokens":     0,
		},
	}
	out, _ := json.Marshal(envelope)
	return string(out)
}

func TestNew(t *testing.T) {
	t.Parallel()
	c := New()
	assert.Equal(t, defaultFilterConfig(), c.filter)
}

func TestClient_Suggest(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, time.April, 27, 0, 0, 0, 0, time.UTC)

	t.Run("No Items Returns ErrNoItems Without HTTP Call", func(t *testing.T) {
		t.Parallel()
		called := 0
		srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called++ }))
		t.Cleanup(srv.Close)

		c := New(option.WithBaseURL(srv.URL), option.WithAPIKey("test"))
		got, err := c.Suggest(context.Background(), day, nil)

		require.ErrorIs(t, err, ErrNoItems)
		assert.Empty(t, got.Twitter)
		assert.Equal(t, 0, called, "must not call API when there are no items")
	})

	t.Run("API Error Wrapped", func(t *testing.T) {
		t.Parallel()
		srv, _ := fakeServer(t, http.StatusInternalServerError, `{"error":{"type":"api_error","message":"boom"}}`)

		c := New(
			option.WithBaseURL(srv.URL),
			option.WithAPIKey("test"),
			option.WithMaxRetries(0),
		)
		_, err := c.Suggest(context.Background(), day, sampleSections())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "anthropic:")
	})

	t.Run("Parse Error Surfaced", func(t *testing.T) {
		t.Parallel()
		envelope, _ := json.Marshal(map[string]any{
			"id": "msg", "type": "message", "role": "assistant",
			"model":   "claude-sonnet-4-6",
			"content": []map[string]any{{"type": "text", "text": "garbage"}},
			"usage":   map[string]int64{"input_tokens": 1, "output_tokens": 1},
		})
		srv, _ := fakeServer(t, http.StatusOK, string(envelope))

		c := New(option.WithBaseURL(srv.URL), option.WithAPIKey("test"))
		_, err := c.Suggest(context.Background(), day, sampleSections())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse:")
	})

	t.Run("OK Populates Date And Sends Cached Prompt", func(t *testing.T) {
		t.Parallel()
		srv, got := fakeServer(t, http.StatusOK, validResponse("hi", "hi\nworld"))

		c := New(option.WithBaseURL(srv.URL), option.WithAPIKey("test"))
		sug, err := c.Suggest(context.Background(), day, sampleSections())
		require.NoError(t, err)

		assert.Equal(t, "hi", sug.Twitter)
		assert.Equal(t, "hi\nworld", sug.LinkedIn)
		assert.Equal(t, day, sug.Date)
		require.Len(t, sug.References, 1)
		assert.Equal(t, news.SourceGoBlog, sug.References[0].Source)

		// Verify the request the SDK actually sent.
		assert.Equal(t, "claude-sonnet-4-6", got.Model)
		require.Len(t, got.System, 2)
		assert.Nil(t, got.System[0].CacheControl, "intro block must not be cache breakpoint")
		require.NotNil(t, got.System[1].CacheControl)
		assert.Equal(t, "ephemeral", got.System[1].CacheControl.Type)

		require.Len(t, got.Messages, 1)
		assert.Equal(t, "user", got.Messages[0].Role)
	})

	t.Run("Filter Caps Items In User Prompt", func(t *testing.T) {
		t.Parallel()
		srv, got := fakeServer(t, http.StatusOK, validResponse("hi", "hi"))

		// 50 items in one source — filter must drop to 3 in the user prompt.
		si := news.SourceItems{Source: news.SourceReddit}
		for i := 0; i < 50; i++ {
			si.Items = append(si.Items, news.Item{
				Source: news.SourceReddit,
				Title:  "t",
				URL:    "https://u",
				Score:  float64(i) / 100,
			})
		}

		c := New(option.WithBaseURL(srv.URL), option.WithAPIKey("test"))
		_, err := c.Suggest(context.Background(), day, []news.SourceItems{si})
		require.NoError(t, err)

		require.Len(t, got.Messages, 1)
		require.Len(t, got.Messages[0].Content, 1)
		body := got.Messages[0].Content[0].Text
		assert.Equal(t, 3, strings.Count(body, `"source":"reddit"`))
	})
}
