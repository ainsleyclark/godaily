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

package news

import (
	"context"
	"time"
)

// Fetcher defines the method for obtaining news items
// from various sources.
type Fetcher interface {
	// Fetch obtains a transforms news articles.
	//
	// Source types are responsible for returning errors
	// if they could not be obtained.
	Fetch(ctx context.Context) ([]Item, error)
}

// SourceItems groups a source with its fetched news items.
type SourceItems struct {
	Source Source `json:"source"`
	Items  []Item `json:"items"`
}

// Author holds identity information about the person or entity that
// published or submitted a news item.
type Author struct {
	Name       string `json:"name,omitempty"`
	Username   string `json:"username,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	ProfileURL string `json:"profile_url,omitempty"`
}

// String returns the best display name for the author, safe on a nil receiver.
func (a *Author) String() string {
	if a == nil {
		return ""
	}
	if a.Name != "" {
		return a.Name
	}
	return a.Username
}

// Item defines a Go Daily news item.
type Item struct {
	Source      Source    `json:"source"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`                    // click target — the external content the source is linking to
	OriginalURL string    `json:"original_url,omitempty"` // listing on the source platform (e.g. HN comments page), when different from URL
	ImageURL    string    `json:"image_url,omitempty"`
	Author      *Author   `json:"author,omitempty"`
	Snippet     string    `json:"snippet"`
	Tag         Tag       `json:"tag"` // source-specific hint ("proposal-accepted", "trending", "official")
	Comments    int       `json:"comments"`
	Score       float64   `json:"score"` // per-source relevance/popularity, normalised across sources
	Published   time.Time `json:"published"`
}

type Tag string

const (
	TagArticle          Tag = "article"
	TagProposal         Tag = "proposal"
	TagProposalAccepted Tag = "proposal_accepted"
	TagProposalShipped  Tag = "proposal_shipped"
	TagVideo            Tag = "video"
	TagPodcast          Tag = "podcast"
)
