// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package godaily

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/data"
	"github.com/ainsleyclark/godaily/pkg/db"
	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	audiencesvc "github.com/ainsleyclark/godaily/pkg/services/audience"
	digestsvc "github.com/ainsleyclark/godaily/pkg/services/digest"
	svcengagement "github.com/ainsleyclark/godaily/pkg/services/engagement"
	socialsvc "github.com/ainsleyclark/godaily/pkg/services/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidates"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform/bluesky"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform/linkedin"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform/mastodon"
	"github.com/ainsleyclark/godaily/pkg/store/emailevents"
	metricsstore "github.com/ainsleyclark/godaily/pkg/store/engagement"
	"github.com/ainsleyclark/godaily/pkg/store/issues"
	"github.com/ainsleyclark/godaily/pkg/store/items"
	"github.com/ainsleyclark/godaily/pkg/store/socialmetrics"
	"github.com/ainsleyclark/godaily/pkg/store/socialposts"
	"github.com/ainsleyclark/godaily/pkg/store/subscribers"
	"github.com/ainsleydev/webkit/pkg/cache"
	"github.com/pkg/errors"
)

// App defines a global state for godaily.
type App struct {
	Config         *env.Config
	DB             *sql.DB
	Repository     *Repository
	Runner         digest.Service
	Social         social.Service
	Cache          cache.Store
	Subscribers    audience.SubscriberService
	EmailEvents    engagement.EventService
	Slack          slack.Sender
	MetricsService engagement.MetricsService
	StatFetchers   map[social.Platform]platform.StatFetcher
}

// Repository defines the datastore for the application.
type Repository struct {
	Issues        digest.IssueRepository
	Items         news.ItemRepository
	Subscribers   audience.SubscriberRepository
	SocialPosts   social.PostRepository
	EmailEvents   engagement.EmailEventRepository
	SocialMetrics engagement.SocialMetricRepository
	Metrics       engagement.MetricsRepository
}

// Service aggregates the domain-level service interfaces for the app.
type Service struct {
	Digest      digest.Service
	Subscribers audience.SubscriberService
	Social      social.Service
	Metrics     engagement.MetricsService
	Events      engagement.EventService
}

// Bootstrap ties all the app dependencies together
// and returns a new App.
func Bootstrap(ctx context.Context) (*App, func(), error) {
	config, err := env.New(ctx)
	if err != nil {
		return nil, func() {}, err
	}

	if news.HasSources() {
		if err = news.Materialise(config); err != nil {
			return nil, func() {}, errors.Wrap(err, "materialising sources")
		}
	}

	conn, err := db.New(ctx, config.TursoURL, config.TursoAuthToken)
	teardown := func() {
		if err = conn.Close(); err != nil {
			slog.ErrorContext(ctx, "Closing connection to database", "error", err)
		}
	}
	if err != nil {
		return nil, teardown, err
	}

	issueStore := issues.New(conn)

	var store cache.Store
	store = cache.NewInMemory(time.Hour * 24 * 30)
	if config.IsDevelopment() {
		osCache, err := cache.NewOSCache(".cache", true)
		if err != nil {
			return nil, teardown, err
		}
		store = osCache
	}

	subsStore := subscribers.New(conn)
	socialPostsStore := socialposts.New(conn)
	itemStore := items.New(conn)

	repo := &Repository{
		Issues:        issueStore,
		Items:         itemStore,
		Subscribers:   subsStore,
		SocialPosts:   socialPostsStore,
		EmailEvents:   emailevents.New(conn),
		SocialMetrics: socialmetrics.New(conn),
		Metrics:       metricsstore.New(conn),
	}

	emailSender := email.New(config.ResendToken)
	slackClient := slack.New(config.SlackToken, config.SlackChannel)
	aiClient := ai.New(config, slackClient)

	aggregator, err := digestsvc.New(emailSender, config.EmailSendAddress, aiClient, slackClient, issueStore, repo.Items, subsStore)
	if err != nil {
		return nil, teardown, err
	}

	socialSvc, err := socialsvc.New(
		buildSocialPosters(config),
		aiClient,
		issueStore,
		repo.Items,
		socialPostsStore,
		slackClient,
	)
	if err != nil {
		return nil, teardown, err
	}
	socialSvc.WithCandidates(buildRotationCandidates(config, repo, socialPostsStore)...)

	subscriberSvc := audiencesvc.New(subsStore, issueStore, emailSender)

	return &App{
		Config:         &config,
		DB:             conn,
		Repository:     repo,
		Runner:         aggregator,
		Social:         socialSvc,
		Cache:          store,
		Subscribers:    subscriberSvc,
		EmailEvents:    svcengagement.NewEvents(repo.EmailEvents, subscriberSvc, itemStore, config.EmailSendAddress),
		Slack:          slackClient,
		MetricsService: svcengagement.New(repo.Metrics, slackClient),
		StatFetchers:   buildStatFetchers(config),
	}, teardown, nil
}

// buildSocialPosters returns the slice of social.Poster implementations
// whose credentials are present in the config. Each platform is opt-in:
// missing creds means the platform is skipped entirely.
func buildSocialPosters(c env.Config) []platform.Poster {
	var out []platform.Poster
	if c.BlueskyHandle != "" && c.BlueskyAppPassword != "" {
		out = append(out, bluesky.New(c.BlueskyHandle, c.BlueskyAppPassword))
	}
	if c.LinkedInOAuthToken != "" && c.LinkedInOrgURN != "" {
		out = append(out, linkedin.New(c.LinkedInOAuthToken, c.LinkedInOrgURN))
	}
	if c.MastodonServer != "" && c.MastodonAppToken != "" {
		out = append(out, mastodon.New(c.MastodonServer, c.MastodonAppToken))
	}
	return out
}

// buildRotationCandidates wires the four kinds the Tue/Fri rotation
// chooses from. The recap candidate is skipped if metrics aren't wired
// (would never happen in production but keeps tests/no-DB bootstraps
// from blowing up).
func buildRotationCandidates(_ env.Config, repo *Repository, posts social.PostRepository) []socialsvc.Candidate {
	out := make([]socialsvc.Candidate, 0, 4)

	out = append(out, candidates.NewNewSource(social.Profiles, posts))
	out = append(out, candidates.NewSpotlight(social.Profiles, posts))
	out = append(out, candidates.NewCTA(posts))
	out = append(out, candidates.NewCommunity(data.Conferences, data.Meetups, posts))

	if repo != nil && repo.Metrics != nil {
		if recapSvc, err := digestsvc.NewRecapService(repo.Metrics); err == nil {
			out = append(out, candidates.NewRecap(recapSvc, posts))
		}
	}
	return out
}

// buildStatFetchers returns a map of platform → StatFetcher for platforms
// whose credentials are present in the config.
func buildStatFetchers(c env.Config) map[social.Platform]platform.StatFetcher {
	out := make(map[social.Platform]platform.StatFetcher)
	if c.BlueskyHandle != "" && c.BlueskyAppPassword != "" {
		out[social.Bluesky] = bluesky.New(c.BlueskyHandle, c.BlueskyAppPassword)
	}
	if c.LinkedInOAuthToken != "" && c.LinkedInOrgURN != "" {
		out[social.LinkedIn] = linkedin.New(c.LinkedInOAuthToken, c.LinkedInOrgURN)
	}
	if c.MastodonServer != "" && c.MastodonAppToken != "" {
		out[social.Mastodon] = mastodon.New(c.MastodonServer, c.MastodonAppToken)
	}
	return out
}
