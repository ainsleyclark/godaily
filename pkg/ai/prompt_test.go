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

package ai

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/news"
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

func TestBuildSystemBlocks(t *testing.T) {
	t.Parallel()

	blocks := buildSystemBlocks()
	require.Len(t, blocks, 2)

	assert.Equal(t, systemIntro, blocks[0].Text)
	assert.Empty(t, blocks[0].CacheControl.Type, "intro block must not be the cache breakpoint")

	assert.Contains(t, blocks[1].Text, "Voice & style guide", "second block must embed style.md")
	assert.Contains(t, blocks[1].Text, "FACTUAL ONLY")
	assert.Equal(t, "ephemeral", string(blocks[1].CacheControl.Type), "trailing block must carry cache breakpoint")
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

func TestBuildSystemText(t *testing.T) {
	t.Parallel()

	t.Run("Empty Blocks", func(t *testing.T) {
		t.Parallel()
		got := buildSystemText(nil)
		assert.Equal(t, "", got)
	})

	t.Run("Single Block", func(t *testing.T) {
		t.Parallel()
		blocks := buildSystemBlocks()[:1]
		got := buildSystemText(blocks)
		assert.Equal(t, systemIntro, got)
	})

	t.Run("Two Blocks Joined With Double Newline", func(t *testing.T) {
		t.Parallel()
		blocks := buildSystemBlocks()
		got := buildSystemText(blocks)
		assert.Contains(t, got, systemIntro)
		assert.Contains(t, got, "\n\n")
		assert.Contains(t, got, "Style guide")
	})

	t.Run("Contains Both Block Texts", func(t *testing.T) {
		t.Parallel()
		blocks := buildDigestSystemBlocks()
		got := buildSystemText(blocks)
		assert.Contains(t, got, digestSystemIntro)
		assert.Contains(t, got, "Voice & style guide")
	})
}
