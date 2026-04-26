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

// Item defines a Go Daily news item.
type Item struct {
	Source    Source
	Title     string
	URL       string
	Author    string
	Snippet   string
	Score     int
	Tag       Tag // source-specific hint ("proposal-accepted", "trending", "official")
	Comments  int
	Published time.Time
}

type Tag string

const (
	TagArticle          Tag = "article"
	TagProposal         Tag = "proposal"
	TagProposalAccepted Tag = "proposal_accepted"
	TagProposalShipped  Tag = "proposal_shipped"
	TagVideo            Tag = "video"
)
