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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/domain/news"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidates"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/rotation"
)

// Tuesday 2026-05-19 at 15:00 UTC — the scheduled rotation slot. Tests
// that don't care about the day-of-week just reuse this constant.
var ctaNow = time.Date(2026, 5, 19, 15, 0, 0, 0, time.UTC)

func TestCTA_Kind(t *testing.T) {
	c := candidates.NewCTA(nil)
	assert.Equal(t, news.SocialPostKindCTA, c.Kind())
}

func TestCTA_EligibleWhenCooldownClear(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := mocknews.NewMockSocialPostRepository(ctrl)

	// HasPostedKindSince must be called with `now - 7d`. We pin `now`
	// and assert that the `since` arg is the right delta.
	posts.EXPECT().
		HasPostedKindSince(gomock.Any(), news.SocialPostKindCTA, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ news.SocialPostKind, platform string, since time.Time) (bool, error) {
			assert.Equal(t, "bluesky", platform, "anchor platform must be bluesky")
			assert.True(t, since.Equal(ctaNow.Add(-7*24*time.Hour)),
				"since must be exactly 7 days before now, got %s", since)
			return false, nil
		})

	c := candidates.NewCTA(posts)
	cctx, ok, err := c.Eligible(context.Background(), ctaNow)
	require.NoError(t, err)
	require.True(t, ok)

	assert.Equal(t, news.SocialPostKindCTA, cctx.Kind)
	assert.True(t, strings.HasPrefix(cctx.Subject, "cta:"), "subject should be 'cta:<key>', got %q", cctx.Subject)
	assert.Equal(t, "https://godaily.dev/", cctx.URL)

	payload, ok := cctx.Payload.(rotation.CTAPayload)
	require.True(t, ok, "Payload must be a rotation.CTAPayload")
	assert.NotEmpty(t, payload.Angle, "angle must be selected from rotation list")
}

func TestCTA_BlockedByCooldown(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := mocknews.NewMockSocialPostRepository(ctrl)

	posts.EXPECT().
		HasPostedKindSince(gomock.Any(), news.SocialPostKindCTA, "bluesky", gomock.Any()).
		Return(true, nil)

	c := candidates.NewCTA(posts)
	_, ok, err := c.Eligible(context.Background(), ctaNow)
	require.NoError(t, err)
	assert.False(t, ok, "cooldown must block the CTA")
}

func TestCTA_RepoErrorIsPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := mocknews.NewMockSocialPostRepository(ctrl)

	posts.EXPECT().
		HasPostedKindSince(gomock.Any(), news.SocialPostKindCTA, "bluesky", gomock.Any()).
		Return(false, errors.New("db down"))

	c := candidates.NewCTA(posts)
	_, _, err := c.Eligible(context.Background(), ctaNow)
	require.Error(t, err)
}

func TestCTA_AngleIsStableWithinSameISOWeek(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := mocknews.NewMockSocialPostRepository(ctrl)
	posts.EXPECT().
		HasPostedKindSince(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(false, nil).Times(2)

	c := candidates.NewCTA(posts)

	tueSameWeek := time.Date(2026, 5, 19, 15, 0, 0, 0, time.UTC) // Tue
	thuSameWeek := time.Date(2026, 5, 21, 15, 0, 0, 0, time.UTC) // Thu — same ISO W21

	a, ok, err := c.Eligible(context.Background(), tueSameWeek)
	require.NoError(t, err)
	require.True(t, ok)

	b, ok, err := c.Eligible(context.Background(), thuSameWeek)
	require.NoError(t, err)
	require.True(t, ok)

	assert.Equal(t, a.Payload.(rotation.CTAPayload).Angle, b.Payload.(rotation.CTAPayload).Angle,
		"angle must be stable across days within the same ISO week")
	assert.Equal(t, a.Subject, b.Subject, "subject must be stable within an ISO week")
}
