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

// Package gemini provides an ai.Prompter implementation backed by Google's
// Gemini REST API via net/http.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/ainsleyclark/godaily/pkg/util/gohttp"
	"github.com/pkg/errors"
)

const (
	geminiModel      = "gemini-2.0-flash"
	geminiDefaultURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"
)

// Client satisfies ai.Prompter using Google's Gemini REST API via net/http.
// system and user are merged into a single user-role message, as the
// free-tier endpoint does not support a distinct system role.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// New constructs a Client with the given API key.
func New(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: geminiDefaultURL,
		http:    gohttp.New(gohttp.WithRetryMethods(http.MethodPost)),
	}
}

// Prompt sends a merged system+user prompt to the Gemini API and returns the
// first candidate's text content bytes.
func (c *Client) Prompt(ctx context.Context, system, user string) ([]byte, error) {
	merged := system + "\n\n" + user
	reqBody, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]any{{"text": merged}}},
		},
		"generationConfig": map[string]any{
			"temperature":     0.4,
			"maxOutputTokens": 1024,
		},
	})
	url := c.baseURL + "?key=" + c.apiKey
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, errors.Wrap(err, "gemini: building request")
	}
	req.Header.Set("Content-Type", "application/json")

	slog.InfoContext(ctx, "Calling Gemini fallback", "model", geminiModel)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "gemini: request")
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, errors.Wrap(err, "gemini: parsing response")
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("gemini: empty candidates in response")
	}

	slog.InfoContext(ctx, "Gemini fallback response received")
	return []byte(parsed.Candidates[0].Content.Parts[0].Text), nil
}
