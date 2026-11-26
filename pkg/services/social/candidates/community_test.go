// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package candidates_test

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/meetup"
	"github.com/ainsleyclark/godaily/pkg/mocks/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidate"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidates"
)

// meetupsYAML has two Meetup.com groups and one group without a feed.
const meetupsYAML = `
- slug: alpha-meetup
  name: Alpha Meetup
  url: https://alpha-meetup.example
  meetup_group: alpha-group
  location: London, UK
  description: Alpha meetup.
  handles:
    linkedin:
      slug: alpha-meetup-group
      urn: ""
    bluesky: ""
    mastodon: ""
- slug: bravo-meetup
  name: Bravo Meetup
  url: https://bravo-meetup.example
  meetup_group: bravo-group
  location: Paris, France
  description: Bravo meetup.
  handles:
    linkedin:
      slug: ""
      urn: urn:li:organization:222
    bluesky: bravo.example
    mastodon: ""
- slug: charlie-meetup
  name: Charlie Meetup
  url: https://charlie.example
  meetup_group: ""
  location: Tokyo, Japan
  description: Not on Meetup.com.
  handles:
    linkedin:
      slug: ""
      urn: ""
    bluesky: ""
    mastodon: ""
`

// confsYAMLFor builds a conferences YAML with start dates relative to
// now: one inside the 7-42 day window (alpha, 2 weeks out, 3 days long),
// one later inside it (bravo, 4 weeks out), one too close, one too far,
// one in the past and one with no start_date.
func confsYAMLFor(now time.Time) string {
	d := func(days int) string { return now.AddDate(0, 0, days).Format(time.DateOnly) }
	return fmt.Sprintf(`
- slug: alpha-conf
  name: Alpha Conf
  url: https://alpha-conf.example
  location: Berlin
  description: Alpha conference.
  start_date: %q
  end_date: %q
  handles:
    linkedin:
      slug: alpha-conf
      urn: urn:li:organization:111
    bluesky: alpha-conf.example
    mastodon: ""
- slug: bravo-conf
  name: Bravo Conf
  url: https://bravo-conf.example
  location: Tokyo
  description: Bravo conference.
  start_date: %q
  end_date: %q
  handles:
    linkedin:
      slug: ""
      urn: ""
    bluesky: ""
    mastodon: ""
- slug: soon-conf
  name: Soon Conf
  url: https://soon.example
  location: Soon
  description: Too close to promote.
  start_date: %q
  end_date: %q
- slug: far-conf
  name: Far Conf
  url: https://far.example
  location: Far
  description: Too far away to promote.
  start_date: %q
  end_date: %q
- slug: past-conf
  name: Past Conf
  url: https://past.example
  location: Past
  description: Already happened.
  start_date: "2000-01-01"
  end_date: "2000-01-02"
- slug: dateless-conf
  name: Dateless Conf
  url: https://dateless.example
  location: Nowhere
  description: No dates.
`, d(14), d(16), d(28), d(28), d(3), d(3), d(90), d(91))
}

// fakeEvents is an in-memory EventSource keyed by Meetup group.
type fakeEvents struct {
	mu      sync.Mutex
	byGroup map[string][]meetup.Event
	errs    map[string]error
	calls   []string
}

func (f *fakeEvents) UpcomingEvents(_ context.Context, group string) ([]meetup.Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, group)
	if err := f.errs[group]; err != nil {
		return nil, err
	}
	return f.byGroup[group], nil
}

func (f *fakeEvents) sortedCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := append([]string(nil), f.calls...)
	sort.Strings(out)
	return out
}

func event(uid string, start time.Time, title string) meetup.Event {
	return meetup.Event{
		UID:   "event_" + uid + "@meetup.com",
		Title: title,
		Start: start,
		URL:   "https://www.meetup.com/x/events/" + uid + "/",
	}
}

// weekIndexForTest mirrors community.weekIndex so tests can predict
// which pool a given `now` prefers. Keep in sync with community.go.
func weekIndexForTest(t time.Time) int {
	return int(t.UTC().Unix() / (7 * 24 * 60 * 60))
}

// pickWednesdayMatching scans forward from start (a Wednesday) and
// returns the first Wednesday whose weekIndex % 3 matches want.
func pickWednesdayMatching(start time.Time, want int) time.Time {
	if start.Weekday() != time.Wednesday {
		panic("pickWednesdayMatching: start must be a Wednesday")
	}
	for i := 0; i < 3; i++ {
		w := start.AddDate(0, 0, 7*i)
		if weekIndexForTest(w)%3 == want {
			return w
		}
	}
	panic("no matching Wednesday found within 3 weeks (impossible)")
}

var (
	startWed = time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC)
	wedMeet  = pickWednesdayMatching(startWed, 0)
	wedConf  = pickWednesdayMatching(startWed, 2)
)

const anchor = "bluesky"

// allowAll makes every posted / cool-down check return false.
func allowAll(posts *mocksocial.MockPostRepository) {
	posts.EXPECT().HasPostedBySubject(gomock.Any(), gomock.Any(), anchor).Return(false, nil).AnyTimes()
	posts.EXPECT().HasPostedSubjectPrefixSince(gomock.Any(), gomock.Any(), anchor, gomock.Any()).Return(false, nil).AnyTimes()
}

func TestCommunity_Kind(t *testing.T) {
	c := candidates.NewCommunity([]byte("[]"), []byte("[]"), nil, nil)
	assert.Equal(t, social.PostKindCommunity, c.Kind())
}

func TestCommunity_Eligible(t *testing.T) {
	ctx := context.Background()

	t.Run("Meetup week picks the soonest in-window event", func(t *testing.T) {
		posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
		now := wedMeet
		src := &fakeEvents{byGroup: map[string][]meetup.Event{
			"alpha-group": {
				event("1", now.Add(24*time.Hour), "Too soon"),
				event("2", now.AddDate(0, 0, 10), "Alpha in ten days"),
			},
			"bravo-group": {
				event("3", now.AddDate(0, 0, 5), "Bravo in five days"),
				event("4", now.AddDate(0, 0, 30), "Too far"),
			},
		}}

		posts.EXPECT().HasPostedBySubject(gomock.Any(), "community:bravo-meetup:event:3", anchor).Return(false, nil)
		posts.EXPECT().HasPostedSubjectPrefixSince(gomock.Any(), "community:bravo-meetup:", anchor, now.Add(-56*24*time.Hour)).Return(false, nil)

		c := candidates.NewCommunity([]byte(confsYAMLFor(now)), []byte(meetupsYAML), posts, src)
		cctx, ok, err := c.Eligible(ctx, now)
		require.NoError(t, err)
		require.True(t, ok)

		assert.Equal(t, social.PostKindCommunity, cctx.Kind)
		assert.Equal(t, "community:bravo-meetup:event:3", cctx.Subject)
		assert.Equal(t, "https://www.meetup.com/x/events/3/", cctx.URL)
		assert.Equal(t, "@bravo.example", cctx.Mention(social.Bluesky))
		assert.Equal(t, []string{"alpha-group", "bravo-group"}, src.sortedCalls(),
			"groups without meetup_group must not be fetched")
	})

	t.Run("Skips posted events and groups in cool-down", func(t *testing.T) {
		posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
		now := wedMeet
		src := &fakeEvents{byGroup: map[string][]meetup.Event{
			"alpha-group": {event("1", now.AddDate(0, 0, 4), "A1"), event("2", now.AddDate(0, 0, 12), "A2")},
			"bravo-group": {event("3", now.AddDate(0, 0, 6), "B1")},
		}}

		gomock.InOrder(
			// A1 already posted.
			posts.EXPECT().HasPostedBySubject(gomock.Any(), "community:alpha-meetup:event:1", anchor).Return(true, nil),
			// B1 not posted, but bravo was promoted recently.
			posts.EXPECT().HasPostedBySubject(gomock.Any(), "community:bravo-meetup:event:3", anchor).Return(false, nil),
			posts.EXPECT().HasPostedSubjectPrefixSince(gomock.Any(), "community:bravo-meetup:", anchor, gomock.Any()).Return(true, nil),
			// A2 is free.
			posts.EXPECT().HasPostedBySubject(gomock.Any(), "community:alpha-meetup:event:2", anchor).Return(false, nil),
			posts.EXPECT().HasPostedSubjectPrefixSince(gomock.Any(), "community:alpha-meetup:", anchor, gomock.Any()).Return(false, nil),
		)

		c := candidates.NewCommunity([]byte(confsYAMLFor(now)), []byte(meetupsYAML), posts, src)
		cctx, ok, err := c.Eligible(ctx, now)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, "community:alpha-meetup:event:2", cctx.Subject)
	})

	t.Run("A failing feed is ignored", func(t *testing.T) {
		posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
		allowAll(posts)
		now := wedMeet
		src := &fakeEvents{
			byGroup: map[string][]meetup.Event{"bravo-group": {event("3", now.AddDate(0, 0, 5), "B1")}},
			errs:    map[string]error{"alpha-group": errors.New("403")},
		}

		c := candidates.NewCommunity([]byte(confsYAMLFor(now)), []byte(meetupsYAML), posts, src)
		cctx, ok, err := c.Eligible(ctx, now)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, "community:bravo-meetup:event:3", cctx.Subject)
	})

	t.Run("Meetup week with no events falls through to conferences", func(t *testing.T) {
		posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
		now := wedMeet
		posts.EXPECT().HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:alpha-conf:%d", now.Year()), anchor).Return(false, nil)

		c := candidates.NewCommunity([]byte(confsYAMLFor(now)), []byte(meetupsYAML), posts, &fakeEvents{})
		cctx, ok, err := c.Eligible(ctx, now)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, fmt.Sprintf("community:alpha-conf:%d", now.Year()), cctx.Subject)
	})

	t.Run("Conference week picks the soonest in-window conference without fetching feeds", func(t *testing.T) {
		posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
		now := wedConf
		year := now.Year()
		// soon/far/past/dateless are filtered before any DB check.
		gomock.InOrder(
			posts.EXPECT().HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:alpha-conf:%d", year), anchor).Return(true, nil),
			posts.EXPECT().HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:bravo-conf:%d", year), anchor).Return(false, nil),
		)
		src := &fakeEvents{}

		c := candidates.NewCommunity([]byte(confsYAMLFor(now)), []byte(meetupsYAML), posts, src)
		cctx, ok, err := c.Eligible(ctx, now)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, fmt.Sprintf("community:bravo-conf:%d", year), cctx.Subject)
		assert.Equal(t, "https://bravo-conf.example", cctx.URL)
		assert.Empty(t, src.sortedCalls())
	})

	t.Run("Conference week with no conference falls through to meetups", func(t *testing.T) {
		posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
		allowAll(posts)
		now := wedConf
		src := &fakeEvents{byGroup: map[string][]meetup.Event{"alpha-group": {event("9", now.AddDate(0, 0, 7), "A")}}}

		c := candidates.NewCommunity([]byte("[]"), []byte(meetupsYAML), posts, src)
		cctx, ok, err := c.Eligible(ctx, now)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, "community:alpha-meetup:event:9", cctx.Subject)
	})

	t.Run("Nothing timely returns not eligible", func(t *testing.T) {
		posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
		now := wedMeet
		src := &fakeEvents{byGroup: map[string][]meetup.Event{"alpha-group": {event("1", now.AddDate(0, 0, 60), "Later")}}}

		c := candidates.NewCommunity([]byte("[]"), []byte(meetupsYAML), posts, src)
		_, ok, err := c.Eligible(ctx, now)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("Nil event source only considers conferences", func(t *testing.T) {
		posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
		allowAll(posts)
		now := wedMeet

		c := candidates.NewCommunity([]byte(confsYAMLFor(now)), []byte(meetupsYAML), posts, nil)
		cctx, ok, err := c.Eligible(ctx, now)
		require.NoError(t, err)
		require.True(t, ok)
		assert.True(t, strings.HasSuffix(cctx.Subject, fmt.Sprintf(":%d", now.Year())))
	})

	t.Run("Repository errors are returned", func(t *testing.T) {
		now := wedMeet
		src := &fakeEvents{byGroup: map[string][]meetup.Event{"alpha-group": {event("1", now.AddDate(0, 0, 5), "A")}}}

		t.Run("Subject check", func(t *testing.T) {
			posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
			posts.EXPECT().HasPostedBySubject(gomock.Any(), gomock.Any(), anchor).Return(false, errors.New("db down"))
			c := candidates.NewCommunity([]byte("[]"), []byte(meetupsYAML), posts, src)
			_, _, err := c.Eligible(ctx, now)
			assert.ErrorContains(t, err, "db down")
		})

		t.Run("Cool-down check", func(t *testing.T) {
			posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
			posts.EXPECT().HasPostedBySubject(gomock.Any(), gomock.Any(), anchor).Return(false, nil)
			posts.EXPECT().HasPostedSubjectPrefixSince(gomock.Any(), gomock.Any(), anchor, gomock.Any()).Return(false, errors.New("db down"))
			c := candidates.NewCommunity([]byte("[]"), []byte(meetupsYAML), posts, src)
			_, _, err := c.Eligible(ctx, now)
			assert.ErrorContains(t, err, "db down")
		})

		t.Run("Conference check", func(t *testing.T) {
			posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
			posts.EXPECT().HasPostedBySubject(gomock.Any(), gomock.Any(), anchor).Return(false, errors.New("db down"))
			c := candidates.NewCommunity([]byte(confsYAMLFor(wedConf)), []byte("[]"), posts, nil)
			_, _, err := c.Eligible(ctx, wedConf)
			assert.ErrorContains(t, err, "db down")
		})
	})
}

func TestCommunity_Generate(t *testing.T) {
	ctx := context.Background()
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	// eligibleEvent returns the CandidateContext for a single bravo or
	// alpha event so Generate can be exercised per platform.
	eligibleEvent := func(t *testing.T, group string, ev meetup.Event) (*candidates.Community, func(social.Platform) string) {
		t.Helper()
		posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
		allowAll(posts)
		c := candidates.NewCommunity([]byte("[]"), []byte(meetupsYAML), posts,
			&fakeEvents{byGroup: map[string][]meetup.Event{group: {ev}}})
		cctx, ok, err := c.Eligible(ctx, wedMeet)
		require.NoError(t, err)
		require.True(t, ok)
		return c, func(p social.Platform) string {
			out, err := c.Generate(ctx, nil, p, cctx)
			require.NoError(t, err)
			return out
		}
	}

	t.Run("Event post carries title, local date and event URL", func(t *testing.T) {
		start := time.Date(wedMeet.Year(), wedMeet.Month(), wedMeet.Day()+8, 18, 0, 0, 0, berlin)
		_, gen := eligibleEvent(t, "alpha-group", event("42", start, "Generics in anger"))

		for _, p := range []social.Platform{social.LinkedIn, social.Bluesky, social.Mastodon} {
			out := gen(p)
			assert.Contains(t, out, `"Generics in anger"`, p)
			assert.Contains(t, out, start.Format("Mon 2 Jan")+", 18:00", p)
			assert.Contains(t, out, "London, UK", p)
			assert.Contains(t, out, "https://www.meetup.com/x/events/42/", p)
			assert.NotContains(t, out, "{{", p)
			assert.NotContains(t, out, "Alpha meetup.", "event posts use the event title, not the group description")
		}
		assert.Contains(t, gen(social.LinkedIn), "https://www.linkedin.com/company/alpha-meetup-group")
		assert.Contains(t, gen(social.Bluesky), "Alpha Meetup", "no Bluesky handle falls back to name")
		assert.Contains(t, gen(social.Mastodon), "#golang")
	})

	t.Run("Event post uses name when LinkedIn URN is configured", func(t *testing.T) {
		_, gen := eligibleEvent(t, "bravo-group", event("7", wedMeet.AddDate(0, 0, 5), "Talk"))
		out := gen(social.LinkedIn)
		assert.True(t, strings.HasPrefix(out, "Bravo Meetup"))
		assert.NotContains(t, out, "urn:li:")
		assert.True(t, strings.HasPrefix(gen(social.Bluesky), "@bravo.example"))
	})

	t.Run("All-day events omit the time", func(t *testing.T) {
		ev := event("8", time.Date(wedMeet.Year(), wedMeet.Month(), wedMeet.Day()+6, 0, 0, 0, 0, time.UTC), "Hack day")
		ev.AllDay = true
		_, gen := eligibleEvent(t, "alpha-group", ev)
		out := gen(social.Bluesky)
		assert.Contains(t, out, ev.Start.Format("Mon 2 Jan"))
		assert.NotContains(t, out, "00:00")
	})

	t.Run("Long event titles are truncated on a word boundary", func(t *testing.T) {
		title := strings.Repeat("concurrency ", 20)
		_, gen := eligibleEvent(t, "alpha-group", event("5", wedMeet.AddDate(0, 0, 5), title))
		out := gen(social.Bluesky)
		assert.Contains(t, out, `concurrency..."`)
		assert.LessOrEqual(t, len([]rune(out)), 300)
	})

	t.Run("Event without URL falls back to the group URL", func(t *testing.T) {
		ev := event("6", wedMeet.AddDate(0, 0, 5), "No link")
		ev.URL = ""
		_, gen := eligibleEvent(t, "alpha-group", ev)
		assert.Contains(t, gen(social.Bluesky), "https://alpha-meetup.example")
	})

	t.Run("Conference post carries dates and countdown", func(t *testing.T) {
		posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
		allowAll(posts)
		now := wedConf
		c := candidates.NewCommunity([]byte(confsYAMLFor(now)), []byte("[]"), posts, nil)
		cctx, ok, err := c.Eligible(ctx, now)
		require.NoError(t, err)
		require.True(t, ok)

		start, end := now.AddDate(0, 0, 14), now.AddDate(0, 0, 16)
		wantDates := start.Format("2 Jan") + " - " + end.Format("2 Jan")
		if start.Month() == end.Month() {
			wantDates = fmt.Sprintf("%d-%s", start.Day(), end.Format("2 Jan"))
		}

		for _, p := range []social.Platform{social.LinkedIn, social.Bluesky, social.Mastodon} {
			out, err := c.Generate(ctx, nil, p, cctx)
			require.NoError(t, err)
			assert.Contains(t, out, wantDates, p)
			assert.Contains(t, out, "Berlin", p)
			assert.Contains(t, out, "https://alpha-conf.example", p)
			assert.NotContains(t, out, "{{", p)
		}

		li, err := c.Generate(ctx, nil, social.LinkedIn, cctx)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(li, "Alpha Conf"))
		assert.NotContains(t, li, "urn:li:")

		bs, err := c.Generate(ctx, nil, social.Bluesky, cctx)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(bs, "@alpha-conf.example"))
	})

	t.Run("Unknown platform returns empty string", func(t *testing.T) {
		_, gen := eligibleEvent(t, "alpha-group", event("1", wedMeet.AddDate(0, 0, 5), "A"))
		assert.Empty(t, gen(social.Platform("nonexistent")))
	})

	t.Run("Missing payload is an error", func(t *testing.T) {
		c := candidates.NewCommunity([]byte("[]"), []byte("[]"), nil, nil)
		_, err := c.Generate(ctx, nil, social.Bluesky, candidate.CandidateContext{Kind: social.PostKindCommunity})
		assert.Error(t, err)
	})
}

// TestCommunity_CycleRatio walks consecutive Wednesdays with both pools
// always holding something timely and confirms the preference produces
// two meetup picks for every conference pick (M-M-C). Guards against
// silent regressions to promoCycleLen.
func TestCommunity_CycleRatio(t *testing.T) {
	const weeks = 12

	posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
	allowAll(posts)

	// A conference every week across the run keeps one in the window.
	var confs strings.Builder
	for i := 0; i < weeks+6; i++ {
		start := startWed.AddDate(0, 0, 7*i+14).Format(time.DateOnly)
		fmt.Fprintf(&confs, "- slug: conf-%02d\n  name: C%d\n  url: https://c.example\n  start_date: %q\n  end_date: %q\n", i, i, start, start)
	}

	var meetupsCount, conferences int
	for i := 0; i < weeks; i++ {
		now := startWed.AddDate(0, 0, 7*i)
		src := &fakeEvents{byGroup: map[string][]meetup.Event{"alpha-group": {event(fmt.Sprint(i), now.AddDate(0, 0, 5), "E")}}}
		c := candidates.NewCommunity([]byte(confs.String()), []byte(meetupsYAML), posts, src)

		cctx, ok, err := c.Eligible(context.Background(), now)
		require.NoError(t, err)
		require.True(t, ok)
		switch {
		case strings.Contains(cctx.Subject, ":event:"):
			meetupsCount++
		case strings.HasPrefix(cctx.Subject, "community:conf-"):
			conferences++
		}
	}

	assert.Equal(t, 8, meetupsCount, "expected 8 meetup weeks in 12-week window")
	assert.Equal(t, 4, conferences, "expected 4 conference weeks in 12-week window")
}
