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

// ExtractJSON returns the first complete, balanced JSON value (object or
// array) found in s, discarding any surrounding prose the model may emit
// despite being told to output JSON alone. It first strips a wrapping markdown
// fence, then scans from the first '{' or '[' to its matching close, respecting
// string literals and escapes so braces inside strings are ignored.
//
// This guards against models that append commentary after the value, e.g.
// `{"a":1}\n\nWait, let me reconsider...`, which would otherwise fail
// json.Unmarshal with "invalid character after top-level value".
//
// It returns "" when no opening bracket is present. When the value is
// unbalanced it returns everything from the first bracket onward so the
// caller's unmarshal surfaces a meaningful error. The result is not validated
// as JSON; callers unmarshal it as usual.
func ExtractJSON(s string) string {
	s = StripFences(s)

	start := strings.IndexAny(s, "{[")
	if start < 0 {
		return ""
	}

	openByte := s[start]
	closeByte := byte('}')
	if openByte == '[' {
		closeByte = ']'
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case openByte:
			depth++
		case closeByte:
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}

	// Unbalanced: hand back from the first bracket so unmarshal reports why.
	return s[start:]
}
