package cmd

import (
	"context"
	"database/sql"

	"github.com/ainsleyclark/godaily/internal/db"
	"github.com/ainsleyclark/godaily/internal/env"
	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/store/issues"
	"github.com/ainsleyclark/godaily/internal/store/items"
	"github.com/ainsleyclark/godaily/internal/store/subscribers"
)

// App defines a global state for godaily.
type App struct {
	Config     *env.Config
	DB         *sql.DB
	Repository *Repository
}

// Repository defines the datastore for the application.,
type Repository struct {
	Issues      news.IssueRepository
	Items       news.ItemRepository
	Subscribers news.SubscriberRepository
}

// Bootstrap ties all the app dependencies together
// and returns a new App.
func Bootstrap(ctx context.Context) (*App, error) {
	config, err := env.New()
	if err != nil {
		return nil, err
	}
	conn, err := db.New(ctx, config.TursoURL, config.TursoAuthToken)
	if err != nil {
		return nil, err
	}
	return &App{
		Config: &config,
		DB:     conn,
		Repository: &Repository{
			Issues:      issues.New(conn),
			Items:       items.New(conn),
			Subscribers: subscribers.New(conn),
		},
	}, nil
}
