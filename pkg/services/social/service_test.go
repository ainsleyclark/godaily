// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/mocks/ai"
	"github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/pkg/mocks/slack"
	"github.com/ainsleyclark/godaily/pkg/mocks/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/featured"
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
	issues    *mockdigest.MockIssueRepository
	items     *mocknews.MockItemRepository
	posts     *mocksocial.MockPostRepository
	slack     *mockslack.MockSender
	posters   []platform.Poster
	reframers map[social.Platform]reframer
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	return &fixture{
		t:        t,
		ctrl:     ctrl,
		prompter: mockai.NewMockPrompter(ctrl),
		issues:   mockdigest.NewMockIssueRepository(ctrl),
		items:    mocknews.NewMockItemRepository(ctrl),
		posts:    mocksocial.NewMockPostRepository(ctrl),
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
func (f *fixture) stubReframer(p social.Platform, stub reframer) {
	if f.reframers == nil {
		f.reframers = defaultReframers()
	}
	f.reframers[p] = stub
}

// newMockPoster returns a Poster whose Platform() returns p.
func newMockPoster(ctrl *gomock.Controller, p social.Platform) *mocksocial.MockPoster {
	mp := mocksocial.NewMockPoster(ctrl)
	mp.EXPECT().Platform().Return(p).AnyTimes()
	return mp
}

const sampleFeatureURL = "https://go.dev/blog/go1.30"

// featureJSON is a canned model response that featured.Feature accepts.
func featureJSON() []byte {
	return []byte(`{"title":"Go 1.30 released","url":"` + sampleFeatureURL + `","source":"go_release","tag":"release","hook":"Go 1.30 ships generic type inference improvements."}`)
}

func sampleIssue() digest.Issue {
	return digest.Issue{ID: 42, Slug: "2026-05-20"}
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

	f.posters = []platform.Poster{newMockPoster(f.ctrl, social.Bluesky)}
	assert.True(t, f.service().HasPosters())
}
