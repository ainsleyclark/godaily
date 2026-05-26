// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package candidates_test

import (
	"context"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/mocks/social"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/mocks/engagement"
	"github.com/ainsleyclark/godaily/pkg/services/digest"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidates"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/rotation"
)

// Friday 2026-05-22 — ISO W21.
var recapNow = time.Date(2026, 5, 22, 15, 0, 0, 0, time.UTC)

func TestRecap_Kind(t *testing.T) {
	c := candidates.NewRecap(nil, nil)
	assert.Equal(t, social.PostKindRecap, c.Kind())
}

func TestRecap_Eligible(t *testing.T) {
	t.Run("Nil recap service is not eligible", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		posts := mocksocial.NewMockPostRepository(ctrl)

		c := candidates.NewRecap(nil, posts)
		_, ok, err := c.Eligible(context.Background(), recapNow)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("Eligible with enough items", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		metrics := mockengagement.NewMockMetricsRepository(ctrl)
		posts := mocksocial.NewMockPostRepository(ctrl)

		posts.EXPECT().
			HasPostedKindSince(gomock.Any(), social.PostKindRecap, "bluesky", gomock.Any()).
			Return(false, nil)
		metrics.EXPECT().
			ItemList(gomock.Any(), gomock.Any()).
			Return([]engagement.ItemMetrics{
				{ItemID: 1, Title: "A", URL: "https://a", Source: "go_blog", Clicks: 30},
				{ItemID: 2, Title: "B", URL: "https://b", Source: "github", Clicks: 20},
				{ItemID: 3, Title: "C", URL: "https://c", Source: "hn", Clicks: 10},
			}, nil)

		svc, err := digest.NewRecapService(metrics)
		require.NoError(t, err)

		c := candidates.NewRecap(svc, posts)
		cctx, ok, err := c.Eligible(context.Background(), recapNow)
		require.NoError(t, err)
		require.True(t, ok)

		assert.Equal(t, social.PostKindRecap, cctx.Kind)
		assert.Equal(t, "recap:2026-W21", cctx.Subject)

		payload, ok := cctx.Payload.(rotation.RecapPayload)
		require.True(t, ok, "Payload must be a RecapPayload")
		assert.Equal(t, "2026-W21", payload.WeekLabel)
		require.Len(t, payload.Items, 3)
		assert.Equal(t, "A", payload.Items[0].Title)
		assert.Equal(t, int64(30), payload.Items[0].Clicks)
	})

	t.Run("Blocked by cooldown", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		metrics := mockengagement.NewMockMetricsRepository(ctrl)
		posts := mocksocial.NewMockPostRepository(ctrl)

		posts.EXPECT().
			HasPostedKindSince(gomock.Any(), social.PostKindRecap, "bluesky", gomock.Any()).
			Return(true, nil)
		// metrics.ItemList must NOT be called when cooldown blocks.

		svc, _ := digest.NewRecapService(metrics)
		c := candidates.NewRecap(svc, posts)
		_, ok, err := c.Eligible(context.Background(), recapNow)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("Too few items is not eligible", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		metrics := mockengagement.NewMockMetricsRepository(ctrl)
		posts := mocksocial.NewMockPostRepository(ctrl)

		posts.EXPECT().
			HasPostedKindSince(gomock.Any(), social.PostKindRecap, "bluesky", gomock.Any()).
			Return(false, nil)
		metrics.EXPECT().
			ItemList(gomock.Any(), gomock.Any()).
			Return([]engagement.ItemMetrics{
				{ItemID: 1, Title: "A", URL: "https://a", Clicks: 5},
				{ItemID: 2, Title: "B", URL: "https://b", Clicks: 2},
			}, nil)

		svc, _ := digest.NewRecapService(metrics)
		c := candidates.NewRecap(svc, posts)
		_, ok, err := c.Eligible(context.Background(), recapNow)
		require.NoError(t, err)
		assert.False(t, ok, "fewer than 3 items must no-op")
	})
}
