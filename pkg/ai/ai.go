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

// Model identifiers passed to Prompt. They are real vendor model IDs, exposed
// here so callers can name an actual model without importing a vendor SDK. The
// primary (Anthropic) provider runs them verbatim; the Client maps them onto
// the fallback's model line when fanning a call out (see geminiModelFor).
const (
	ModelSonnet = "claude-sonnet-4-6" // balanced default
	ModelOpus   = "claude-opus-4-7"   // highest quality, for the edition intro
)

// Prompter abstracts a single AI prompt round-trip.
// model selects the vendor model (use the Model* constants); system is the
// task directive; user is the data payload.
// Implementations must be safe for concurrent use.
type Prompter interface {
	Prompt(ctx context.Context, model, system, user string) ([]byte, error)
}
