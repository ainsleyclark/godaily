// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prompts

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

func TestFilterItems(t *testing.T) {
	t.Parallel()

	mk := func(src news.Source, scores ...float64) news.SourceItems {
		out := news.SourceItems{Source: src}
		for i, s := range scores {
			out.Items = append(out.Items, news.Item{
				Source: src,
				Title:  string(src) + "-" + string(rune('a'+i)),
				URL:    "https://example.com",
				Score:  s,
			})
		}
		return out
	}

	tt := map[string]struct {
		sections []news.SourceItems
		cfg      filterConfig
		want     func(t *testing.T, items []promptItem)
	}{
		"Empty Sections": {
			sections: nil,
			cfg:      defaultFilterConfig(),
			want: func(t *testing.T, items []promptItem) {
				t.Helper()
				assert.Empty(t, items)
			},
		},
		"Zero TopPerSource": {
			sections: []news.SourceItems{mk(news.SourceHN, 0.5)},
			cfg:      filterConfig{topPerSource: 0, totalCap: 12},
			want: func(t *testing.T, items []promptItem) {
				t.Helper()
				assert.Nil(t, items)
			},
		},
		"Zero TotalCap": {
			sections: []news.SourceItems{mk(news.SourceHN, 0.5)},
			cfg:      filterConfig{topPerSource: 3, totalCap: 0},
			want: func(t *testing.T, items []promptItem) {
				t.Helper()
				assert.Nil(t, items)
			},
		},
		"Caps Per Source": {
			sections: []news.SourceItems{mk(news.SourceHN, 0.9, 0.8, 0.7, 0.6, 0.5)},
			cfg:      filterConfig{topPerSource: 2, totalCap: 12},
			want: func(t *testing.T, items []promptItem) {
				t.Helper()
				require.Len(t, items, 2)
				assert.Equal(t, 0.9, items[0].Score)
				assert.Equal(t, 0.8, items[1].Score)
			},
		},
		"Sorts Within Section By Score Desc": {
			// All items share the same (zero-value) tag, so they fold into a
			// single section and round-robin reduces to plain score ordering.
			sections: []news.SourceItems{
				mk(news.SourceHN, 0.4, 0.3),
				mk(news.SourceGoBlog, 0.95, 0.5),
				mk(news.SourceReddit, 0.6),
			},
			cfg: defaultFilterConfig(),
			want: func(t *testing.T, items []promptItem) {
				t.Helper()
				require.Len(t, items, 5)
				scores := make([]float64, len(items))
				for i, it := range items {
					scores[i] = it.Score
				}
				assert.Equal(t, []float64{0.95, 0.6, 0.5, 0.4, 0.3}, scores)
			},
		},
		"Truncates To TotalCap": {
			sections: []news.SourceItems{
				mk(news.SourceHN, 0.9, 0.8, 0.7),
				mk(news.SourceReddit, 0.6, 0.5, 0.4),
				mk(news.SourceLobsters, 0.3, 0.2, 0.1),
			},
			cfg: filterConfig{topPerSource: 3, totalCap: 4},
			want: func(t *testing.T, items []promptItem) {
				t.Helper()
				require.Len(t, items, 4)
				assert.Equal(t, 0.9, items[0].Score)
				assert.Equal(t, 0.6, items[3].Score)
			},
		},
		"Round Robin Interleaves Sections": {
			// Every populated section contributes its top item before any
			// section gets a second slot, so a proposal-heavy day cannot push
			// discussions and articles down the payload. Accepted proposals are
			// their own section, distinct from open proposals.
			sections: []news.SourceItems{
				{Source: news.SourceGitHub, Items: []news.Item{
					{Source: news.SourceGitHub, Title: "prop-a", URL: "https://p1", Tag: news.TagProposal, Score: 0.9},
					{Source: news.SourceGitHub, Title: "prop-b", URL: "https://p2", Tag: news.TagProposalAccepted, Score: 0.8},
					{Source: news.SourceGitHub, Title: "prop-c", URL: "https://p3", Tag: news.TagProposal, Score: 0.7},
				}},
				{Source: news.SourceReddit, Items: []news.Item{
					{Source: news.SourceReddit, Title: "disc", URL: "https://d", Tag: news.TagDiscussion, Score: 0.6},
				}},
				{Source: news.SourceGoBlog, Items: []news.Item{
					{Source: news.SourceGoBlog, Title: "art", URL: "https://a", Tag: news.TagArticle, Score: 0.5},
				}},
			},
			cfg: defaultFilterConfig(),
			want: func(t *testing.T, items []promptItem) {
				t.Helper()
				require.Len(t, items, 5)
				// Four sections are populated; each contributes its top item in
				// round one (the first four slots) before any gets a second.
				sections := make([]string, 4)
				for i, it := range items[:4] {
					sections[i] = it.Section
				}
				assert.ElementsMatch(t, []string{"Accepted Proposals", "Proposals", "Discussions", "Articles"}, sections)
				// prop-c is the second open proposal — it only appears in round
				// two, after every section has had its first slot.
				assert.Equal(t, "prop-c", items[4].Title)
			},
		},
		"Proposals Cannot Crowd Out Other Sections": {
			// With a tight cap, the surviving slots span sections instead of
			// all going to the highest-scoring (proposal) section.
			sections: []news.SourceItems{
				{Source: news.SourceGitHub, Items: []news.Item{
					{Source: news.SourceGitHub, Title: "prop-a", URL: "https://p1", Tag: news.TagProposal, Score: 0.9},
					{Source: news.SourceGitHub, Title: "prop-b", URL: "https://p2", Tag: news.TagProposal, Score: 0.8},
					{Source: news.SourceGitHub, Title: "prop-c", URL: "https://p3", Tag: news.TagProposal, Score: 0.7},
				}},
				{Source: news.SourceReddit, Items: []news.Item{
					{Source: news.SourceReddit, Title: "disc", URL: "https://d", Tag: news.TagDiscussion, Score: 0.2},
				}},
				{Source: news.SourceGoBlog, Items: []news.Item{
					{Source: news.SourceGoBlog, Title: "art", URL: "https://a", Tag: news.TagArticle, Score: 0.1},
				}},
			},
			cfg: filterConfig{topPerSource: 3, totalCap: 3},
			want: func(t *testing.T, items []promptItem) {
				t.Helper()
				require.Len(t, items, 3)
				sections := make([]string, len(items))
				for i, it := range items {
					sections[i] = it.Section
				}
				assert.ElementsMatch(t, []string{"Proposals", "Discussions", "Articles"}, sections)
			},
		},
		"Skips Jobs Social Events And Trending": {
			sections: []news.SourceItems{
				{Source: news.SourceHNJobs, Items: []news.Item{
					{Source: news.SourceHNJobs, Title: "job", URL: "https://j", Tag: news.TagJobs, Score: 0.9},
				}},
				{Source: news.SourceMastodon, Items: []news.Item{
					{Source: news.SourceMastodon, Title: "toot", URL: "https://s", Tag: news.TagSocial, Score: 0.8},
				}},
				{Source: news.SourceMeetup, Items: []news.Item{
					{Source: news.SourceMeetup, Title: "meetup", URL: "https://e", Tag: news.TagEvent, Score: 0.7},
				}},
				{Source: news.SourceGitHubTrending, Items: []news.Item{
					{Source: news.SourceGitHubTrending, Title: "repo", URL: "https://t", Tag: news.TagTrending, Score: 0.6},
				}},
				{Source: news.SourceGoBlog, Items: []news.Item{
					{Source: news.SourceGoBlog, Title: "art", URL: "https://a", Tag: news.TagArticle, Score: 0.1},
				}},
			},
			cfg: defaultFilterConfig(),
			want: func(t *testing.T, items []promptItem) {
				t.Helper()
				require.Len(t, items, 1)
				assert.Equal(t, "art", items[0].Title)
			},
		},
		"Copies All Item Fields": {
			sections: []news.SourceItems{{
				Source: news.SourceGoBlog,
				Items: []news.Item{{
					Source:  news.SourceGoBlog,
					Title:   "t",
					URL:     "https://u",
					Author:  &news.Author{Name: "a"},
					Tag:     news.TagArticle,
					Snippet: "snip",
					Score:   0.5,
				}},
			}},
			cfg: defaultFilterConfig(),
			want: func(t *testing.T, items []promptItem) {
				t.Helper()
				require.Len(t, items, 1)
				assert.Equal(t, "go_blog", items[0].Source)
				assert.Equal(t, "t", items[0].Title)
				assert.Equal(t, "https://u", items[0].URL)
				assert.Equal(t, "a", items[0].Author)
				assert.Equal(t, "article", items[0].Tag)
				assert.Equal(t, "Articles", items[0].Section)
				assert.Equal(t, "snip", items[0].Snippet)
				assert.Equal(t, 0.5, items[0].Score)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := filterItems(test.sections, test.cfg)
			test.want(t, got)
		})
	}
}

func TestBuildUserPrompt(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, time.April, 27, 0, 0, 0, 0, time.UTC)
	items := []promptItem{
		{Source: "hacker_news", Title: "x", URL: "https://x", Score: 0.5},
	}

	got := buildUserPrompt(day, items)
	assert.Contains(t, got, "Date: 2026-04-27")
	assert.Contains(t, got, "Return the JSON object only.")

	// The middle line is JSON; round-trip it.
	idx := strings.Index(got, "[")
	end := strings.LastIndex(got, "]")
	require.GreaterOrEqual(t, idx, 0)
	require.GreaterOrEqual(t, end, idx)

	var parsed []promptItem
	require.NoError(t, json.Unmarshal([]byte(got[idx:end+1]), &parsed))
	assert.Equal(t, items, parsed)
}

func TestBuildSuggestSystem(t *testing.T) {
	t.Parallel()

	got := buildSuggestSystem()
	assert.Contains(t, got, systemIntro)
	assert.Contains(t, got, "## Style guide")
	assert.Contains(t, got, "Voice & style guide")
}

func TestBuildDigestSystem(t *testing.T) {
	t.Parallel()

	got := buildDigestSystem()
	assert.Contains(t, got, introStyleMD)
	assert.Contains(t, got, "Voice & style guide")
	// The headline rule must not reinstate a section pecking order: sections
	// carry no automatic priority and only the release/security carve-outs
	// may pre-empt editorial judgement.
	assert.Contains(t, got, "Sections have no automatic priority")
	assert.Contains(t, got, "Release carve-out")
	assert.Contains(t, got, "Severity override")
	assert.NotContains(t, got, "highest-priority section")
}
