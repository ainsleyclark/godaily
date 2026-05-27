// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidate"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/featured"
)

var _ social.Service = (*Service)(nil)

// Service publishes social media posts for both the daily featured slot
// and the Tue/Fri rotation slot.
type Service struct {
	posters      []platform.Poster
	prompter     ai.Prompter
	issues       digest.IssueRepository
	items        news.ItemRepository
	posts        social.PostRepository
	slack        slack.Sender
	reframers    map[social.Platform]reframer
	candidates   []candidate.Candidate
	statFetchers map[social.Platform]platform.StatFetcher
}

// New creates a new social Service. It reads platform credentials from
// config to bootstrap posters, rotation candidates, and stat fetchers.
// The service errors if prompter, issues, items, or posts are nil.
// slackSender may be nil to disable Slack notifications. metrics may be
// nil — the recap candidate is then skipped.
func New(
	config env.Config,
	prompter ai.Prompter,
	issues digest.IssueRepository,
	items news.ItemRepository,
	posts social.PostRepository,
	metrics engagement.MetricsRepository,
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
		posters:      buildPosters(config),
		prompter:     prompter,
		issues:       issues,
		items:        items,
		posts:        posts,
		slack:        slackSender,
		reframers:    defaultReframers(),
		candidates:   buildCandidates(posts, metrics),
		statFetchers: buildStatFetchers(config),
	}, nil
}

// StatFetchers returns the per-platform StatFetcher map built from the
// config the service was constructed with. Consumed by the engagement
// metrics flow at app level.
func (s *Service) StatFetchers() map[social.Platform]platform.StatFetcher {
	return s.statFetchers
}

// hasPosters reports whether the service has any platforms configured.
// Evaluated internally by Post and Rotate to short-circuit when no creds
// are wired.
func (s *Service) hasPosters() bool {
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
