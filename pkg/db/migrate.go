// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"context"
	"database/sql"
	"embed"

	"github.com/ainsleydev/webkit/pkg/enforce"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Up applies any pending schema migrations to db using goose with the
// embedded migrations FS. Safe to call repeatedly; goose tracks applied
// versions in goose_db_version.
func Up(ctx context.Context, db *sql.DB) error {
	enforce.NotNil(db, "database connection is required")
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
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.DownContext(ctx, db, "migrations")
}
