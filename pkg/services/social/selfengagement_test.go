// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mocksocial "github.com/ainsleyclark/godaily/pkg/mocks/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
)

func TestSelfEngagementFetcher_Stats(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name    string
		inner   platform.Stats
		likes   int64
		reposts int64
		want    platform.Stats
	}{
		{
			name:    "subtracts offsets",
			inner:   platform.Stats{Likes: 5, Reposts: 3, Comments: 2, Impressions: 100},
			likes:   1,
			reposts: 1,
			want:    platform.Stats{Likes: 4, Reposts: 2, Comments: 2, Impressions: 100},
		},
		{
			name:    "clamps at zero",
			inner:   platform.Stats{Likes: 0, Reposts: 0},
			likes:   1,
			reposts: 1,
			want:    platform.Stats{Likes: 0, Reposts: 0},
		},
		{
			name:    "comments and impressions untouched",
			inner:   platform.Stats{Likes: 2, Reposts: 2, Comments: 7, Impressions: 500},
			likes:   1,
			reposts: 1,
			want:    platform.Stats{Likes: 1, Reposts: 1, Comments: 7, Impressions: 500},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mock := mocksocial.NewMockStatFetcher(ctrl)
			mock.EXPECT().Stats(gomock.Any(), gomock.Any()).Return(tc.inner, nil)

			f := selfEngagementFetcher{inner: mock, likes: tc.likes, reposts: tc.reposts}
			got, err := f.Stats(context.Background(), "https://example.com")
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSelfEngagementFetcher_PropagatesError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	sentinel := errors.New("platform down")
	mock := mocksocial.NewMockStatFetcher(ctrl)
	mock.EXPECT().Stats(gomock.Any(), gomock.Any()).Return(platform.Stats{}, sentinel)

	f := selfEngagementFetcher{inner: mock, likes: 1, reposts: 1}
	_, err := f.Stats(context.Background(), "https://example.com")
	assert.ErrorIs(t, err, sentinel)
}
