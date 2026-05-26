// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/data"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestConferences(notifyDates ...string) *Conferences {
	var datesYAML string
	for _, d := range notifyDates {
		datesYAML += "\n    - " + d
	}
	yaml := `
- slug: gophercon-test-2099
  name: GopherCon Test 2099
  url: https://gophercon.test/
  location: Testville, TX
  start_date: 2099-08-10
  end_date: 2099-08-12
  description: "A test conference."
  notify_dates:` + datesYAML
	return NewConferences([]byte(yaml))
}

func TestConferences_Fetch_NoMatchToday(t *testing.T) {
	t.Parallel()

	c := makeTestConferences("2099-01-01", "2099-04-01", "2099-08-03")
	c.now = func() time.Time { return time.Date(2099, 5, 10, 8, 0, 0, 0, time.UTC) }

	items, err := c.Fetch(context.Background())
	require.NoError(t, err)
	assert.Empty(t, items, "no notify_date matches today; expected no items")
}

func TestConferences_Fetch_MatchesAnnouncement(t *testing.T) {
	t.Parallel()

	c := makeTestConferences("2099-05-01", "2099-06-01", "2099-08-03")
	c.now = func() time.Time { return time.Date(2099, 5, 1, 6, 0, 0, 0, time.UTC) }

	items, err := c.Fetch(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)

	got := items[0]
	assert.Equal(t, news.SourceConferences, got.Source)
	assert.Equal(t, "GopherCon Test 2099", got.Title)
	assert.Equal(t, "https://gophercon.test/#announce", got.URL)
	assert.Equal(t, news.TagConference, got.Tag)
	assert.Equal(t, "A test conference.", got.Snippet)

	// Published should be yesterday noon (inside the collect window).
	yesterday := time.Date(2099, 4, 30, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, yesterday, got.Published)
}

func TestConferences_Fetch_MatchesReminder(t *testing.T) {
	t.Parallel()

	c := makeTestConferences("2099-05-01", "2099-06-01", "2099-08-03")
	c.now = func() time.Time { return time.Date(2099, 6, 1, 6, 0, 0, 0, time.UTC) }

	items, err := c.Fetch(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)

	assert.Equal(t, "https://gophercon.test/#reminder", items[0].URL)
	assert.Equal(t, news.TagConferenceReminder, items[0].Tag)
}

func TestConferences_Fetch_MatchesAlert(t *testing.T) {
	t.Parallel()

	c := makeTestConferences("2099-05-01", "2099-06-01", "2099-08-03")
	c.now = func() time.Time { return time.Date(2099, 8, 3, 6, 0, 0, 0, time.UTC) }

	items, err := c.Fetch(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)

	assert.Equal(t, "https://gophercon.test/#alert", items[0].URL)
	assert.Equal(t, news.TagConferenceAlert, items[0].Tag)
}

func TestConferences_Fetch_MultipleConferences(t *testing.T) {
	t.Parallel()

	yaml := `
- slug: conf-a
  name: Conference A
  url: https://conf-a.test/
  location: City A
  start_date: 2099-09-01
  end_date: 2099-09-01
  description: "Conference A."
  notify_dates:
    - 2099-05-01
- slug: conf-b
  name: Conference B
  url: https://conf-b.test/
  location: City B
  start_date: 2099-10-01
  end_date: 2099-10-01
  description: "Conference B."
  notify_dates:
    - 2099-05-01
`
	c := NewConferences([]byte(yaml))
	c.now = func() time.Time { return time.Date(2099, 5, 1, 9, 0, 0, 0, time.UTC) }

	items, err := c.Fetch(context.Background())
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestConferences_Fetch_RealYAML(t *testing.T) {
	t.Parallel()

	// Smoke-test that the committed conferences.yaml parses without error.
	c := NewConferences(data.Conferences)
	assert.NotNil(t, c)
	assert.NotEmpty(t, c.conferences)
}

func TestConferences_Fetch_FallbackSnippet(t *testing.T) {
	t.Parallel()

	yaml := `
- slug: conf-no-desc
  name: Conf No Desc
  url: https://conf.test/
  location: Somewhere
  start_date: 2099-07-15
  end_date: 2099-07-15
  notify_dates:
    - 2099-05-01
`
	c := NewConferences([]byte(yaml))
	c.now = func() time.Time { return time.Date(2099, 5, 1, 9, 0, 0, 0, time.UTC) }

	items, err := c.Fetch(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	// Falls back to "Location · Date" format when description is empty.
	assert.Contains(t, items[0].Snippet, "Somewhere")
	assert.Contains(t, items[0].Snippet, "2099")
}
