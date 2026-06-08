// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/mocks/engagement"
	digestsvc "github.com/ainsleyclark/godaily/pkg/services/digest"
)

// monBuild is the real recap trigger: Monday 2026-05-25 02:00 UTC
// (digest build time), which falls in ISO W22. The default recap
// window is therefore the previous complete week, W21, running
// [2026-05-18 00:00, 2026-05-25 00:00).
//
// fri is retained for the custom-window test only.
var (
	monBuild      = time.Date(2026, 5, 25, 2, 0, 0, 0, time.UTC)
	prevWeekStart = time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	prevWeekEnd   = time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	fri           = time.Date(2026, 5, 22, 15, 0, 0, 0, time.UTC)
)

func TestNewRecapService(t *testing.T) {
	t.Run("Requires a metrics repository", func(t *testing.T) {
		_, err := digestsvc.NewRecapService(nil)
		require.Error(t, err)
	})

	t.Run("Returns a service when wired correctly", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		svc, err := digestsvc.NewRecapService(mockengagement.NewMockMetricsRepository(ctrl))
		require.NoError(t, err)
		require.NotNil(t, svc)
	})
}

func TestRecapService_Top(t *testing.T) {
	// Regression: a Monday build must look back at the previous complete
	// week, not the near-empty slice of the week that just started. The
	// recap was moved Friday→Monday but the default window used to end at
	// "now", collapsing to [thisMon 00:00, thisMon 02:00) and never
	// clearing recapMinItems — so no draft was ever created.
	t.Run("Default window is the previous complete ISO week", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mr := mockengagement.NewMockMetricsRepository(ctrl)

		mr.EXPECT().
			ItemList(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, f engagement.MetricsFilter) ([]engagement.ItemMetrics, error) {
				require.NotNil(t, f.From)
				require.NotNil(t, f.To)
				assert.True(t, f.From.Equal(prevWeekStart), "From should be last Monday 00:00 UTC, got %s", *f.From)
				assert.True(t, f.To.Equal(prevWeekEnd), "To should be this Monday 00:00 UTC, got %s", *f.To)
				assert.Equal(t, 3, f.Limit, "default limit is 3")
				return []engagement.ItemMetrics{
					{ItemID: 1, Title: "A", URL: "https://a", Clicks: 30},
					{ItemID: 2, Title: "B", URL: "https://b", Clicks: 20},
					{ItemID: 3, Title: "C", URL: "https://c", Clicks: 10},
				}, nil
			})

		svc, err := digestsvc.NewRecapService(mr)
		require.NoError(t, err)

		top, err := svc.Top(context.Background(), monBuild, digest.TopOptions{})
		require.NoError(t, err)
		require.True(t, top.HasItems())
		assert.Equal(t, "2026-W21", top.Period.Label)
		assert.Len(t, top.Items, 3)
		assert.Equal(t, int64(30), top.Items[0].Clicks)
	})

	t.Run("Respects custom limit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mr := mockengagement.NewMockMetricsRepository(ctrl)

		mr.EXPECT().
			ItemList(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, f engagement.MetricsFilter) ([]engagement.ItemMetrics, error) {
				assert.Equal(t, 5, f.Limit)
				return []engagement.ItemMetrics{}, nil
			})

		svc, _ := digestsvc.NewRecapService(mr)
		_, err := svc.Top(context.Background(), fri, digest.TopOptions{N: 5})
		require.NoError(t, err)
	})

	t.Run("Respects custom window", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mr := mockengagement.NewMockMetricsRepository(ctrl)

		mr.EXPECT().
			ItemList(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, f engagement.MetricsFilter) ([]engagement.ItemMetrics, error) {
				require.NotNil(t, f.From)
				assert.True(t, f.From.Equal(fri.Add(-24*time.Hour)), "From should be 24h before now")
				return nil, nil
			})

		svc, _ := digestsvc.NewRecapService(mr)
		_, err := svc.Top(context.Background(), fri, digest.TopOptions{Window: 24 * time.Hour})
		require.NoError(t, err)
	})

	t.Run("Below MinItems returns the zero value", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mr := mockengagement.NewMockMetricsRepository(ctrl)

		mr.EXPECT().
			ItemList(gomock.Any(), gomock.Any()).
			Return([]engagement.ItemMetrics{
				{ItemID: 1, Title: "A", URL: "https://a", Clicks: 5},
				{ItemID: 2, Title: "B", URL: "https://b", Clicks: 2},
			}, nil)

		svc, _ := digestsvc.NewRecapService(mr)
		top, err := svc.Top(context.Background(), fri, digest.TopOptions{MinItems: 3})
		require.NoError(t, err)
		assert.False(t, top.HasItems(), "below MinItems must return zero value")
		assert.Empty(t, top.Period.Label)
	})

	t.Run("Propagates repository errors", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mr := mockengagement.NewMockMetricsRepository(ctrl)
		mr.EXPECT().
			ItemList(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("boom"))

		svc, _ := digestsvc.NewRecapService(mr)
		_, err := svc.Top(context.Background(), fri, digest.TopOptions{})
		require.Error(t, err)
	})

	t.Run("Sunday resolves its own ISO week, then looks back one week", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mr := mockengagement.NewMockMetricsRepository(ctrl)

		// Sunday 2026-05-24 belongs to ISO W21 (Mon 2026-05-18), so the
		// previous complete week is W20: [2026-05-11, 2026-05-18).
		sun := time.Date(2026, 5, 24, 15, 0, 0, 0, time.UTC)
		w20Start := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)

		mr.EXPECT().
			ItemList(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, f engagement.MetricsFilter) ([]engagement.ItemMetrics, error) {
				assert.True(t, f.From.Equal(w20Start), "Sunday should look back to the prior complete week, got %s", *f.From)
				assert.True(t, f.To.Equal(prevWeekStart), "To should be this ISO week's Monday, got %s", *f.To)
				return []engagement.ItemMetrics{{ItemID: 1, Clicks: 1}}, nil
			})

		svc, _ := digestsvc.NewRecapService(mr)
		top, err := svc.Top(context.Background(), sun, digest.TopOptions{})
		require.NoError(t, err)
		assert.Equal(t, "2026-W20", top.Period.Label)
	})
}
