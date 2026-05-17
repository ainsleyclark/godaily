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
	"github.com/ainsleyclark/godaily/pkg/digest"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/news"
	_ "github.com/ainsleyclark/godaily/pkg/source" // registers all fetchers via init()
	"github.com/ainsleyclark/godaily/pkg/store/issues"
	"github.com/ainsleyclark/godaily/pkg/store/items"
	"github.com/ainsleyclark/godaily/pkg/store/subscribers"
	"github.com/ainsleyclark/godaily/pkg/subscriber"
	"github.com/ainsleydev/webkit/pkg/cache"
)

// App defines a global state for godaily.
type App struct {
	Config      *env.Config
	DB          *sql.DB
	Repository  *Repository
	Runner      digest.Runner
	Cache       cache.Store
	Subscribers subscriber.Subscriber
	Slack       slack.Sender
}

// Repository defines the datastore for the application.
type Repository struct {
	Issues      news.IssueRepository
	Items       news.ItemRepository
	Subscribers news.SubscriberRepository
}

// Bootstrap ties all the app dependencies together
// and returns a new App.
func Bootstrap(ctx context.Context) (*App, func(), error) {
	config, err := env.New(ctx)
	if err != nil {
		return nil, func() {}, err
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

	cachedIssues := issues.NewCaching(issueStore, store)

	subsStore := subscribers.New(conn)

	repo := &Repository{
		Issues:      cachedIssues,
		Items:       items.New(conn),
		Subscribers: subsStore,
	}

	emailSender := email.New(config.ResendToken)
	slackClient := slack.New(config.SlackToken, config.SlackChannel)

	aiClient := ai.New(config)

	aggregator, err := digest.New(emailSender, config.EmailSendAddress, aiClient, slackClient, issueStore, repo.Items, subsStore)
	if err != nil {
		return nil, teardown, err
	}

	return &App{
		Config:      &config,
		DB:          conn,
		Repository:  repo,
		Runner:      aggregator,
		Cache:       store,
		Subscribers: subscriber.New(subsStore, cachedIssues, emailSender),
		Slack:       slackClient,
	}, teardown, nil
}
