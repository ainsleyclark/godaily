// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gemini provides an ai.Prompter implementation backed by Google's
// Gemini API via the official go-genai SDK.
package gemini

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"
	"google.golang.org/genai"
)

// contentGenerator abstracts genai.Models to allow test doubles.
type contentGenerator interface {
	GenerateContent(ctx context.Context, model string, contents []*genai.Content, cfg *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
}

// Client satisfies ai.Prompter using Google's Gemini API via the go-genai SDK.
type Client struct {
	gen contentGenerator
}

// New constructs a Client with the given API key.
func New(apiKey string) (*Client, error) {
	c, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, errors.Wrap(err, "gemini: creating client")
	}
	return &Client{gen: c.Models}, nil
}

// Prompt sends system and user prompts to Gemini and returns the first
// candidate's text content bytes. model is a Gemini model ID; the ai.Client
// maps the requested model onto Gemini's line before calling here.
func (c *Client) Prompt(ctx context.Context, model, system, user string) ([]byte, error) {
	slog.InfoContext(
		ctx, "Calling Gemini fallback",
		"model", model,
		"system_len", len(system),
		"user_len", len(user),
	)

	resp, err := c.gen.GenerateContent(
		ctx, model,
		[]*genai.Content{
			{Role: "user", Parts: []*genai.Part{genai.NewPartFromText(user)}},
		},
		&genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{genai.NewPartFromText(system)},
			},
			Temperature:     genai.Ptr(float32(0.4)),
			MaxOutputTokens: 1024,
		},
	)
	if err != nil {
		slog.ErrorContext(
			ctx, "Gemini API request failed",
			"model", model,
			"err", err,
		)
		return nil, errors.Wrap(err, "gemini: generate content")
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		var finishReason string
		if len(resp.Candidates) > 0 {
			finishReason = string(resp.Candidates[0].FinishReason)
		}
		slog.WarnContext(
			ctx, "Gemini returned empty candidates",
			"model", model,
			"finish_reason", finishReason,
		)
		return nil, errors.New("gemini: empty candidates in response")
	}

	text := resp.Text()
	slog.InfoContext(
		ctx, "Gemini fallback response received",
		"model", model,
		"response_len", len(text),
	)
	return []byte(text), nil
}
