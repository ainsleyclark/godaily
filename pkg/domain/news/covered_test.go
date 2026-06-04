// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package news_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

func TestExcludeCovered(t *testing.T) {
	t.Parallel()

	titles := func(items []news.Item) []string {
		out := make([]string, len(items))
		for i, it := range items {
			out[i] = it.Title
		}
		return out
	}

	t.Run("No covered items returns input unchanged", func(t *testing.T) {
		t.Parallel()
		items := []news.Item{{Title: "Go 1.26.4 is released", URL: "https://go.dev/dl"}}
		assert.Equal(t, items, news.ExcludeCovered(items, nil))
	})

	t.Run("Drops exact canonical URL match", func(t *testing.T) {
		t.Parallel()
		items := []news.Item{
			{Title: "Go 1.26.4 is released", URL: "https://go.dev/dl"},
			{Title: "A new web framework", URL: "https://example.com/web"},
		}
		covered := []news.Item{{Title: "Go 1.26.4 is released", URL: "https://go.dev/dl"}}
		assert.Equal(t, []string{"A new web framework"}, titles(news.ExcludeCovered(items, covered)))
	})

	t.Run("Drops when item URL matches a covered original URL", func(t *testing.T) {
		t.Parallel()
		items := []news.Item{{Title: "Thread", URL: "https://news.ycombinator.com/item?id=1"}}
		covered := []news.Item{{Title: "Some article", URL: "https://blog/x", OriginalURL: "https://news.ycombinator.com/item?id=1"}}
		assert.Empty(t, news.ExcludeCovered(items, covered))
	})

	t.Run("Drops cross-source re-post by normalised title despite different URL", func(t *testing.T) {
		t.Parallel()
		// The exact bug: a release covered yesterday from go.dev, re-posted
		// today as an r/golang thread carrying the Reddit permalink as its URL.
		items := []news.Item{
			{Title: "Go 1.26.4 is released!", URL: "https://www.reddit.com/r/golang/comments/1tvgabw/go_1264_is_released/", Tag: news.TagDiscussion},
			{Title: "Generics deep dive", URL: "https://example.com/generics", Tag: news.TagArticle},
		}
		covered := []news.Item{
			{Title: "Go 1.26.4 is released", URL: "https://go.dev/blog/go1.26.4", Tag: news.TagRelease},
		}
		got := news.ExcludeCovered(items, covered)
		assert.Equal(t, []string{"Generics deep dive"}, titles(got))
	})

	t.Run("Keeps distinct stories", func(t *testing.T) {
		t.Parallel()
		items := []news.Item{
			{Title: "Go 1.27 proposal accepted", URL: "https://go.dev/p/27"},
			{Title: "New profiling tool released", URL: "https://example.com/prof"},
		}
		covered := []news.Item{{Title: "Go 1.26.4 is released", URL: "https://go.dev/dl"}}
		assert.Len(t, news.ExcludeCovered(items, covered), 2)
	})

	t.Run("Generic short title does not suppress a distinct item", func(t *testing.T) {
		t.Parallel()
		// Two-token titles carry too little signal to match on, so they must
		// not collide just because the words happen to overlap.
		items := []news.Item{{Title: "Go released", URL: "https://example.com/a"}}
		covered := []news.Item{{Title: "Go released", URL: "https://go.dev/other"}}
		assert.Len(t, news.ExcludeCovered(items, covered), 1, "thin title must not be a match key")
	})

	t.Run("Title match tolerates punctuation and case differences", func(t *testing.T) {
		t.Parallel()
		items := []news.Item{{Title: "GO 1.26.4 — IS released", URL: "https://a"}}
		covered := []news.Item{{Title: "go 1.26.4 is released", URL: "https://b"}}
		assert.Empty(t, news.ExcludeCovered(items, covered))
	})
}
