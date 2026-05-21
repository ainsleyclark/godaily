// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// Package prompts provides domain-level prompt construction, AI invocation, and
// response parsing for Go news digests and social-post suggestions.
package prompts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

type (
	// Suggestion is the structured output from Suggest.
	Suggestion struct {
		Date       time.Time `json:"date"`
		Post       string    `json:"post"`
		References []Ref     `json:"references"`
	}
	// Ref is a single item the model cited when drafting the post.
	Ref struct {
		Title  string      `json:"title"`
		URL    string      `json:"url"`
		Source news.Source `json:"source"`
	}
	// DigestMeta is the structured output returned by Synthesise.
	DigestMeta struct {
		Title string `json:"title"` // ≤80 chars — email subject / card title
		Intro string `json:"intro"` // 1–2 sentence digest intro paragraph
	}
)

// ErrNoItems is returned when there is nothing to summarise.
var ErrNoItems = errors.New("prompts: no items to summarise")

// Markdown renders a Suggestion as a human-readable markdown document
// suitable for stdout, the email digest, or copy/paste.
func (s Suggestion) Markdown() string {
	var b strings.Builder

	fmt.Fprintf(&b, "## Suggested post: %s\n\n", s.Date.Format("2006-01-02"))

	b.WriteString(s.Post)
	b.WriteString("\n\n")

	if len(s.References) > 0 {
		b.WriteString("### References\n\n")
		for _, r := range s.References {
			fmt.Fprintf(&b, "- [%s](%s) (%s)\n", r.Title, r.URL, r.Source)
		}
	}

	return b.String()
}

// JSON renders the Suggestion as indented JSON. Useful for piping into
// jq or storing alongside the daily digest output.
func (s Suggestion) JSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "\t")
}
