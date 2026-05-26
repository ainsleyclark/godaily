// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gemini

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeGeminiServer(t *testing.T, status int, body string) (*httptest.Server, *[]byte) {
	t.Helper()
	var captured []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		captured = raw
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &captured
}

func validGeminiResponse(text string) string {
	resp := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{
						{"text": text},
					},
				},
			},
		},
	}
	out, _ := json.Marshal(resp)
	return string(out)
}

func TestClient_Prompt(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		srv, _ := fakeGeminiServer(t, http.StatusOK, validGeminiResponse(`{"post":"hello"}`))

		c := New("test-key")
		c.baseURL = srv.URL

		got, err := c.Prompt(context.Background(), "system prompt", "user payload")
		require.NoError(t, err)
		assert.Equal(t, `{"post":"hello"}`, string(got))
	})

	t.Run("HTTP Non-200 Returns Error", func(t *testing.T) {
		t.Parallel()

		srv, _ := fakeGeminiServer(t, http.StatusUnauthorized, `{"error":"invalid key"}`)

		c := New("bad-key")
		c.baseURL = srv.URL

		_, err := c.Prompt(context.Background(), "sys", "user")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("Malformed JSON Returns Error", func(t *testing.T) {
		t.Parallel()

		srv, _ := fakeGeminiServer(t, http.StatusOK, "not json")

		c := New("test-key")
		c.baseURL = srv.URL

		_, err := c.Prompt(context.Background(), "sys", "user")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "gemini: parsing response")
	})

	t.Run("Empty Candidates Returns Error", func(t *testing.T) {
		t.Parallel()

		body, _ := json.Marshal(map[string]any{"candidates": []any{}})
		srv, _ := fakeGeminiServer(t, http.StatusOK, string(body))

		c := New("test-key")
		c.baseURL = srv.URL

		_, err := c.Prompt(context.Background(), "sys", "user")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty candidates")
	})

	t.Run("System And User Merged In Request Body", func(t *testing.T) {
		t.Parallel()

		srv, captured := fakeGeminiServer(t, http.StatusOK, validGeminiResponse("ok"))

		c := New("test-key")
		c.baseURL = srv.URL

		_, err := c.Prompt(context.Background(), "system text", "user text")
		require.NoError(t, err)

		var req struct {
			Contents []struct {
				Role  string `json:"role"`
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"contents"`
		}
		require.NoError(t, json.Unmarshal(*captured, &req))

		require.Len(t, req.Contents, 1)
		assert.Equal(t, "user", req.Contents[0].Role)
		require.Len(t, req.Contents[0].Parts, 1)
		merged := req.Contents[0].Parts[0].Text
		assert.True(t, strings.Contains(merged, "system text"), "system text must be in merged body")
		assert.True(t, strings.Contains(merged, "user text"), "user text must be in merged body")
	})
}
