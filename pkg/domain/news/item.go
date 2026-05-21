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

// Item defines a Go Daily news item.
type Item struct {
	ID          int64     `json:"id"`
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

//go:generate go run go.uber.org/mock/mockgen -package=mocknews -destination=../../mocks/domain/news/ItemRepository.go . ItemRepository

// ItemListOptions filters for List queries.
type ItemListOptions struct {
	IssueID *int64
	From    *time.Time
	To      *time.Time
}

// ItemRepository defines the methods for interacting with the Item store.
type ItemRepository interface {
	Find(ctx context.Context, id int64) (Item, error)
	List(ctx context.Context, opts ItemListOptions) ([]Item, error)
	Create(ctx context.Context, issueID *int64, position int, item Item) (Item, error)
	DeleteByIssue(ctx context.Context, issueID int64) error
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

type Tag string

const (
	TagArticle          Tag = "article"
	TagProposal         Tag = "proposal"
	TagProposalAccepted Tag = "proposal_accepted"
	TagProposalShipped  Tag = "proposal_shipped"
	TagVideo            Tag = "video"
	TagPodcast          Tag = "podcast"
	TagRelease          Tag = "release"
	TagDiscussion       Tag = "discussion"
	TagTrending         Tag = "trending"
	TagEvent            Tag = "event"
)

// SectionTags lists the canonical section tags in display order. Each digest
// section is keyed by one of these tags; other tags fold into one of them via
// Tag.Section().
var SectionTags = []Tag{
	TagRelease,
	TagProposal,
	TagDiscussion,
	TagEvent,
	TagArticle,
	TagVideo,
	TagTrending,
}

// NoLimit disables the per-section item cap when used in SectionLimits.
const NoLimit = 0

// SectionLimits caps the number of items shown per section in a digest.
// Use NoLimit (0) for unlimited. Adjust these to tune digest density.
var SectionLimits = map[Tag]int{
	TagEvent:      5,
	TagRelease:    5,
	TagProposal:   NoLimit,
	TagArticle:    5,
	TagDiscussion: 8,
	TagVideo:      5,
	TagTrending:   5,
}

// Section returns the canonical section tag this tag renders under.
// TagPodcast folds into TagVideo; the proposal-lifecycle tags fold into
// TagProposal. Other tags return themselves.
func (t Tag) Section() Tag {
	switch t {
	case TagPodcast:
		return TagVideo
	case TagProposalAccepted, TagProposalShipped:
		return TagProposal
	}
	return t
}

var sectionTitles = map[Tag]string{
	TagEvent:      "Events",
	TagRelease:    "Releases",
	TagProposal:   "Proposals",
	TagArticle:    "Articles",
	TagDiscussion: "Discussions",
	TagVideo:      "Videos",
	TagTrending:   "Trending",
}

// Title returns the display heading for a section tag. Defined for the six
// canonical section tags; non-section tags resolve via Section() first.
func (t Tag) Title() string {
	return sectionTitles[t.Section()]
}
