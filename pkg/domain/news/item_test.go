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
		"ProposalAccepted folds":      {in: TagProposalAccepted, want: TagProposal},
		"ProposalShipped folds":       {in: TagProposalShipped, want: TagProposal},
		"Podcast folds into video":    {in: TagPodcast, want: TagVideo},
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
		"ProposalAccepted": {in: TagProposalAccepted, want: "Proposals"},
		"Podcast":          {in: TagPodcast, want: "Videos"},
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
