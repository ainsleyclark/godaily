// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package news

import (
	"context"
	"sort"
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
	Position    int64     `json:"position"`  // ordering within a digest issue; 0 when not linked
	InDigest    bool      `json:"in_digest"` // true when the item is linked to a digest issue
}

//go:generate go run go.uber.org/mock/mockgen -package=mocknews -destination=../../mocks/news/ItemRepository.go . ItemRepository

// ItemSort selects the ordering used by ListBrowse.
type ItemSort string

const (
	ItemSortNew ItemSort = "new" // published DESC
	ItemSortTop ItemSort = "top" // score DESC
	ItemSortHot ItemSort = "hot" // score with recency decay
)

// ItemListOptions filters for List/ListBrowse queries.
type ItemListOptions struct {
	IssueID *int64     // only items linked to this digest issue
	From    *time.Time // only items published at or after this time
	To      *time.Time // only items published strictly before this time

	Sources  []Source // OR-match across sources
	Tags     []Tag    // OR-match across tags
	Search   string   // LIKE over title + summary
	Sort     ItemSort // New | Top | Hot ; default New
	InDigest *bool    // nil = all; true = only digested; false = only raw
	Page     int64    // 1-based; 0 = no pagination
	PerPage  int64    // 0 = default
}

// SourceCount is an aggregate count of items grouped by source.
type SourceCount struct {
	Source Source `json:"source"`
	Count  int64  `json:"count"`
}

// TagCount is an aggregate count of items grouped by tag.
type TagCount struct {
	Tag   Tag   `json:"tag"`
	Count int64 `json:"count"`
}

// ItemRepository defines the methods for interacting with the Item store.
type ItemRepository interface {
	Find(ctx context.Context, id int64) (Item, error)
	List(ctx context.Context, opts ItemListOptions) ([]Item, error)
	Count(ctx context.Context) (int64, error)
	// CountMatching returns the number of items matching opts, ignoring its
	// pagination fields. It is a cheap SELECT COUNT(*) for browse totals.
	CountMatching(ctx context.Context, opts ItemListOptions) (int64, error)
	SourceCounts(ctx context.Context) ([]SourceCount, error)
	TagCounts(ctx context.Context) ([]TagCount, error)
	Create(ctx context.Context, issueID *int64, position int, item Item) (Item, error)
	DeleteByIssue(ctx context.Context, issueID int64) error
	// Delete permanently removes the item row from the store, regardless of
	// whether it is linked to an issue. Returns store.ErrNotFound if no row
	// with the given id exists.
	Delete(ctx context.Context, id int64) error
	// LinkToIssue links a currently-unlinked item to the given draft issue,
	// setting items.issue_id and appending it after the issue's existing items
	// (position = current max + 1). Fails with digest.ErrIssueNotDraft if the
	// issue is not in draft status, and with store.ErrNotFound if the issue does
	// not exist or the item does not exist / is already linked to an issue.
	LinkToIssue(ctx context.Context, issueID, itemID int64) error
	// UnlinkFromIssue clears the items.issue_id for the given (issueID, itemID) pair.
	// The item row is preserved (it remains in the raw pool, in_digest=false). Fails
	// with digest.ErrIssueNotDraft if the issue is not in draft status, and with
	// store.ErrNotFound if the issue or the link does not exist.
	UnlinkFromIssue(ctx context.Context, issueID, itemID int64) error
	// ReorderInIssue rewrites the position of each item within the issue using
	// the supplied order — orderedItemIDs[i] gets position i. The set must match
	// the full list of currently linked items exactly; partial reorders are
	// rejected. Fails with digest.ErrIssueNotDraft if the issue is not draft.
	ReorderInIssue(ctx context.Context, issueID int64, orderedItemIDs []int64) error
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
	TagSocial             Tag = "social"
)

// SectionTags lists the canonical section tags in display order. Each digest
// section is keyed by one of these tags; other tags fold into one of them via
// Tag.Section().
var SectionTags = []Tag{
	TagRelease,
	TagProposalAccepted,
	TagProposal,
	TagConference,
	TagDiscussion,
	TagEvent,
	TagArticle,
	TagTutorial,
	TagVideo,
	TagTrending,
	TagSecurity,
	TagSocial,
	TagJobs,
}

// NoLimit disables the per-section item cap when used in SectionLimits.
const NoLimit = 0

// SectionLimits caps the number of items shown per section in a digest.
// Use NoLimit (0) for unlimited. Adjust these to tune digest density.
var SectionLimits = map[Tag]int{
	TagEvent:            5,
	TagConference:       NoLimit,
	TagRelease:          5,
	TagSecurity:         3,
	TagProposalAccepted: NoLimit,
	TagProposal:         NoLimit,
	TagArticle:          5,
	TagTutorial:         5,
	TagDiscussion:       8,
	TagVideo:            5,
	TagJobs:             5,
	TagTrending:         5,
	TagSocial:           5,
}

// SelectForDigest is the single definition of which items make up a digest and
// in what order. It groups items into canonical sections (Tag.Section()), sorts
// each section by score descending, applies the per-section SectionLimits caps,
// and returns the surviving items as a flat slice in canonical SectionTags
// order. Build links exactly this set to the issue (position = index), so the
// persisted rows alone determine what ships — the email and web are then pure
// renderers ordering by position. Input order is otherwise preserved for items
// of equal score (stable sort).
func SelectForDigest(items []Item) []Item {
	bucket := make(map[Tag][]Item, len(SectionTags))
	for _, item := range items {
		section := item.Tag.Section()
		bucket[section] = append(bucket[section], item)
	}

	out := make([]Item, 0, len(items))
	for _, tag := range SectionTags {
		sectionItems := bucket[tag]
		if len(sectionItems) == 0 {
			continue
		}
		sort.SliceStable(sectionItems, func(i, j int) bool {
			return sectionItems[i].Score > sectionItems[j].Score
		})
		if limit := SectionLimits[tag]; limit > NoLimit && len(sectionItems) > limit {
			sectionItems = sectionItems[:limit]
		}
		out = append(out, sectionItems...)
	}
	return out
}

// Section returns the canonical section tag this tag renders under.
// TagPodcast folds into TagVideo; TagProposalShipped folds into TagProposal
// (TagProposalAccepted is its own section); conference reminder/alert tags
// fold into TagConference. Other tags return themselves.
func (t Tag) Section() Tag {
	switch t {
	case TagPodcast:
		return TagVideo
	case TagProposalShipped:
		return TagProposal
	case TagConferenceReminder, TagConferenceAlert:
		return TagConference
	}
	return t
}

var sectionTitles = map[Tag]string{
	TagEvent:            "Events",
	TagConference:       "Conferences",
	TagRelease:          "Releases",
	TagSecurity:         "Security",
	TagProposalAccepted: "Accepted Proposals",
	TagProposal:         "Proposals",
	TagArticle:          "Articles",
	TagTutorial:         "Tutorials",
	TagDiscussion:       "Discussions",
	TagVideo:            "Videos",
	TagTrending:         "Trending",
	TagSocial:           "Social",
	TagJobs:             "Jobs",
}

// Title returns the display heading for a section tag. Defined for the six
// canonical section tags; non-section tags resolve via Section() first.
func (t Tag) Title() string {
	return sectionTitles[t.Section()]
}
