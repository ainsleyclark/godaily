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

package candidates_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	socialgw "github.com/ainsleyclark/godaily/pkg/gateway/social"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/domain/news"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidates"
)

// confsYAML and meetupsYAML are deliberately small so subject expectations
// stay readable. Conferences end_date covers a past + a future entry so
// the upcoming filter has something to discard.
const (
	confsYAML = `
- slug: alpha-conf
  name: Alpha Conf
  url: https://alpha-conf.example
  location: Berlin
  description: Alpha conference.
  end_date: "2099-01-01"
  linkedin: alpha-conf
  bluesky: alpha-conf.example
  mastodon: ""
- slug: bravo-conf
  name: Bravo Conf
  url: https://bravo-conf.example
  location: Tokyo
  description: Bravo conference.
  end_date: "2099-06-01"
  linkedin: ""
  bluesky: ""
  mastodon: ""
- slug: past-conf
  name: Past Conf
  url: https://past-conf.example
  location: Nowhere
  description: This conference has already happened.
  end_date: "2000-01-01"
  linkedin: ""
  bluesky: ""
  mastodon: ""
`

	meetupsYAML = `
- slug: alpha-meetup
  name: Alpha Meetup
  url: https://alpha-meetup.example
  location: London
  description: Alpha meetup.
  linkedin: alpha-meetup-group
  bluesky: ""
  mastodon: ""
- slug: bravo-meetup
  name: Bravo Meetup
  url: https://bravo-meetup.example
  location: Paris
  description: Bravo meetup.
  linkedin: ""
  bluesky: ""
  mastodon: ""
`
)

// weekIndexForTest mirrors community.weekIndex so tests can predict
// which pool a given `now` selects without exposing the production
// helper. Keep in sync with community.go.
func weekIndexForTest(t time.Time) int {
	return int(t.UTC().Unix() / (7 * 24 * 60 * 60))
}

// pickWednesdayMatching scans forward from start (must be a Wednesday)
// and returns the first Wednesday whose weekIndex % 3 matches want.
// Used to find test-fixture dates without hand-computing modulos.
func pickWednesdayMatching(start time.Time, want int) time.Time {
	require := func() {
		if start.Weekday() != time.Wednesday {
			panic("pickWednesdayMatching: start must be a Wednesday")
		}
	}
	require()
	for i := 0; i < 3; i++ {
		w := start.AddDate(0, 0, 7*i)
		if weekIndexForTest(w)%3 == want {
			return w
		}
	}
	panic("no matching Wednesday found within 3 weeks (impossible)")
}

// Pick fixture dates: a meetup-week Wed (mod 3 in {0,1}) and a
// conference-week Wed (mod 3 == 2). 2026-05-20 is a Wednesday; from
// there we find the nearest Wednesday matching each pool slot.
var (
	startWed = time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC)
	wedMeet  = pickWednesdayMatching(startWed, 0)
	wedConf  = pickWednesdayMatching(startWed, 2)
)

func TestCommunity_Kind(t *testing.T) {
	c := candidates.NewCommunity([]byte("[]"), []byte("[]"), nil)
	assert.Equal(t, news.SocialPostKindCommunity, c.Kind())
}

func TestCommunity_Eligible(t *testing.T) {
	t.Run("Meetup week picks first unposted meetup alphabetically", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocknews.NewMockSocialPostRepository(ctrl)

		year := wedMeet.Year()
		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:alpha-meetup:%d", year), "bluesky").
			Return(false, nil)

		c := candidates.NewCommunity([]byte(confsYAML), []byte(meetupsYAML), posts)
		cctx, ok, err := c.Eligible(context.Background(), wedMeet)
		require.NoError(t, err)
		require.True(t, ok)

		assert.Equal(t, news.SocialPostKindCommunity, cctx.Kind)
		assert.Equal(t, fmt.Sprintf("community:alpha-meetup:%d", year), cctx.Subject)
		assert.Equal(t, "https://alpha-meetup.example", cctx.URL)
		assert.Equal(t, "https://www.linkedin.com/company/alpha-meetup-group", cctx.Mentions[socialgw.PlatformLinkedIn])
	})

	t.Run("Conference week picks first unposted upcoming conference", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocknews.NewMockSocialPostRepository(ctrl)

		year := wedConf.Year()
		// past-conf is filtered out before any DB check; alpha-conf is first
		// alphabetically among upcoming.
		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:alpha-conf:%d", year), "bluesky").
			Return(false, nil)

		c := candidates.NewCommunity([]byte(confsYAML), []byte(meetupsYAML), posts)
		cctx, ok, err := c.Eligible(context.Background(), wedConf)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, fmt.Sprintf("community:alpha-conf:%d", year), cctx.Subject)
		assert.Equal(t, "@alpha-conf.example", cctx.Mentions[socialgw.PlatformBluesky])
		assert.Equal(t, "https://www.linkedin.com/company/alpha-conf", cctx.Mentions[socialgw.PlatformLinkedIn])
	})

	t.Run("Rotates past already-posted entry", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocknews.NewMockSocialPostRepository(ctrl)

		year := wedMeet.Year()
		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:alpha-meetup:%d", year), "bluesky").
			Return(true, nil)
		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:bravo-meetup:%d", year), "bluesky").
			Return(false, nil)

		c := candidates.NewCommunity([]byte(confsYAML), []byte(meetupsYAML), posts)
		cctx, ok, err := c.Eligible(context.Background(), wedMeet)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, fmt.Sprintf("community:bravo-meetup:%d", year), cctx.Subject)
	})

	t.Run("Exhausted meetup pool falls through to conferences", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocknews.NewMockSocialPostRepository(ctrl)

		year := wedMeet.Year()
		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:alpha-meetup:%d", year), "bluesky").
			Return(true, nil)
		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:bravo-meetup:%d", year), "bluesky").
			Return(true, nil)
		// past-conf is filtered before any DB hit. First fallback is alpha-conf.
		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:alpha-conf:%d", year), "bluesky").
			Return(false, nil)

		c := candidates.NewCommunity([]byte(confsYAML), []byte(meetupsYAML), posts)
		cctx, ok, err := c.Eligible(context.Background(), wedMeet)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, fmt.Sprintf("community:alpha-conf:%d", year), cctx.Subject)
	})

	t.Run("Both pools exhausted returns not eligible", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocknews.NewMockSocialPostRepository(ctrl)

		year := wedConf.Year()
		// Conference week → conferences first, then meetups.
		for _, slug := range []string{"alpha-conf", "bravo-conf", "alpha-meetup", "bravo-meetup"} {
			posts.EXPECT().
				HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:%s:%d", slug, year), "bluesky").
				Return(true, nil)
		}

		c := candidates.NewCommunity([]byte(confsYAML), []byte(meetupsYAML), posts)
		_, ok, err := c.Eligible(context.Background(), wedConf)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestCommunity_Generate(t *testing.T) {
	t.Run("Splices LinkedIn URL when handle is configured", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocknews.NewMockSocialPostRepository(ctrl)

		year := wedMeet.Year()
		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:alpha-meetup:%d", year), "bluesky").
			Return(false, nil)

		c := candidates.NewCommunity([]byte(confsYAML), []byte(meetupsYAML), posts)
		cctx, ok, err := c.Eligible(context.Background(), wedMeet)
		require.NoError(t, err)
		require.True(t, ok)

		out, err := c.Generate(context.Background(), nil, socialgw.PlatformLinkedIn, cctx)
		require.NoError(t, err)
		assert.Contains(t, out, "https://www.linkedin.com/company/alpha-meetup-group")
		assert.Contains(t, out, "https://alpha-meetup.example")
		assert.Contains(t, out, "Alpha meetup.")
	})

	t.Run("Falls back to plain name when platform has no mention", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocknews.NewMockSocialPostRepository(ctrl)

		year := wedMeet.Year()
		// alpha-meetup has no Bluesky handle in the fixture.
		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:alpha-meetup:%d", year), "bluesky").
			Return(false, nil)

		c := candidates.NewCommunity([]byte(confsYAML), []byte(meetupsYAML), posts)
		cctx, ok, err := c.Eligible(context.Background(), wedMeet)
		require.NoError(t, err)
		require.True(t, ok)

		out, err := c.Generate(context.Background(), nil, socialgw.PlatformBluesky, cctx)
		require.NoError(t, err)
		assert.Contains(t, out, "Alpha Meetup")
		assert.NotContains(t, out, "@") // no Bluesky handle → no @mention.
	})

	t.Run("Renders Bluesky mention with @ prefix when handle is set", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocknews.NewMockSocialPostRepository(ctrl)

		year := wedConf.Year()
		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:alpha-conf:%d", year), "bluesky").
			Return(false, nil)

		c := candidates.NewCommunity([]byte(confsYAML), []byte(meetupsYAML), posts)
		cctx, ok, err := c.Eligible(context.Background(), wedConf)
		require.NoError(t, err)
		require.True(t, ok)

		out, err := c.Generate(context.Background(), nil, socialgw.PlatformBluesky, cctx)
		require.NoError(t, err)
		assert.Contains(t, out, "@alpha-conf.example")
	})

	t.Run("Unknown platform returns empty string", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocknews.NewMockSocialPostRepository(ctrl)

		year := wedMeet.Year()
		posts.EXPECT().
			HasPostedBySubject(gomock.Any(), fmt.Sprintf("community:alpha-meetup:%d", year), "bluesky").
			Return(false, nil)

		c := candidates.NewCommunity([]byte(confsYAML), []byte(meetupsYAML), posts)
		cctx, ok, err := c.Eligible(context.Background(), wedMeet)
		require.NoError(t, err)
		require.True(t, ok)

		out, err := c.Generate(context.Background(), nil, socialgw.Platform("nonexistent"), cctx)
		require.NoError(t, err)
		assert.Empty(t, out)
	})
}

// TestCommunity_CycleRatio walks several consecutive Wednesdays and
// confirms the rotation produces ~2 meetup picks for every 1 conference
// pick (M-M-C). This guards against silent regressions to the
// promoCycleLen constant.
func TestCommunity_CycleRatio(t *testing.T) {
	const weeks = 12

	ctrl := gomock.NewController(t)
	posts := mocknews.NewMockSocialPostRepository(ctrl)
	// Every check returns false (nothing posted yet), so the picker
	// always lands on the first entry in the chosen pool.
	posts.EXPECT().
		HasPostedBySubject(gomock.Any(), gomock.Any(), "bluesky").
		Return(false, nil).
		AnyTimes()

	c := candidates.NewCommunity([]byte(confsYAML), []byte(meetupsYAML), posts)

	var meetups, conferences int
	for i := 0; i < weeks; i++ {
		now := startWed.AddDate(0, 0, 7*i)
		cctx, ok, err := c.Eligible(context.Background(), now)
		require.NoError(t, err)
		require.True(t, ok)
		switch {
		case strings.Contains(cctx.Subject, "-meetup:"):
			meetups++
		case strings.Contains(cctx.Subject, "-conf:"):
			conferences++
		}
	}

	// 12 Wednesdays in the M-M-C pattern = 8 meetups + 4 conferences.
	assert.Equal(t, 8, meetups, "expected 8 meetup weeks in 12-week window")
	assert.Equal(t, 4, conferences, "expected 4 conference weeks in 12-week window")
}
