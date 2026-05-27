// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

//go:generate go run go.uber.org/mock/mockgen -package=mocknews -destination=../../mocks/news/ItemRepository.go . ItemRepository

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
	TagArticle            Tag = "article"
	TagTutorial           Tag = "tutorial"
	TagProposal           Tag = "proposal"
	TagProposalAccepted   Tag = "proposal_accepted"
	TagProposalShipped    Tag = "proposal_shipped"
	TagVideo              Tag = "video"
	TagPodcast            Tag = "podcast"
	TagRelease            Tag = "release"
	TagSecurity           Tag = "security"
	TagDiscussion         Tag = "discussion"
	TagTrending           Tag = "trending"
	TagEvent              Tag = "event"
	TagConference         Tag = "conference"          // major Go conference announcement
	TagConferenceReminder Tag = "conference_reminder" // ~3 months before
	TagConferenceAlert    Tag = "conference_alert"    // ~1 week before
	TagJobs               Tag = "jobs"
)

// SectionTags lists the canonical section tags in display order. Each digest
// section is keyed by one of these tags; other tags fold into one of them via
// Tag.Section().
var SectionTags = []Tag{
	TagRelease,
	TagProposal,
	TagConference,
	TagDiscussion,
	TagEvent,
	TagArticle,
	TagTutorial,
	TagVideo,
	TagTrending,
	TagSecurity,
	TagJobs,
}

// NoLimit disables the per-section item cap when used in SectionLimits.
const NoLimit = 0

// SectionLimits caps the number of items shown per section in a digest.
// Use NoLimit (0) for unlimited. Adjust these to tune digest density.
var SectionLimits = map[Tag]int{
	TagEvent:      5,
	TagConference: NoLimit,
	TagRelease:    5,
	TagSecurity:   3,
	TagProposal:   NoLimit,
	TagArticle:    5,
	TagTutorial:   5,
	TagDiscussion: 8,
	TagVideo:      5,
	TagJobs:       5,
	TagTrending:   5,
}

// Section returns the canonical section tag this tag renders under.
// TagPodcast folds into TagVideo; the proposal-lifecycle tags fold into
// TagProposal; conference reminder/alert tags fold into TagConference.
// Other tags return themselves.
func (t Tag) Section() Tag {
	switch t {
	case TagPodcast:
		return TagVideo
	case TagProposalAccepted, TagProposalShipped:
		return TagProposal
	case TagConferenceReminder, TagConferenceAlert:
		return TagConference
	}
	return t
}

var sectionTitles = map[Tag]string{
	TagEvent:      "Events",
	TagConference: "Conferences",
	TagRelease:    "Releases",
	TagSecurity:   "Security",
	TagProposal:   "Proposals",
	TagArticle:    "Articles",
	TagTutorial:   "Tutorials",
	TagDiscussion: "Discussions",
	TagVideo:      "Videos",
	TagTrending:   "Trending",
	TagJobs:       "Jobs",
}

// Title returns the display heading for a section tag. Defined for the six
// canonical section tags; non-section tags resolve via Section() first.
func (t Tag) Title() string {
	return sectionTitles[t.Section()]
}
