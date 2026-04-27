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

// Package store provides typed access to the godaily database. It is the
// single home for SQL — both the sqlc-generated query methods and the
// hand-rolled domain helpers that need transactions or token generation.
package store

//go:generate sqlc generate -f ../../sqlc.yaml

import (
	"context"
	"database/sql"

	"github.com/ainsleydev/webkit/pkg/enforce"
	"github.com/pkg/errors"
)

// Store is the typed entry point for godaily persistence. It embeds the
// sqlc-generated *Queries so callers can invoke any generated query
// directly, and holds the underlying *sql.DB for transactional helpers.
type Store struct {
	*Queries
	db *sql.DB
}

// NewStore creates a new Store backed by db.
//
// The constructor is named NewStore (not New) because sqlc owns the
// package-level New(DBTX) *Queries symbol.
func NewStore(db *sql.DB) *Store {
	enforce.NotNil(db, "database connection is required")
	return &Store{
		Queries: New(db),
		db:      db,
	}
}

// Tx runs fn inside a database transaction. The transaction commits if fn
// returns nil and rolls back otherwise. Inside fn, callers should use the
// supplied *Queries — it is bound to the active transaction.
func (s *Store) Tx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "beginning transaction")
	}

	if err := fn(s.Queries.WithTx(tx)); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Wrapf(err, "rolling back: %v; original error", rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "committing transaction")
	}
	return nil
}
