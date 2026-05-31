// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gemini

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

// fakeGenerator is a test double for contentGenerator.
type fakeGenerator struct {
	resp *genai.GenerateContentResponse
	err  error
}

func (f *fakeGenerator) GenerateContent(_ context.Context, _ string, _ []*genai.Content, _ *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	return f.resp, f.err
}

func textResponse(text string) *genai.GenerateContentResponse {
	return &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{genai.NewPartFromText(text)},
				},
			},
		},
	}
}

func TestClient_Prompt(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		c := &Client{gen: &fakeGenerator{resp: textResponse(`{"post":"hello"}`)}}

		got, err := c.Prompt(context.Background(), "", "system prompt", "user payload")
		require.NoError(t, err)
		assert.Equal(t, `{"post":"hello"}`, string(got))
	})

	t.Run("API Error Returns Wrapped Error", func(t *testing.T) {
		t.Parallel()

		c := &Client{gen: &fakeGenerator{err: errors.New("HTTP 400: invalid request")}}

		_, err := c.Prompt(context.Background(), "", "sys", "user")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "gemini: generate content")
		assert.Contains(t, err.Error(), "HTTP 400")
	})

	t.Run("Empty Candidates Returns Error", func(t *testing.T) {
		t.Parallel()

		c := &Client{gen: &fakeGenerator{resp: &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{},
		}}}

		_, err := c.Prompt(context.Background(), "", "sys", "user")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty candidates")
	})

	t.Run("System Instruction Passed Separately", func(t *testing.T) {
		t.Parallel()

		var capturedCfg *genai.GenerateContentConfig
		var capturedContents []*genai.Content

		gen := &captureGenerator{
			resp: textResponse("ok"),
			onCall: func(contents []*genai.Content, cfg *genai.GenerateContentConfig) {
				capturedContents = contents
				capturedCfg = cfg
			},
		}

		c := &Client{gen: gen}
		_, err := c.Prompt(context.Background(), "", "system text", "user text")
		require.NoError(t, err)

		require.Len(t, capturedContents, 1)
		assert.Equal(t, "user", capturedContents[0].Role)
		require.Len(t, capturedContents[0].Parts, 1)
		assert.Equal(t, "user text", capturedContents[0].Parts[0].Text)

		require.NotNil(t, capturedCfg.SystemInstruction)
		require.Len(t, capturedCfg.SystemInstruction.Parts, 1)
		assert.Equal(t, "system text", capturedCfg.SystemInstruction.Parts[0].Text)
	})

	t.Run("Maps Requested Model To Gemini Line", func(t *testing.T) {
		t.Parallel()

		tests := map[string]struct {
			requested string
			want      string
		}{
			"empty defaults to flash": {requested: "", want: geminiFlash},
			"sonnet maps to flash":    {requested: "claude-sonnet-4-6", want: geminiFlash},
			"opus maps to pro":        {requested: "claude-opus-4-7", want: geminiPro},
		}

		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				var capturedModel string
				gen := &captureGenerator{
					resp:    textResponse("ok"),
					onModel: func(model string) { capturedModel = model },
				}

				c := &Client{gen: gen}
				_, err := c.Prompt(context.Background(), tc.requested, "sys", "user")
				require.NoError(t, err)
				assert.Equal(t, tc.want, capturedModel)
			})
		}
	})
}

type captureGenerator struct {
	resp    *genai.GenerateContentResponse
	onCall  func(contents []*genai.Content, cfg *genai.GenerateContentConfig)
	onModel func(model string)
}

func (g *captureGenerator) GenerateContent(_ context.Context, model string, contents []*genai.Content, cfg *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	if g.onCall != nil {
		g.onCall(contents, cfg)
	}
	if g.onModel != nil {
		g.onModel(model)
	}
	return g.resp, nil
}
