//go:build !serverless

// Package db — sqlite driver registration for local dev and tests.
// Excluded from serverless builds (build tag: serverless) because production
// always uses a remote Turso URL; the sqlite driver is never invoked there.
package db

import _ "modernc.org/sqlite"
