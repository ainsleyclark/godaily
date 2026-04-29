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

package db

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"

	"github.com/ainsleydev/webkit/pkg/enforce"
	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Up applies any pending schema migrations to db using goose with the
// embedded migrations FS. Safe to call repeatedly; goose tracks applied
// versions in goose_db_version.
func Up(ctx context.Context, db *sql.DB) error {
	enforce.NotNil(db, "database connection is required")

	provider, err := newProvider(db)
	if err != nil {
		return err
	}

	if _, err = provider.Up(ctx); err != nil {
		return errors.Wrap(err, "applying migrations")
	}

	return nil
}

// Down rolls back the most recently applied migration. Intended for `make
// migrate-down` during development; production paths should use forward-only
// migrations.
func Down(ctx context.Context, db *sql.DB) error {
	enforce.NotNil(db, "database connection is required")

	provider, err := newProvider(db)
	if err != nil {
		return err
	}

	if _, err = provider.Down(ctx); err != nil {
		return errors.Wrap(err, "rolling back migration")
	}

	return nil
}

func newProvider(db *sql.DB) (*goose.Provider, error) {
	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return nil, errors.Wrap(err, "rooting migrations FS")
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, sub, goose.WithIsolateDDL(true))
	if err != nil {
		return nil, errors.Wrap(err, "creating goose provider")
	}

	return provider, nil
}
