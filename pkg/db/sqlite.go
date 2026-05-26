// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !serverless

// Package db — sqlite driver registration for local dev and tests.
// Excluded from serverless builds (build tag: serverless) because production
// always uses a remote Turso URL; the sqlite driver is never invoked there.
package db

import _ "modernc.org/sqlite"
