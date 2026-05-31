// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ai provides a transport-only abstraction for AI prompt round-trips.
// Domain logic (prompt building, response parsing, Go-news types) lives in
// pkg/digest/prompts; this package only defines the Prompter interface and the
// chaining Client.
package ai

import (
	"context"
	"strings"
)

//go:generate go run go.uber.org/mock/mockgen -package=mockai -destination=../mocks/ai/Prompter.go . Prompter

// Model identifiers. The Model* values are caller-facing: passed to Prompt to
// name an actual model without importing a vendor SDK, and run verbatim by the
// primary (Anthropic) provider. The gemini* values are internal routing
// targets the Client maps onto when fanning a call out to the Gemini fallback;
// callers never pass them (an unmapped Gemini ID is invalid for the primary).
const (
	ModelSonnet = "claude-sonnet-4-6" // balanced default
	ModelOpus   = "claude-opus-4-7"   // highest quality, for the edition intro

	geminiFlash = "gemini-2.0-flash" // fallback balanced default
	geminiPro   = "gemini-2.5-pro"   // fallback premium, mirrors an Opus-class request
)

// geminiModelFor maps the requested (primary/Anthropic) model onto the Gemini
// model the fallback should run. Premium (Opus-class) requests map to Pro,
// everything else to Flash.
func geminiModelFor(model string) string {
	if strings.HasPrefix(model, "claude-opus") {
		return geminiPro
	}
	return geminiFlash
}

// Prompter abstracts a single AI prompt round-trip.
// model selects the vendor model (use the Model* constants); system is the
// task directive; user is the data payload.
// Implementations must be safe for concurrent use.
type Prompter interface {
	Prompt(ctx context.Context, model, system, user string) ([]byte, error)
}
