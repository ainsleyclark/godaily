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

// stubReleases implements ReleaseFetcher with a fixed return value.
type stubReleases struct {
	out []candidates.GitHubRelease
	err error
}

func (s *stubReleases) LatestReleases(_ context.Context) ([]candidates.GitHubRelease, error) {
	return s.out, s.err
}

func TestSelfRelease_Kind(t *testing.T) {
	c := candidates.NewSelfRelease(nil, nil)
	assert.Equal(t, news.SocialPostKindSelfRelease, c.Kind())
}

func TestSelfRelease_NilFetcherIsNotEligible(t *testing.T) {
	c := candidates.NewSelfRelease(nil, nil)
	_, ok, err := c.Eligible(context.Background(), time.Now().UTC())
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestSelfRelease_FetcherErrorIsPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := mocknews.NewMockSocialPostRepository(ctrl)
	c := candidates.NewSelfRelease(&stubReleases{err: errors.New("boom")}, posts)

	_, _, err := c.Eligible(context.Background(), time.Now().UTC())
	require.Error(t, err)
}

func TestSelfRelease_NoReleasesIsNotEligible(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := mocknews.NewMockSocialPostRepository(ctrl)
	c := candidates.NewSelfRelease(&stubReleases{out: nil}, posts)

	_, ok, err := c.Eligible(context.Background(), time.Now().UTC())
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestSelfRelease_PicksFirstUnpostedNonDraftNonPrerelease(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := mocknews.NewMockSocialPostRepository(ctrl)

	releases := []candidates.GitHubRelease{
		{TagName: "v1.5.0-rc1", Prerelease: true}, // skip
		{TagName: "v1.4.0-draft", Draft: true},    // skip
		{TagName: "", Body: "no tag"},             // skip
		{
			TagName: "v1.4.0", Name: "GoDaily 1.4", HTMLURL: "https://github.com/x/r/v1.4.0",
			Body: "feature foo", PublishedAt: time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
		},
		{TagName: "v1.3.0"}, // older — wouldn't be reached
	}

	// v1.4.0 hasn't been posted yet.
	posts.EXPECT().
		HasPostedBySubject(gomock.Any(), "self_release:v1.4.0", "bluesky").
		Return(false, nil)

	c := candidates.NewSelfRelease(&stubReleases{out: releases}, posts)
	cctx, ok, err := c.Eligible(context.Background(), time.Now().UTC())
	require.NoError(t, err)
	require.True(t, ok)

	assert.Equal(t, news.SocialPostKindSelfRelease, cctx.Kind)
	assert.Equal(t, "self_release:v1.4.0", cctx.Subject)
	assert.Equal(t, "https://github.com/x/r/v1.4.0", cctx.URL)

	payload, ok := cctx.Payload.(rotation.SelfReleasePayload)
	require.True(t, ok, "Payload must be a SelfReleasePayload")
	assert.Equal(t, "v1.4.0", payload.Tag)
	assert.Equal(t, "GoDaily 1.4", payload.Name)
	assert.Equal(t, "feature foo", payload.Body)
}

func TestSelfRelease_SkipsAlreadyPostedAndPicksNext(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := mocknews.NewMockSocialPostRepository(ctrl)

	releases := []candidates.GitHubRelease{
		{TagName: "v1.4.0", HTMLURL: "https://x/v1.4.0"},
		{TagName: "v1.3.0", HTMLURL: "https://x/v1.3.0"},
	}

	posts.EXPECT().
		HasPostedBySubject(gomock.Any(), "self_release:v1.4.0", "bluesky").
		Return(true, nil)
	posts.EXPECT().
		HasPostedBySubject(gomock.Any(), "self_release:v1.3.0", "bluesky").
		Return(false, nil)

	c := candidates.NewSelfRelease(&stubReleases{out: releases}, posts)
	cctx, ok, err := c.Eligible(context.Background(), time.Now().UTC())
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "self_release:v1.3.0", cctx.Subject)
}

func TestSelfRelease_AllReleasesAlreadyPosted(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := mocknews.NewMockSocialPostRepository(ctrl)

	releases := []candidates.GitHubRelease{{TagName: "v1.4.0"}, {TagName: "v1.3.0"}}
	posts.EXPECT().
		HasPostedBySubject(gomock.Any(), "self_release:v1.4.0", "bluesky").
		Return(true, nil)
	posts.EXPECT().
		HasPostedBySubject(gomock.Any(), "self_release:v1.3.0", "bluesky").
		Return(true, nil)

	c := candidates.NewSelfRelease(&stubReleases{out: releases}, posts)
	_, ok, err := c.Eligible(context.Background(), time.Now().UTC())
	require.NoError(t, err)
	assert.False(t, ok)
}
