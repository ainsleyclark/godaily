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
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
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
	Config       *env.Config
	DB           *sql.DB
	Repository   *Repository
	Service      *Service
	Cache        cache.Store
	Slack        slack.Sender
	StatFetchers map[social.Platform]platform.StatFetcher
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
		Items:         items.NewCaching(itemStore, store),
		Subscribers:   subsStore,
		SocialPosts:   socialPostsStore,
		EmailEvents:   emailevents.New(conn),
		SocialMetrics: socialmetrics.New(conn),
		Metrics:       metricsstore.New(conn),
	}

	emailSender := email.New(config.ResendToken)
	slackClient := slack.New(config.SlackToken, config.SlackChannel)
	aiClient := ai.New(config)

	socialSvc, err := socialsvc.New(
		config,
		aiClient,
		issueStore,
		repo.Items,
		socialPostsStore,
		repo.Metrics,
		slackClient,
	)
	if err != nil {
		return nil, teardown, err
	}

	aggregator, err := digestsvc.New(emailSender, config.EmailSendAddress, aiClient, slackClient, issueStore, repo.Items, subsStore, socialSvc)
	if err != nil {
		return nil, teardown, err
	}

	subscriberSvc := audiencesvc.New(subsStore, issueStore, emailSender)

	return &App{
		Config:     &config,
		DB:         conn,
		Repository: repo,
		Service: &Service{
			Digest:      aggregator,
			Social:      socialSvc,
			Subscribers: subscriberSvc,
			Events:      svcengagement.NewEvents(repo.EmailEvents, subscriberSvc, itemStore, config.EmailSendAddress),
			Metrics:     svcengagement.New(repo.Metrics, slackClient),
		},
		Cache:        store,
		Slack:        slackClient,
		StatFetchers: socialSvc.StatFetchers(),
	}, teardown, nil
}
