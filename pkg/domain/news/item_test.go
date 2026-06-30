// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package news

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTag_Section(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   Tag
		want Tag
	}{
		"Release stays release":       {in: TagRelease, want: TagRelease},
		"Article stays article":       {in: TagArticle, want: TagArticle},
		"Discussion stays discussion": {in: TagDiscussion, want: TagDiscussion},
		"Video stays video":           {in: TagVideo, want: TagVideo},
		"Trending stays trending":     {in: TagTrending, want: TagTrending},
		"Proposal stays proposal":     {in: TagProposal, want: TagProposal},
		"ProposalAccepted is its own": {in: TagProposalAccepted, want: TagProposalAccepted},
		"ProposalShipped folds":       {in: TagProposalShipped, want: TagProposal},
		"Podcast folds into video":    {in: TagPodcast, want: TagVideo},
		"Jobs stays jobs":             {in: TagJobs, want: TagJobs},
		"Unknown tag returns itself":  {in: Tag("mystery"), want: Tag("mystery")},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, test.in.Section())
		})
	}
}

func TestTag_Title(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   Tag
		want string
	}{
		"Release":          {in: TagRelease, want: "Releases"},
		"Article":          {in: TagArticle, want: "Articles"},
		"Discussion":       {in: TagDiscussion, want: "Discussions"},
		"Video":            {in: TagVideo, want: "Videos"},
		"Trending":         {in: TagTrending, want: "Trending"},
		"Proposal":         {in: TagProposal, want: "Proposals"},
		"ProposalAccepted": {in: TagProposalAccepted, want: "Accepted Proposals"},
		"ProposalShipped":  {in: TagProposalShipped, want: "Proposals"},
		"Podcast":          {in: TagPodcast, want: "Videos"},
		"Jobs":             {in: TagJobs, want: "Jobs"},
		"Unknown empty":    {in: Tag("mystery"), want: ""},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, test.in.Title())
		})
	}
}

func TestAuthor_String(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input *Author
		want  string
	}{
		"Nil receiver":      {input: nil, want: ""},
		"Name set":          {input: &Author{Name: "Alan Donovan"}, want: "Alan Donovan"},
		"Username only":     {input: &Author{Username: "griesemer"}, want: "griesemer"},
		"Name and username": {input: &Author{Name: "Robert Griesemer", Username: "griesemer"}, want: "Robert Griesemer"},
		"All fields":        {input: &Author{Name: "Ainsley", Username: "ainsleyclark", AvatarURL: "https://example.com/avatar.png", ProfileURL: "https://github.com/ainsleyclark"}, want: "Ainsley"},
		"Empty struct":      {input: &Author{}, want: ""},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := test.input.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSelectForDigest(t *testing.T) {
	t.Parallel()

	mk := func(tag Tag, title string, score float64) Item {
		return Item{Tag: tag, Title: title, URL: "https://example.com/" + title, Score: score}
	}
	titles := func(items []Item) []string {
		out := make([]string, len(items))
		for i, it := range items {
			out[i] = it.Title
		}
		return out
	}

	t.Run("Orders sections canonically and scores within", func(t *testing.T) {
		t.Parallel()
		// Articles (lower in SectionTags) given before Releases (higher); the
		// output must put the release section first, and order each section by
		// score descending.
		in := []Item{
			mk(TagArticle, "article-low", 1),
			mk(TagArticle, "article-high", 9),
			mk(TagRelease, "release-only", 5),
		}
		got := SelectForDigest(in)
		assert.Equal(t, []string{"release-only", "article-high", "article-low"}, titles(got))
	})

	t.Run("Applies per-section caps", func(t *testing.T) {
		t.Parallel()
		// Security caps at 3 (SectionLimits); the two lowest-scored drop.
		in := []Item{
			mk(TagSecurity, "sec-1", 10),
			mk(TagSecurity, "sec-2", 8),
			mk(TagSecurity, "sec-3", 6),
			mk(TagSecurity, "sec-4", 4),
			mk(TagSecurity, "sec-5", 2),
		}
		got := SelectForDigest(in)
		assert.Equal(t, []string{"sec-1", "sec-2", "sec-3"}, titles(got))
	})

	t.Run("Folds non-canonical tags into their section", func(t *testing.T) {
		t.Parallel()
		in := []Item{
			mk(TagPodcast, "a-podcast", 3),
			mk(TagVideo, "a-video", 7),
		}
		got := SelectForDigest(in)
		// Both land in the Video section, score-ordered.
		assert.Equal(t, []string{"a-video", "a-podcast"}, titles(got))
	})

	t.Run("Accepted proposals are their own section above open proposals", func(t *testing.T) {
		t.Parallel()
		// Open proposals fold to TagProposal; accepted proposals stand alone in
		// TagProposalAccepted, which sorts before TagProposal in SectionTags.
		// Shipped still folds into the open-proposal section.
		in := []Item{
			mk(TagProposal, "open", 9),
			mk(TagProposalAccepted, "accepted", 1),
			mk(TagProposalShipped, "shipped", 5),
		}
		got := SelectForDigest(in)
		assert.Equal(t, []string{"accepted", "open", "shipped"}, titles(got))
	})

	t.Run("Empty input yields empty output", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, SelectForDigest(nil))
	})
}
