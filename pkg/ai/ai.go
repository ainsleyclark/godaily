// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ai provides a transport-only abstraction for AI prompt round-trips.
// Domain logic (prompt building, response parsing, Go-news types) lives in
// pkg/digest/prompts; this package only defines the Prompter interface and the
// chaining Client.
package ai

import "context"

//go:generate go run go.uber.org/mock/mockgen -package=mockai -destination=../mocks/ai/Prompter.go . Prompter

// Model identifiers callers pass to Prompt.
const (
	ModelSonnet = "claude-sonnet-4-6" // Anthropic, balanced default
	ModelOpus   = "claude-opus-4-7"   // Anthropic, highest quality

	ModelGeminiFlash = "gemini-2.0-flash" // Gemini, balanced default
	ModelGeminiPro   = "gemini-2.5-pro"   // Gemini, highest quality
)

// Prompter abstracts a single AI prompt round-trip.
// model selects the vendor model (use the Model* constants); system is the
// task directive; user is the data payload.
// Implementations must be safe for concurrent use.
type Prompter interface {
	Prompt(ctx context.Context, model, system, user string) ([]byte, error)
}
