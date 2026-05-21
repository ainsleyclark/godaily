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
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package digest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

func TestGroupIntoSections(t *testing.T) {
	t.Parallel()

	published := time.Now()

	t.Run("Same URL same tag is deduplicated", func(t *testing.T) {
		t.Parallel()
		items := []news.Item{
			{Source: news.SourceMeetup, URL: "https://meetup.com/e/1", Tag: news.TagEvent, Published: published},
			{Source: news.SourceMeetup, URL: "https://meetup.com/e/1", Tag: news.TagEvent, Published: published},
		}
		sections := groupIntoSections(items)
		require.Len(t, sections, 1)
		assert.Len(t, sections[0].Items, 1, "duplicate url+tag should be dropped")
	})

	t.Run("Same URL different tag both kept", func(t *testing.T) {
		// TagEvent (announcement) and a future TagEventRecap for the same URL must
		// both appear so the two-moment event lifecycle works correctly.
		t.Parallel()
		const tagEventRecap news.Tag = "event_recap" // planned tag, not yet a constant
		items := []news.Item{
			{Source: news.SourceMeetup, URL: "https://meetup.com/e/1", Tag: news.TagEvent, Published: published},
			{Source: news.SourceMeetup, URL: "https://meetup.com/e/1", Tag: tagEventRecap, Published: published},
		}
		sections := groupIntoSections(items)
		require.Len(t, sections, 1)
		assert.Len(t, sections[0].Items, 2, "same url with different tags must both be included")
	})

	t.Run("Different URLs different sources grouped correctly", func(t *testing.T) {
		t.Parallel()
		items := []news.Item{
			{Source: news.SourceGoBlog, URL: "https://go.dev/a", Tag: news.TagArticle, Published: published},
			{Source: news.SourceMeetup, URL: "https://meetup.com/e/1", Tag: news.TagEvent, Published: published},
		}
		sections := groupIntoSections(items)
		assert.Len(t, sections, 2)
	})

	t.Run("Empty input returns empty sections", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, groupIntoSections(nil))
		assert.Empty(t, groupIntoSections([]news.Item{}))
	})

	t.Run("Sections sorted by source priority", func(t *testing.T) {
		t.Parallel()
		items := []news.Item{
			{Source: news.SourceMedium, URL: "https://medium.com/a", Tag: news.TagArticle, Published: published},
			{Source: news.SourceGoBlog, URL: "https://go.dev/b", Tag: news.TagArticle, Published: published},
		}
		sections := groupIntoSections(items)
		require.Len(t, sections, 2)
		assert.Equal(t, news.SourceGoBlog, sections[0].Source, "higher priority source should be first")
	})
}

func TestAggregator_Collect_FutureDatePersistence(t *testing.T) {
	start, end := collectWindow(time.Now())
	future := end.Add(time.Hour) // simulates Published: time.Now() from a source like meetup

	registry := map[news.Source]news.Fetcher{
		news.SourceDevTo: mockFetcher{
			items: []news.Item{
				{Title: "future event", URL: "https://meetup.com/e/99", Published: future},
			},
		},
	}

	t.Run("Future-dated item is clamped and persisted", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		_, itemRepo := newTestStores(t)
		agg := Aggregator{items: itemRepo}

		_, err := agg.Collect(t.Context(), CollectOptions{Sources: []news.Source{news.SourceDevTo}})
		require.NoError(t, err)

		got, err := itemRepo.List(t.Context(), news.ItemListOptions{From: &start, To: &end})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "future event", got[0].Title)
		assert.Equal(t, start.Add(time.Hour), got[0].Published, "published should be clamped to start+1h")
	})
}

func TestAggregator_Collect_MultiDayDedup(t *testing.T) {
	start, end := collectWindow(time.Now())
	// Simulate a meetup event whose Published gets clamped on both days.
	future := end.Add(time.Hour)

	eventItem := news.Item{
		Title:     "Go London Meetup",
		URL:       "https://meetup.com/londongophers/events/1/",
		Tag:       news.TagEvent,
		Published: future,
	}

	registry := map[news.Source]news.Fetcher{
		news.SourceDevTo: mockFetcher{items: []news.Item{eventItem}},
	}

	t.Run("Same event collected on a second day is silently skipped", func(t *testing.T) {
		t.Cleanup(news.SwapRegistry(registry))

		_, itemRepo := newTestStores(t)
		agg := Aggregator{items: itemRepo}
		opts := CollectOptions{Sources: []news.Source{news.SourceDevTo}}

		// First collect — should store the event.
		_, err := agg.Collect(t.Context(), opts)
		require.NoError(t, err)

		first, err := itemRepo.List(t.Context(), news.ItemListOptions{From: &start, To: &end})
		require.NoError(t, err)
		require.Len(t, first, 1, "event should be stored on first collect")

		// Second collect (simulating the next day's run with the same event).
		// The idempotency check won't fire because we're reusing the same window
		// in this test; the unique constraint is the guard we're exercising.
		_, err = agg.Collect(t.Context(), CollectOptions{
			Sources: []news.Source{news.SourceDevTo},
			DryRun:  false,
		})
		// The second collect skips (idempotency check) — confirm no extra rows.
		require.NoError(t, err)

		second, err := itemRepo.List(t.Context(), news.ItemListOptions{From: &start, To: &end})
		require.NoError(t, err)
		assert.Len(t, second, 1, "second collect must not duplicate the event row")
	})
}
