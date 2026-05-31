// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	mockai "github.com/ainsleyclark/godaily/pkg/mocks/ai"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	mockslack "github.com/ainsleyclark/godaily/pkg/mocks/slack"
	mocksocial "github.com/ainsleyclark/godaily/pkg/mocks/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidate"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/featured"
	slacksdk "github.com/slack-go/slack"
)

// constReframer returns a reframer that always returns text.
func constReframer(text string) reframer {
	return func(_ context.Context, _ ai.Prompter, _ featured.Featured) (string, error) {
		return text, nil
	}
}

type fixture struct {
	t          *testing.T
	ctrl       *gomock.Controller
	prompter   *mockai.MockPrompter
	issues     *mockdigest.MockIssueRepository
	items      *mocknews.MockItemRepository
	posts      *mocksocial.MockPostRepository
	slack      *mockslack.MockSender
	posters    []platform.Poster
	candidates []candidate.Candidate
	reframers  map[social.Platform]reframer
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

// service constructs a Service with the fixture's mocks. Posters and
// candidates are injected directly rather than bootstrapped from env —
// the package's bootstrap helpers are tested separately and would
// otherwise require real credentials to produce a usable poster.
func (f *fixture) service() *Service {
	f.t.Helper()
	svc, err := New(env.Config{}, f.prompter, f.issues, f.items, f.posts, nil, f.slack)
	require.NoError(f.t, err)
	svc.posters = f.posters
	svc.candidates = f.candidates
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

// flattenSlackRequest concatenates the Request's Text, every section /
// header block's text, and every button's label + URL into one string.
// Tests use it to assert on rich Slack messages with a single Contains.
func flattenSlackRequest(req slack.Request) string {
	var b strings.Builder
	b.WriteString(req.Text)
	for _, blk := range req.Blocks.BlockSet {
		switch v := blk.(type) {
		case *slacksdk.SectionBlock:
			if v.Text != nil {
				b.WriteString("\n")
				b.WriteString(v.Text.Text)
			}
		case *slacksdk.HeaderBlock:
			if v.Text != nil {
				b.WriteString("\n")
				b.WriteString(v.Text.Text)
			}
		case *slacksdk.ActionBlock:
			for _, el := range v.Elements.ElementSet {
				if btn, ok := el.(*slacksdk.ButtonBlockElement); ok {
					b.WriteString("\n")
					if btn.Text != nil {
						b.WriteString(btn.Text.Text)
						b.WriteString(" ")
					}
					b.WriteString(btn.URL)
				}
			}
		}
	}
	return b.String()
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
		_, err := New(env.Config{}, f.prompter, f.issues, f.items, f.posts, nil, nil)
		require.NoError(t, err)
	})

	tt := map[string]func(*fixture) error{
		"Nil prompter": func(f *fixture) error { _, e := New(env.Config{}, nil, f.issues, f.items, f.posts, nil, nil); return e },
		"Nil issues": func(f *fixture) error {
			_, e := New(env.Config{}, f.prompter, nil, f.items, f.posts, nil, nil)
			return e
		},
		"Nil items": func(f *fixture) error {
			_, e := New(env.Config{}, f.prompter, f.issues, nil, f.posts, nil, nil)
			return e
		},
		"Nil socialposts": func(f *fixture) error {
			_, e := New(env.Config{}, f.prompter, f.issues, f.items, nil, nil, nil)
			return e
		},
	}
	for name, build := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := build(f)
			require.Error(t, err)
		})
	}
}
