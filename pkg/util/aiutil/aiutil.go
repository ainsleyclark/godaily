// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package aiutil provides shared helpers for working with AI model responses.
package aiutil

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// SanitisePost removes characters that brand rules forbid but the model
// occasionally emits anyway. Currently replaces em dashes (—) with a
// plain hyphen so they never reach platform APIs.
func SanitisePost(s string) string {
	return strings.ReplaceAll(s, "—", "-")
}

// TruncatePost caps s at limit runes, trimming on a word boundary and
// appending a single-rune ellipsis when content is dropped. It returns s
// unchanged when it already fits (or when limit is non-positive).
//
// Platform APIs such as Bluesky reject posts over a grapheme-cluster limit
// (300 for Bluesky). A grapheme cluster is always one or more runes, so a
// rune-based cap of N guarantees the result is at most N graphemes — making
// this safe to enforce those limits without a Unicode segmentation library.
func TruncatePost(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= limit {
		return s
	}

	// Reserve one rune for the ellipsis we append below.
	const ellipsis = "…"
	cut := string([]rune(s)[:limit-1])

	// Back off to the last whitespace so we don't slice a word in half.
	// Skip this when there's no interior whitespace (e.g. a single long token).
	if idx := strings.LastIndexFunc(cut, unicode.IsSpace); idx > 0 {
		cut = cut[:idx]
	}

	return strings.TrimRight(cut, " \t\r\n") + ellipsis
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
