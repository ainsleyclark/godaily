// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package aiutil provides shared helpers for working with AI model responses.
package aiutil

import "strings"

// SanitisePost removes characters that brand rules forbid but the model
// occasionally emits anyway. Currently replaces em dashes (—) with a
// plain hyphen so they never reach platform APIs.
func SanitisePost(s string) string {
	return strings.ReplaceAll(s, "—", "-")
}

// StripFences defensively removes a wrapping ```json ... ``` (or plain
// ``` ... ```) block if the model emits one despite being told not to.
// Anything outside the fence is discarded.
func StripFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[i+1:]
	} else {
		return s
	}
	if j := strings.LastIndex(s, "```"); j >= 0 {
		s = s[:j]
	}
	return strings.TrimSpace(s)
}
