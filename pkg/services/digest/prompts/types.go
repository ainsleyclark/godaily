// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
	// Suggestion is the structured output from Suggest: a small set of
	// candidate posts, each about a different story, for the owner to
	// choose between.
	Suggestion struct {
		Date  time.Time `json:"date"`
		Posts []Post    `json:"posts"`
	}
	// Post is a single drafted social-media post about one story, along
	// with the item(s) it is based on.
	Post struct {
		Text       string `json:"post"`
		References []Ref  `json:"references"`
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

	fmt.Fprintf(&b, "## Suggested posts: %s\n\n", s.Date.Format("2006-01-02"))

	for i, p := range s.Posts {
		fmt.Fprintf(&b, "### Post %d\n\n", i+1)
		b.WriteString(p.Text)
		b.WriteString("\n\n")

		if len(p.References) > 0 {
			b.WriteString("#### References\n\n")
			for _, r := range p.References {
				fmt.Fprintf(&b, "- [%s](%s) (%s)\n", r.Title, r.URL, r.Source)
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// JSON renders the Suggestion as indented JSON. Useful for piping into
// jq or storing alongside the daily digest output.
func (s Suggestion) JSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "\t")
}
