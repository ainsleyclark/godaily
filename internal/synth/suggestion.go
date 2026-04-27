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

package synth

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/ainsleyclark/godaily/internal/news"
)

const maxTwitterChars = 280

// ErrNoItems is returned by Suggest when there is nothing to summarise.
// Callers can treat this as a soft skip rather than a hard error.
var ErrNoItems = errors.New("synth: no items to suggest from")

type (
	// Suggestion is the structured output returned by Suggest.
	Suggestion struct {
		Date       time.Time `json:"date"`
		Twitter    string    `json:"twitter"`
		LinkedIn   string    `json:"linkedin"`
		References []Ref     `json:"references"`
	}
	// Ref is a single item the model cited when drafting the posts. Source
	// is the news.Source string ("hacker_news", "go_blog", ...).
	Ref struct {
		Title  string      `json:"title"`
		URL    string      `json:"url"`
		Source news.Source `json:"source"`
	}
)

// Markdown renders a Suggestion as a human-readable markdown document
// suitable for stdout, the email digest, or copy/paste.
func (s Suggestion) Markdown() string {
	var b strings.Builder

	fmt.Fprintf(&b, "## Suggested posts — %s\n\n", s.Date.Format("2006-01-02"))

	b.WriteString("### Twitter\n\n")
	b.WriteString(s.Twitter)
	b.WriteString("\n\n")

	b.WriteString("### LinkedIn\n\n")
	b.WriteString(s.LinkedIn)
	b.WriteString("\n\n")

	if len(s.References) > 0 {
		b.WriteString("### References\n\n")
		for _, r := range s.References {
			fmt.Fprintf(&b, "- [%s](%s) — %s\n", r.Title, r.URL, r.Source)
		}
	}

	return b.String()
}

// JSON renders the Suggestion as indented JSON. Useful for piping into
// jq or storing alongside the daily digest output.
func (s Suggestion) JSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "\t")
}

// stripFences defensively removes a wrapping ```json ... ``` (or plain
// ``` ... ```) block if the model emits one despite being told not to.
// Anything outside the fence is discarded.
func stripFences(s string) string {
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

// parseResponse extracts the suggestion JSON from the model's text
// blocks, validates length and required fields, and returns a populated
// Suggestion (without Date — that is filled in by the caller).
func parseResponse(m *anthropic.Message) (Suggestion, error) {
	if m == nil {
		return Suggestion{}, errors.New("nil message")
	}

	var raw strings.Builder
	for _, b := range m.Content {
		if b.Type == "text" {
			raw.WriteString(b.Text)
		}
	}
	body := stripFences(raw.String())
	if body == "" {
		return Suggestion{}, errors.New("empty response body")
	}

	var out struct {
		Twitter    string `json:"twitter"`
		LinkedIn   string `json:"linkedin"`
		References []Ref  `json:"references"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return Suggestion{}, fmt.Errorf("parse: %w (raw=%q)", err, body)
	}

	if out.Twitter == "" || out.LinkedIn == "" {
		return Suggestion{}, errors.New("missing twitter or linkedin field")
	}
	if n := utf8.RuneCountInString(out.Twitter); n > maxTwitterChars {
		return Suggestion{}, fmt.Errorf("twitter post is %d chars, max %d", n, maxTwitterChars)
	}

	return Suggestion{
		Twitter:    out.Twitter,
		LinkedIn:   out.LinkedIn,
		References: out.References,
	}, nil
}
