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

package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validDigestResponse builds a fake Anthropic envelope with a DigestMeta payload.
func validDigestResponse(title, intro string) string {
	inner, _ := json.Marshal(DigestMeta{Title: title, Intro: intro})
	envelope := map[string]any{
		"id":            "msg_digest",
		"type":          "message",
		"role":          "assistant",
		"model":         "claude-sonnet-4-6",
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"content":       []map[string]any{{"type": "text", "text": string(inner)}},
		"usage": map[string]int64{
			"input_tokens":                100,
			"output_tokens":               30,
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens":     600,
		},
	}
	out, _ := json.Marshal(envelope)
	return string(out)
}

func TestParseDigestResponse(t *testing.T) {
	t.Parallel()

	validJSON := `{"title":"Go 1.24 lands","intro":"Goroutines got faster."}`

	tt := map[string]struct {
		msg     *anthropic.Message
		wantErr string
		check   func(t *testing.T, m DigestMeta)
	}{
		"Nil Message": {
			msg:     nil,
			wantErr: "nil message",
		},
		"Empty Body": {
			msg:     &anthropic.Message{},
			wantErr: "empty response body",
		},
		"Invalid JSON": {
			msg:     makeTextMessage("not json"),
			wantErr: "parse (raw=",
		},
		"Missing Title": {
			msg:     makeTextMessage(`{"title":"","intro":"something"}`),
			wantErr: "missing title field",
		},
		"Missing Intro": {
			msg:     makeTextMessage(`{"title":"Go 1.24 lands","intro":""}`),
			wantErr: "missing intro field",
		},
		"Title Too Long Warns But Returns": {
			msg: makeTextMessage(`{"title":"` + strings.Repeat("a", 81) + `","intro":"x"}`),
			check: func(t *testing.T, m DigestMeta) {
				t.Helper()
				assert.Equal(t, 81, utf8.RuneCountInString(m.Title))
			},
		},
		"Valid": {
			msg: makeTextMessage(validJSON),
			check: func(t *testing.T, m DigestMeta) {
				t.Helper()
				assert.Equal(t, "Go 1.24 lands", m.Title)
				assert.Equal(t, "Goroutines got faster.", m.Intro)
			},
		},
		"Valid With Fenced JSON": {
			msg: makeTextMessage("```json\n" + validJSON + "\n```"),
			check: func(t *testing.T, m DigestMeta) {
				t.Helper()
				assert.Equal(t, "Go 1.24 lands", m.Title)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := parseDigestResponse(test.msg)
			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				return
			}
			require.NoError(t, err)
			test.check(t, got)
		})
	}
}

func TestClient_Synthesise(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, time.April, 27, 0, 0, 0, 0, time.UTC)

	t.Run("No Items Returns ErrNoItems Without HTTP Call", func(t *testing.T) {
		t.Parallel()
		called := 0
		srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called++ }))
		t.Cleanup(srv.Close)

		c := New("test", option.WithBaseURL(srv.URL))
		got, err := c.Synthesise(context.Background(), day, nil)

		require.ErrorIs(t, err, ErrNoItems)
		assert.Empty(t, got.Title)
		assert.Equal(t, 0, called, "must not call API when there are no items")
	})

	t.Run("API Error Wrapped", func(t *testing.T) {
		t.Parallel()
		srv, _ := fakeServer(t, http.StatusInternalServerError, `{"error":{"type":"api_error","message":"boom"}}`)

		c := New("test", option.WithBaseURL(srv.URL), option.WithMaxRetries(0))
		_, err := c.Synthesise(context.Background(), day, sampleSections())
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

		c := New("test", option.WithBaseURL(srv.URL))
		_, err := c.Synthesise(context.Background(), day, sampleSections())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse (raw=")
	})

	t.Run("OK Returns Title And Intro With Cached Prompt", func(t *testing.T) {
		t.Parallel()
		srv, got := fakeServer(t, http.StatusOK, validDigestResponse("Go 1.24 lands", "Goroutines got faster."))

		c := New("test", option.WithBaseURL(srv.URL))
		meta, err := c.Synthesise(context.Background(), day, sampleSections())
		require.NoError(t, err)

		assert.Equal(t, "Go 1.24 lands", meta.Title)
		assert.Equal(t, "Goroutines got faster.", meta.Intro)

		assert.Equal(t, "claude-sonnet-4-6", got.Model)
		require.Len(t, got.System, 2)
		assert.Nil(t, got.System[0].CacheControl, "intro block must not be cache breakpoint")
		require.NotNil(t, got.System[1].CacheControl)
		assert.Equal(t, "ephemeral", got.System[1].CacheControl.Type)
	})
}
