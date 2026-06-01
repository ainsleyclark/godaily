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
		"Sorts Across Sources By Score Desc": {
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
		"Section Priority Beats Score": {
			// A high-scoring security item must not outrank or crowd out a
			// lower-scoring proposal: proposals are a higher-priority section.
			sections: []news.SourceItems{
				{Source: news.SourceGoVuln, Items: []news.Item{
					{Source: news.SourceGoVuln, Title: "vuln", URL: "https://v", Tag: news.TagSecurity, Score: 0.99},
				}},
				{Source: news.SourceGitHub, Items: []news.Item{
					{Source: news.SourceGitHub, Title: "prop", URL: "https://p", Tag: news.TagProposal, Score: 0.10},
				}},
			},
			cfg: filterConfig{topPerSource: 3, totalCap: 1},
			want: func(t *testing.T, items []promptItem) {
				t.Helper()
				require.Len(t, items, 1)
				assert.Equal(t, "Proposals", items[0].Section)
				assert.Equal(t, "prop", items[0].Title)
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
	assert.Contains(t, got, "Editorial voice guide")
	// The section order is injected from news.SectionTags, so Proposals must
	// appear ahead of Security in the rendered headline rule.
	assert.Contains(t, got, sectionOrder())
	assert.Less(t, strings.Index(got, "Proposals"), strings.Index(got, "Security"))
	assert.NotContains(t, got, "%s")
}
