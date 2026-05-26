// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/featured"
)

// Service publishes social media posts for both the daily featured slot
// and the Tue/Fri rotation slot.
type Service struct {
	posters    []platform.Poster
	prompter   ai.Prompter
	issues     digest.IssueRepository
	items      news.ItemRepository
	posts      social.PostRepository
	slack      slack.Sender
	reframers  map[social.Platform]reframer
	candidates []Candidate
}

// New creates a new social Service. posters may be empty (nothing to post);
// the service errors if prompter, issues, items, or posts are nil.
// slackSender may be nil to disable Slack notifications. Rotation candidates
// must be wired separately via WithCandidates if Rotate will be called.
func New(
	posters []platform.Poster,
	prompter ai.Prompter,
	issues digest.IssueRepository,
	items news.ItemRepository,
	posts social.PostRepository,
	slackSender slack.Sender,
) (*Service, error) {
	if prompter == nil {
		return nil, errors.New("social: ai.Prompter is required")
	}
	if issues == nil || items == nil {
		return nil, errors.New("social: issue and item repositories are required")
	}
	if posts == nil {
		return nil, errors.New("social: social post repository is required")
	}
	return &Service{
		posters:   posters,
		prompter:  prompter,
		issues:    issues,
		items:     items,
		posts:     posts,
		slack:     slackSender,
		reframers: defaultReframers(),
	}, nil
}

// WithCandidates registers the rotation candidates the service offers when
// Rotate is called. Order matters per-day but final selection is by the
// day-aware logic in rotation.go.
func (s *Service) WithCandidates(cs ...Candidate) *Service {
	s.candidates = cs
	return s
}

// HasPosters reports whether the service has any platforms configured.
// Useful for callers that want to short-circuit when no creds are set.
func (s *Service) HasPosters() bool {
	return len(s.posters) > 0
}

// reframer reframes a featured item for one platform. Function-typed so
// tests can inject deterministic text without going through the AI.
type reframer func(ctx context.Context, p ai.Prompter, f featured.Featured) (string, error)

// defaultReframers maps each Platform to its production reframing prompt.
func defaultReframers() map[social.Platform]reframer {
	return map[social.Platform]reframer{
		social.Bluesky:  featured.Bluesky,
		social.LinkedIn: featured.LinkedIn,
		social.Mastodon: featured.Mastodon,
	}
}
