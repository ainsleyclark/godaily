// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"context"
	"database/sql"
	"embed"
	"sync"

	"github.com/ainsleydev/webkit/pkg/enforce"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// migrateMu serialises calls to Up/Down. goose configures its base FS and
// dialect via process-global state, so concurrent migrations (e.g. parallel
// tests each spinning up their own DB) would race on those globals. Holding
// the lock across the whole call keeps the globals consistent for the duration
// of the run.
var migrateMu sync.Mutex

// Up applies any pending schema migrations to db using goose with the
// embedded migrations FS. Safe to call repeatedly; goose tracks applied
// versions in goose_db_version.
func Up(ctx context.Context, db *sql.DB) error {
	enforce.NotNil(db, "database connection is required")
	migrateMu.Lock()
	defer migrateMu.Unlock()
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.UpContext(ctx, db, "migrations")
}

// Down rolls back the most recently applied migration. Intended for `make
// migrate-down` during development; production paths should use forward-only
// migrations.
func Down(ctx context.Context, db *sql.DB) error {
	enforce.NotNil(db, "database connection is required")
	migrateMu.Lock()
	defer migrateMu.Unlock()
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.DownContext(ctx, db, "migrations")
}
