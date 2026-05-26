// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import "time"

// PostOptions controls a single Post invocation.
type PostOptions struct {
	// Date is the digest date — the issue slug is its UTC YYYY-MM-DD.
	Date time.Time

	// DryRun runs the full pipeline (DB read, AI calls, text generation)
	// but skips both platform HTTP and the social_posts insert.
	DryRun bool

	// Platforms optionally restricts which configured posters run. When
	// empty, every configured poster runs. Unknown platforms are ignored
	// with a log line.
	Platforms []Platform
}

// PostResult summarises one platform's outcome.
type PostResult struct {
	Platform Platform
	Kind     PostKind
	Text     string
	PostURL  string
	Err      error

	// Skipped is true when this platform was already posted for the same
	// idempotency key (issue or subject) on this run.
	Skipped bool
}

// RotateOptions controls a single Rotate invocation.
type RotateOptions struct {
	// Now is the wall clock used to pick the day's candidate list. Tuesday
	// runs the self_release/spotlight/cta rotation; Friday runs recap only.
	// Any other day is a no-op.
	Now time.Time

	// DryRun runs the candidate's full pipeline (eligibility + AI
	// generation) but skips platform HTTP and the social_posts insert.
	DryRun bool

	// Platforms optionally restricts which configured posters run.
	Platforms []Platform

	// ForceKind, when non-empty, bypasses the day-aware routing and runs
	// the named candidate's Eligible check directly. Used by the CLI to
	// test a specific kind out-of-band.
	ForceKind PostKind
}
