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

package social

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	socialgw "github.com/ainsleyclark/godaily/pkg/gateway/social"
	mockai "github.com/ainsleyclark/godaily/pkg/mocks/ai"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/domain/news"
	mockslack "github.com/ainsleyclark/godaily/pkg/mocks/slack"
	mocksocial "github.com/ainsleyclark/godaily/pkg/mocks/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/featured"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// constReframer returns a reframer that always returns text.
func constReframer(text string) reframer {
	return func(_ context.Context, _ ai.Prompter, _ featured.Featured) (string, error) {
		return text, nil
	}
}

type fixture struct {
	t         *testing.T
	ctrl      *gomock.Controller
	prompter  *mockai.MockPrompter
	issues    *mocknews.MockIssueRepository
	items     *mocknews.MockItemRepository
	posts     *mocknews.MockSocialPostRepository
	slack     *mockslack.MockSender
	posters   []socialgw.Poster
	reframers map[socialgw.Platform]reframer
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	return &fixture{
		t:        t,
		ctrl:     ctrl,
		prompter: mockai.NewMockPrompter(ctrl),
		issues:   mocknews.NewMockIssueRepository(ctrl),
		items:    mocknews.NewMockItemRepository(ctrl),
		posts:    mocknews.NewMockSocialPostRepository(ctrl),
		slack:    mockslack.NewMockSender(ctrl),
	}
}

func (f *fixture) service() *Service {
	f.t.Helper()
	svc, err := New(f.posters, f.prompter, f.issues, f.items, f.posts, f.slack)
	require.NoError(f.t, err)
	if f.reframers != nil {
		svc.reframers = f.reframers
	}
	return svc
}

// stubReframer replaces the reframer for one platform on this fixture's
// service. It does not touch any package-level state, so tests are safe
// to run in parallel.
func (f *fixture) stubReframer(p socialgw.Platform, stub reframer) {
	if f.reframers == nil {
		f.reframers = defaultReframers()
	}
	f.reframers[p] = stub
}

// newMockPoster returns a Poster whose Platform() returns p.
func newMockPoster(ctrl *gomock.Controller, p socialgw.Platform) *mocksocial.MockPoster {
	mp := mocksocial.NewMockPoster(ctrl)
	mp.EXPECT().Platform().Return(p).AnyTimes()
	return mp
}

const sampleFeatureURL = "https://go.dev/blog/go1.30"

// featureJSON is a canned model response that featured.Feature accepts.
func featureJSON() []byte {
	return []byte(`{"title":"Go 1.30 released","url":"` + sampleFeatureURL + `","source":"go_release","tag":"release","hook":"Go 1.30 ships generic type inference improvements."}`)
}

func sampleIssue() news.Issue {
	return news.Issue{ID: 42, Slug: "2026-05-20"}
}

func sampleItems() []news.Item {
	return []news.Item{
		{
			Title:  "Go 1.30 released",
			URL:    "https://go.dev/blog/go1.30",
			Source: news.SourceGoRelease,
			Tag:    news.TagRelease,
			Score:  0.9,
		},
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	t.Run("Happy path", func(t *testing.T) {
		t.Parallel()
		_, err := New(nil, f.prompter, f.issues, f.items, f.posts, nil)
		require.NoError(t, err)
	})

	tt := map[string]func(*fixture) error{
		"Nil prompter":    func(f *fixture) error { _, e := New(nil, nil, f.issues, f.items, f.posts, nil); return e },
		"Nil issues":      func(f *fixture) error { _, e := New(nil, f.prompter, nil, f.items, f.posts, nil); return e },
		"Nil items":       func(f *fixture) error { _, e := New(nil, f.prompter, f.issues, nil, f.posts, nil); return e },
		"Nil socialposts": func(f *fixture) error { _, e := New(nil, f.prompter, f.issues, f.items, nil, nil); return e },
	}
	for name, build := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := build(f)
			require.Error(t, err)
		})
	}
}

func TestService_HasPosters(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	assert.False(t, f.service().HasPosters())

	f.posters = []socialgw.Poster{newMockPoster(f.ctrl, socialgw.PlatformBluesky)}
	assert.True(t, f.service().HasPosters())
}

func TestService_Post_NoPostersSkips(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	res, err := f.service().Post(t.Context(), PostOptions{Date: time.Now()})
	require.NoError(t, err)
	assert.Empty(t, res)
}

func TestService_Post_IssueNotFound(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.posters = []socialgw.Poster{newMockPoster(f.ctrl, socialgw.PlatformBluesky)}

	f.issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-20").
		Return(news.Issue{}, store.ErrNotFound)

	date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
	_, err := f.service().Post(t.Context(), PostOptions{Date: date})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no digest")
}

func TestService_Post_NoItemsSkips(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.posters = []socialgw.Poster{newMockPoster(f.ctrl, socialgw.PlatformBluesky)}

	issue := sampleIssue()
	f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(issue, nil)
	f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil)

	date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
	res, err := f.service().Post(t.Context(), PostOptions{Date: date})
	require.NoError(t, err)
	assert.Empty(t, res)
}

func TestService_Post_HappyPath(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	issue := sampleIssue()

	f.stubReframer(socialgw.PlatformBluesky, constReframer(
		"Go 1.30 lands generic inference improvements.\n\nhttps://go.dev/blog/go1.30\n#golang",
	))

	bluesky := newMockPoster(f.ctrl, socialgw.PlatformBluesky)
	bluesky.EXPECT().Post(gomock.Any(), gomock.Any()).Return(
		socialgw.Result{PostURL: "https://bsky.app/profile/godaily/post/abc"}, nil,
	)
	f.posters = []socialgw.Poster{bluesky}

	f.issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-20").Return(issue, nil)
	f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
	f.prompter.EXPECT().
		Prompt(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(featureJSON(), nil)

	f.posts.EXPECT().
		HasPosted(gomock.Any(), int64(42), "bluesky").Return(false, nil)
	f.posts.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p news.SocialPost) (news.SocialPost, error) {
			require.NotNil(t, p.IssueID)
			assert.Equal(t, int64(42), *p.IssueID)
			assert.Equal(t, news.SocialPostKindFeatured, p.Kind)
			assert.Equal(t, "bluesky", p.Platform)
			assert.Contains(t, p.Text, "Go 1.30")
			assert.Equal(t, "https://bsky.app/profile/godaily/post/abc", p.PostURL)
			p.ID = 1
			return p, nil
		})

	// One success Slack notification expected, carrying the post URL.
	var successMsg string
	f.slack.EXPECT().
		MustSend(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, m string) { successMsg = m })

	date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
	res, err := f.service().Post(t.Context(), PostOptions{Date: date})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, socialgw.PlatformBluesky, res[0].Platform)
	assert.Equal(t, "https://bsky.app/profile/godaily/post/abc", res[0].PostURL)
	assert.False(t, res[0].Skipped)
	assert.Contains(t, successMsg, "featured")
	assert.Contains(t, successMsg, "Bluesky")
	assert.Contains(t, successMsg, "https://bsky.app/profile/godaily/post/abc")
}

func TestService_Post_SkipsAlreadyPosted(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	bluesky := newMockPoster(f.ctrl, socialgw.PlatformBluesky)
	// bluesky.Post must NOT be called when HasPosted returns true.
	f.posters = []socialgw.Poster{bluesky}

	f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(sampleIssue(), nil)
	f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
	f.prompter.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(featureJSON(), nil)

	f.posts.EXPECT().HasPosted(gomock.Any(), int64(42), "bluesky").Return(true, nil)

	date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
	res, err := f.service().Post(t.Context(), PostOptions{Date: date})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.True(t, res[0].Skipped)
}

func TestService_Post_DryRun(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	f.stubReframer(socialgw.PlatformBluesky, constReframer("dry-run text"))

	bluesky := newMockPoster(f.ctrl, socialgw.PlatformBluesky)
	// bluesky.Post must NOT be called in dry-run.
	f.posters = []socialgw.Poster{bluesky}

	f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(sampleIssue(), nil)
	f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
	f.prompter.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(featureJSON(), nil)
	// posts.HasPosted + posts.Create must NOT be called in dry-run.

	date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
	res, err := f.service().Post(t.Context(), PostOptions{Date: date, DryRun: true})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "dry-run text", res[0].Text)
	assert.Empty(t, res[0].PostURL)
}

func TestService_Post_PosterErrorNotifiesSlack(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	f.stubReframer(socialgw.PlatformBluesky, constReframer("ok"))

	bluesky := newMockPoster(f.ctrl, socialgw.PlatformBluesky)
	bluesky.EXPECT().Post(gomock.Any(), gomock.Any()).Return(socialgw.Result{}, errors.New("API down"))
	f.posters = []socialgw.Poster{bluesky}

	f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(sampleIssue(), nil)
	f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
	f.prompter.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(featureJSON(), nil)
	f.posts.EXPECT().HasPosted(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)

	// Capture the Slack notification so we can assert on its content.
	var slackMsg string
	f.slack.EXPECT().
		MustSend(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, m string) { slackMsg = m })

	date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
	res, err := f.service().Post(t.Context(), PostOptions{Date: date})
	require.Error(t, err)
	require.Len(t, res, 1)
	assert.Contains(t, res[0].Err.Error(), "API down")
	assert.Contains(t, slackMsg, "bluesky")
}

func TestService_Post_PlatformsFilter(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	f.stubReframer(socialgw.PlatformMastodon, constReframer("mastodon text"))

	bluesky := newMockPoster(f.ctrl, socialgw.PlatformBluesky)
	mastodon := newMockPoster(f.ctrl, socialgw.PlatformMastodon)
	mastodon.EXPECT().Post(gomock.Any(), gomock.Any()).Return(
		socialgw.Result{PostURL: "https://mastodon.social/@godaily/9"}, nil,
	)
	f.posters = []socialgw.Poster{bluesky, mastodon}

	f.issues.EXPECT().FindBySlug(gomock.Any(), gomock.Any()).Return(sampleIssue(), nil)
	f.items.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleItems(), nil)
	f.prompter.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(featureJSON(), nil)
	f.posts.EXPECT().HasPosted(gomock.Any(), gomock.Any(), "mastodon").Return(false, nil)
	f.posts.EXPECT().Create(gomock.Any(), gomock.Any()).Return(news.SocialPost{}, nil)

	// Wet run posts a single platform — one success Slack notification.
	f.slack.EXPECT().MustSend(gomock.Any(), gomock.Any())

	date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
	res, err := f.service().Post(t.Context(), PostOptions{
		Date:      date,
		Platforms: []socialgw.Platform{socialgw.PlatformMastodon},
	})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, socialgw.PlatformMastodon, res[0].Platform)
}

func TestSelectPosters(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	bluesky := newMockPoster(ctrl, socialgw.PlatformBluesky)
	linkedin := newMockPoster(ctrl, socialgw.PlatformLinkedIn)
	mastodon := newMockPoster(ctrl, socialgw.PlatformMastodon)
	all := []socialgw.Poster{bluesky, linkedin, mastodon}

	t.Run("Empty wanted returns all", func(t *testing.T) {
		t.Parallel()
		got := selectPosters(all, nil)
		assert.Len(t, got, 3)
	})

	t.Run("Subset returns only matches", func(t *testing.T) {
		t.Parallel()
		got := selectPosters(all, []socialgw.Platform{socialgw.PlatformLinkedIn, socialgw.PlatformMastodon})
		require.Len(t, got, 2)
		assert.Equal(t, socialgw.PlatformLinkedIn, got[0].Platform())
		assert.Equal(t, socialgw.PlatformMastodon, got[1].Platform())
	})

	t.Run("Unknown wanted yields empty", func(t *testing.T) {
		t.Parallel()
		got := selectPosters(all, []socialgw.Platform{"twitter"})
		assert.Empty(t, got)
	})
}
