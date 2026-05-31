// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package candidates_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/mocks/social"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidates"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/rotation"
)

// Tuesday 2026-05-19 at 15:00 UTC — the scheduled rotation slot.
var ctaNow = time.Date(2026, 5, 19, 15, 0, 0, 0, time.UTC)

func TestCTA_Kind(t *testing.T) {
	c := candidates.NewCTA(nil)
	assert.Equal(t, social.PostKindCTA, c.Kind())
}

func TestCTA_Eligible(t *testing.T) {
	t.Run("Eligible when cooldown clear", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocksocial.NewMockPostRepository(ctrl)

		posts.EXPECT().
			HasPostedKindSince(gomock.Any(), social.PostKindCTA, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ social.PostKind, platform string, since time.Time) (bool, error) {
				assert.Equal(t, "bluesky", platform, "anchor platform must be bluesky")
				assert.True(t, since.Equal(ctaNow.Add(-13*24*time.Hour)),
					"since must be exactly 13 days before now, got %s", since)
				return false, nil
			})

		c := candidates.NewCTA(posts)
		cctx, ok, err := c.Eligible(context.Background(), ctaNow)
		require.NoError(t, err)
		require.True(t, ok)

		assert.Equal(t, social.PostKindCTA, cctx.Kind)
		assert.True(t, strings.HasPrefix(cctx.Subject, "cta:"),
			"subject should be 'cta:<key>', got %q", cctx.Subject)
		assert.Equal(t, "https://godaily.dev/", cctx.URL)

		payload, ok := cctx.Payload.(rotation.CTAPayload)
		require.True(t, ok, "Payload must be a rotation.CTAPayload")
		assert.NotEmpty(t, payload.Angle, "angle must be selected from rotation list")
	})

	t.Run("Blocked by cooldown", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocksocial.NewMockPostRepository(ctrl)
		posts.EXPECT().
			HasPostedKindSince(gomock.Any(), social.PostKindCTA, "bluesky", gomock.Any()).
			Return(true, nil)

		c := candidates.NewCTA(posts)
		_, ok, err := c.Eligible(context.Background(), ctaNow)
		require.NoError(t, err)
		assert.False(t, ok, "cooldown must block the CTA")
	})

	t.Run("Repository error is propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocksocial.NewMockPostRepository(ctrl)
		posts.EXPECT().
			HasPostedKindSince(gomock.Any(), social.PostKindCTA, "bluesky", gomock.Any()).
			Return(false, errors.New("db down"))

		c := candidates.NewCTA(posts)
		_, _, err := c.Eligible(context.Background(), ctaNow)
		require.Error(t, err)
	})

	t.Run("Angle is stable within the same ISO week", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocksocial.NewMockPostRepository(ctrl)
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
	})
}
