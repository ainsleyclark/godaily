// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package prompts provides domain-level prompt construction, AI invocation, and
// response parsing for Go news digests.
package prompts

import "errors"

type (
	// DigestMeta is the structured output returned by Synthesise.
	DigestMeta struct {
		Title string `json:"title"` // ≤80 chars — email subject / card title
		Intro string `json:"intro"` // a single short editorial paragraph (~2-4 sentences), no line breaks
	}
)

// ErrNoItems is returned when there is nothing to summarise.
var ErrNoItems = errors.New("prompts: no items to summarise")
