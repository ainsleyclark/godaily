// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ai provides a transport-only abstraction for AI prompt round-trips.
// Domain logic (prompt building, response parsing, Go-news types) lives in
// pkg/digest/prompts; this package only defines the Prompter/Provider
// interfaces and the chaining Client.
package ai

import "context"

//go:generate go run go.uber.org/mock/mockgen -package=mockai -destination=../mocks/ai/Prompter.go . Prompter,Provider

// Model identifiers passed to PromptWithModel. They are the real vendor model
// IDs, exposed here so callers can name an actual model without importing a
// vendor SDK. The Anthropic provider uses them verbatim; the Gemini fallback
// maps them onto its own model line.
const (
	ModelSonnet = "claude-sonnet-4-6" // balanced default
	ModelOpus   = "claude-opus-4-7"   // highest quality, for the edition intro
)

// Prompter is the caller-facing abstraction services depend on.
// system is the task directive; user is the data payload.
// Prompt uses the default model; PromptWithModel opts into a specific one.
// Implementations must be safe for concurrent use.
type Prompter interface {
	Prompt(ctx context.Context, system, user string) ([]byte, error)
	PromptWithModel(ctx context.Context, model, system, user string) ([]byte, error)
}

// Provider is the low-level, model-aware round-trip implemented by each vendor
// (Anthropic, Gemini). It is internal to the ai layer: the chaining Client
// composes Providers and satisfies the caller-facing Prompter.
type Provider interface {
	Prompt(ctx context.Context, model, system, user string) ([]byte, error)
}
