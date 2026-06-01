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

func newFetcher(t *testing.T, stats platform.Stats, liked, reposted bool) selfEngagementFetcher {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	statMock := mocksocial.NewMockStatFetcher(ctrl)
	statMock.EXPECT().Stats(gomock.Any(), gomock.Any()).Return(stats, nil)

	checkerMock := mocksocial.NewMockReactionChecker(ctrl)
	checkerMock.EXPECT().HasLiked(gomock.Any(), gomock.Any()).Return(liked, nil)
	checkerMock.EXPECT().HasReposted(gomock.Any(), gomock.Any()).Return(reposted, nil)

	return selfEngagementFetcher{inner: statMock, checker: checkerMock}
}

func TestSelfEngagementFetcher_Stats(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name     string
		inner    platform.Stats
		liked    bool
		reposted bool
		want     platform.Stats
	}{
		{
			name:     "subtracts when both liked and reposted",
			inner:    platform.Stats{Likes: 5, Reposts: 3, Comments: 2, Impressions: 100},
			liked:    true,
			reposted: true,
			want:     platform.Stats{Likes: 4, Reposts: 2, Comments: 2, Impressions: 100},
		},
		{
			name:     "subtracts only likes when not reposted",
			inner:    platform.Stats{Likes: 5, Reposts: 3, Comments: 2, Impressions: 100},
			liked:    true,
			reposted: false,
			want:     platform.Stats{Likes: 4, Reposts: 3, Comments: 2, Impressions: 100},
		},
		{
			name:     "subtracts only reposts when not liked",
			inner:    platform.Stats{Likes: 5, Reposts: 3, Comments: 2, Impressions: 100},
			liked:    false,
			reposted: true,
			want:     platform.Stats{Likes: 5, Reposts: 2, Comments: 2, Impressions: 100},
		},
		{
			name:     "subtracts nothing when neither liked nor reposted",
			inner:    platform.Stats{Likes: 5, Reposts: 3, Comments: 2, Impressions: 100},
			liked:    false,
			reposted: false,
			want:     platform.Stats{Likes: 5, Reposts: 3, Comments: 2, Impressions: 100},
		},
		{
			name:     "clamps likes at zero",
			inner:    platform.Stats{Likes: 0, Reposts: 0},
			liked:    true,
			reposted: true,
			want:     platform.Stats{Likes: 0, Reposts: 0},
		},
		{
			name:     "comments and impressions are untouched",
			inner:    platform.Stats{Likes: 2, Reposts: 2, Comments: 7, Impressions: 500},
			liked:    true,
			reposted: true,
			want:     platform.Stats{Likes: 1, Reposts: 1, Comments: 7, Impressions: 500},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newFetcher(t, tc.inner, tc.liked, tc.reposted)
			got, err := f.Stats(context.Background(), "https://example.com")
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSelfEngagementFetcher_PropagatesInnerError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	sentinel := errors.New("platform down")
	statMock := mocksocial.NewMockStatFetcher(ctrl)
	statMock.EXPECT().Stats(gomock.Any(), gomock.Any()).Return(platform.Stats{}, sentinel)

	checkerMock := mocksocial.NewMockReactionChecker(ctrl)

	f := selfEngagementFetcher{inner: statMock, checker: checkerMock}
	_, err := f.Stats(context.Background(), "https://example.com")
	assert.ErrorIs(t, err, sentinel)
}

func TestSelfEngagementFetcher_CheckerErrorsAreNonFatal(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	inner := platform.Stats{Likes: 5, Reposts: 3, Comments: 1, Impressions: 50}
	statMock := mocksocial.NewMockStatFetcher(ctrl)
	statMock.EXPECT().Stats(gomock.Any(), gomock.Any()).Return(inner, nil)

	checkerErr := errors.New("API timeout")
	checkerMock := mocksocial.NewMockReactionChecker(ctrl)
	checkerMock.EXPECT().HasLiked(gomock.Any(), gomock.Any()).Return(false, checkerErr)
	checkerMock.EXPECT().HasReposted(gomock.Any(), gomock.Any()).Return(false, checkerErr)

	f := selfEngagementFetcher{inner: statMock, checker: checkerMock}
	got, err := f.Stats(context.Background(), "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, inner, got, "nothing should be deducted when checker fails")
}
