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

// Package backup exports the database as a gzip-compressed SQL dump and
// optionally uploads it to Backblaze B2 via the S3-compatible API.
package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
)

// Export dumps all application tables from db as a gzip-compressed SQL file.
// The returned bytes can be written to disk or uploaded directly.
func Export(ctx context.Context, db *sql.DB) ([]byte, error) {
	tables, err := listTables(ctx, db)
	if err != nil {
		return nil, err
	}

	var sqlBuf bytes.Buffer
	sqlBuf.WriteString("BEGIN;\n\n")

	for _, table := range tables {
		schema, err := tableSchema(ctx, db, table)
		if err != nil {
			return nil, errors.Wrapf(err, "fetching schema for table %q", table)
		}
		sqlBuf.WriteString(schema)
		sqlBuf.WriteString(";\n\n")

		if err = tableInserts(ctx, db, table, &sqlBuf); err != nil {
			return nil, errors.Wrapf(err, "dumping rows for table %q", table)
		}
	}

	sqlBuf.WriteString("COMMIT;\n")

	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	if _, err = w.Write(sqlBuf.Bytes()); err != nil {
		return nil, errors.Wrap(err, "gzip write")
	}
	if err = w.Close(); err != nil {
		return nil, errors.Wrap(err, "gzip close")
	}

	return gz.Bytes(), nil
}

// Upload sends data to the configured Backblaze B2 bucket using the
// S3-compatible API. filename is the object key inside the bucket.
func Upload(ctx context.Context, cfg env.Config, filename string, data []byte) error {
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(cfg.B2Endpoint),
		Region:       "auto",
		Credentials:  aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.B2KeyID, cfg.B2AppKey, "")),
		// Force path-style addressing required by B2.
		UsePathStyle: true,
	})

	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.B2BucketName),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/gzip"),
	})
	return errors.Wrap(err, "uploading backup to B2")
}

// listTables returns application table names, excluding SQLite internals and
// the Goose migration tracking table.
func listTables(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT name FROM sqlite_master
		 WHERE type = 'table'
		   AND name NOT LIKE 'sqlite_%'
		   AND name != 'goose_db_version'
		 ORDER BY name`)
	if err != nil {
		return nil, errors.Wrap(err, "querying sqlite_master")
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

// tableSchema returns the CREATE TABLE statement for the given table.
func tableSchema(ctx context.Context, db *sql.DB, table string) (string, error) {
	var schema string
	err := db.QueryRowContext(ctx,
		`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`, table,
	).Scan(&schema)
	return schema, errors.Wrap(err, "scanning schema")
}

// tableInserts writes INSERT statements for every row in table into buf.
func tableInserts(ctx context.Context, db *sql.DB, table string, buf *bytes.Buffer) error {
	//nolint:gosec // table name comes from sqlite_master, not user input.
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`SELECT * FROM "%s"`, table))
	if err != nil {
		return errors.Wrap(err, "selecting rows")
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return errors.Wrap(err, "fetching columns")
	}

	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err = rows.Scan(ptrs...); err != nil {
			return errors.Wrap(err, "scanning row")
		}

		placeholders := make([]string, len(vals))
		for i, v := range vals {
			placeholders[i] = sqlLiteral(v)
		}

		fmt.Fprintf(buf, "INSERT INTO %q VALUES (%s);\n", table, strings.Join(placeholders, ", "))
	}

	if rows.Err() != nil {
		return errors.Wrap(rows.Err(), "iterating rows")
	}

	buf.WriteByte('\n')
	return nil
}

// sqlLiteral converts a Go value into its SQLite literal representation.
func sqlLiteral(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "1"
		}
		return "0"
	case []byte:
		return fmt.Sprintf("X'%X'", val)
	case string:
		// Escape single quotes by doubling them.
		return fmt.Sprintf("'%s'", strings.ReplaceAll(val, "'", "''"))
	default:
		return fmt.Sprintf("'%v'", v)
	}
}
