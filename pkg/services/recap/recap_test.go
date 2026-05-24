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

package recap_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/services/recap"
)

func TestNew_requiresMetrics(t *testing.T) {
	_, err := recap.New(nil)
	require.Error(t, err)
}

// Friday 2026-05-22 is in ISO week W21. Monday of that week is 2026-05-18.
var (
	fri = time.Date(2026, 5, 22, 15, 0, 0, 0, time.UTC)
	mon = time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
)

func TestTop_DefaultWindowIsThisISOWeek(t *testing.T) {
	ctrl := gomock.NewController(t)
	mr := mockengagement.NewMockMetricsRepository(ctrl)

	mr.EXPECT().
		ItemList(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f engagement.MetricsFilter) ([]engagement.ItemMetrics, error) {
			require.NotNil(t, f.From)
			require.NotNil(t, f.To)
			assert.True(t, f.From.Equal(mon), "From should be Monday 00:00 UTC, got %s", *f.From)
			assert.True(t, f.To.Equal(fri), "To should be now")
			assert.Equal(t, 3, f.Limit, "default limit is 3")
			return []engagement.ItemMetrics{
				{ItemID: 1, Title: "A", URL: "https://a", Clicks: 30},
				{ItemID: 2, Title: "B", URL: "https://b", Clicks: 20},
				{ItemID: 3, Title: "C", URL: "https://c", Clicks: 10},
			}, nil
		})

	svc, err := recap.New(mr)
	require.NoError(t, err)

	top, err := svc.Top(context.Background(), fri, recap.TopOptions{})
	require.NoError(t, err)
	require.True(t, top.HasItems())
	assert.Equal(t, "2026-W21", top.Period.Label)
	assert.Len(t, top.Items, 3)
	assert.Equal(t, int64(30), top.Items[0].Clicks)
}

func TestTop_RespectsCustomLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	mr := mockengagement.NewMockMetricsRepository(ctrl)

	mr.EXPECT().
		ItemList(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f engagement.MetricsFilter) ([]engagement.ItemMetrics, error) {
			assert.Equal(t, 5, f.Limit)
			return []engagement.ItemMetrics{}, nil
		})

	svc, _ := recap.New(mr)
	_, err := svc.Top(context.Background(), fri, recap.TopOptions{N: 5})
	require.NoError(t, err)
}

func TestTop_RespectsCustomWindow(t *testing.T) {
	ctrl := gomock.NewController(t)
	mr := mockengagement.NewMockMetricsRepository(ctrl)

	mr.EXPECT().
		ItemList(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f engagement.MetricsFilter) ([]engagement.ItemMetrics, error) {
			require.NotNil(t, f.From)
			assert.True(t, f.From.Equal(fri.Add(-24*time.Hour)), "From should be 24h before now")
			return nil, nil
		})

	svc, _ := recap.New(mr)
	_, err := svc.Top(context.Background(), fri, recap.TopOptions{Window: 24 * time.Hour})
	require.NoError(t, err)
}

func TestTop_BelowMinItemsReturnsZeroValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	mr := mockengagement.NewMockMetricsRepository(ctrl)
	mr.EXPECT().
		ItemList(gomock.Any(), gomock.Any()).
		Return([]engagement.ItemMetrics{
			{ItemID: 1, Title: "A", URL: "https://a", Clicks: 5},
			{ItemID: 2, Title: "B", URL: "https://b", Clicks: 2},
		}, nil)

	svc, _ := recap.New(mr)
	top, err := svc.Top(context.Background(), fri, recap.TopOptions{MinItems: 3})
	require.NoError(t, err)
	assert.False(t, top.HasItems(), "below MinItems must return zero value")
	assert.Empty(t, top.Period.Label)
}

func TestTop_PropagatesRepoErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	mr := mockengagement.NewMockMetricsRepository(ctrl)
	mr.EXPECT().
		ItemList(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("boom"))

	svc, _ := recap.New(mr)
	_, err := svc.Top(context.Background(), fri, recap.TopOptions{})
	require.Error(t, err)
}

func TestTop_SundayUsesPreviousISOWeek(t *testing.T) {
	ctrl := gomock.NewController(t)
	mr := mockengagement.NewMockMetricsRepository(ctrl)

	// Sunday 2026-05-24 — ISO-week wise still belongs to W21 (Mon 2026-05-18).
	sun := time.Date(2026, 5, 24, 15, 0, 0, 0, time.UTC)

	mr.EXPECT().
		ItemList(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f engagement.MetricsFilter) ([]engagement.ItemMetrics, error) {
			assert.True(t, f.From.Equal(mon), "Sunday should still treat the prior Mon as week start, got %s", *f.From)
			return []engagement.ItemMetrics{{ItemID: 1, Clicks: 1}}, nil
		})

	svc, _ := recap.New(mr)
	top, err := svc.Top(context.Background(), sun, recap.TopOptions{})
	require.NoError(t, err)
	assert.Equal(t, "2026-W21", top.Period.Label)
}
