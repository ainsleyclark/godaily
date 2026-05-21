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

package godaily

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/db"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	socialgw "github.com/ainsleyclark/godaily/pkg/gateway/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/social/bluesky"
	"github.com/ainsleyclark/godaily/pkg/gateway/social/linkedin"
	"github.com/ainsleyclark/godaily/pkg/gateway/social/mastodon"
	"github.com/ainsleyclark/godaily/pkg/services/digest"
	"github.com/ainsleyclark/godaily/pkg/services/emailevent"
	"github.com/ainsleyclark/godaily/pkg/services/social"
	"github.com/ainsleyclark/godaily/pkg/services/subscriber"
	_ "github.com/ainsleyclark/godaily/pkg/source" // registers all fetchers via init()
	"github.com/ainsleyclark/godaily/pkg/store/emailevents"
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
	Config       *env.Config
	DB           *sql.DB
	Repository   *Repository
	Runner       digest.Runner
	Social       *social.Service
	Cache        cache.Store
	Subscribers  subscriber.Subscriber
	EmailEvents  *emailevent.Service
	Slack        slack.Sender
	StatFetchers map[socialgw.Platform]socialgw.StatFetcher
}

// Repository defines the datastore for the application.
type Repository struct {
	Issues        news.IssueRepository
	Items         news.ItemRepository
	Subscribers   news.SubscriberRepository
	SocialPosts   news.SocialPostRepository
	EmailEvents   engagement.EmailEventRepository
	SocialMetrics engagement.SocialMetricRepository
}

// Bootstrap ties all the app dependencies together
// and returns a new App.
func Bootstrap(ctx context.Context) (*App, func(), error) {
	config, err := env.New(ctx)
	if err != nil {
		return nil, func() {}, err
	}

	if err = news.Materialise(config); err != nil {
		return nil, func() {}, errors.Wrap(err, "materialising sources")
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

	repo := &Repository{
		Issues:        issueStore,
		Items:         items.New(conn),
		Subscribers:   subsStore,
		SocialPosts:   socialPostsStore,
		EmailEvents:   emailevents.New(conn),
		SocialMetrics: socialmetrics.New(conn),
	}

	emailSender := email.New(config.ResendToken)
	slackClient := slack.New(config.SlackToken, config.SlackChannel)
	aiClient := ai.New(config, slackClient)

	aggregator, err := digest.New(emailSender, config.EmailSendAddress, aiClient, slackClient, issueStore, repo.Items, subsStore)
	if err != nil {
		return nil, teardown, err
	}

	socialSvc, err := social.New(
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

	subscriberSvc := subscriber.New(subsStore, issueStore, emailSender)

	return &App{
		Config:       &config,
		DB:           conn,
		Repository:   repo,
		Runner:       aggregator,
		Social:       socialSvc,
		Cache:        store,
		Subscribers:  subscriberSvc,
		EmailEvents:  emailevent.New(repo.EmailEvents, subscriberSvc),
		Slack:        slackClient,
		StatFetchers: buildStatFetchers(config),
	}, teardown, nil
}

// buildSocialPosters returns the slice of social.Poster implementations
// whose credentials are present in the config. Each platform is opt-in:
// missing creds means the platform is skipped entirely.
func buildSocialPosters(c env.Config) []socialgw.Poster {
	var out []socialgw.Poster
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

// buildStatFetchers returns a map of platform → StatFetcher for platforms
// whose credentials are present in the config.
func buildStatFetchers(c env.Config) map[socialgw.Platform]socialgw.StatFetcher {
	out := make(map[socialgw.Platform]socialgw.StatFetcher)
	if c.BlueskyHandle != "" && c.BlueskyAppPassword != "" {
		out[socialgw.PlatformBluesky] = bluesky.New(c.BlueskyHandle, c.BlueskyAppPassword)
	}
	if c.LinkedInOAuthToken != "" && c.LinkedInOrgURN != "" {
		out[socialgw.PlatformLinkedIn] = linkedin.New(c.LinkedInOAuthToken, c.LinkedInOrgURN)
	}
	if c.MastodonServer != "" && c.MastodonAppToken != "" {
		out[socialgw.PlatformMastodon] = mastodon.New(c.MastodonServer, c.MastodonAppToken)
	}
	return out
}
