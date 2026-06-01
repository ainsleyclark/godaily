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

	// Kinds optionally restricts PublishDrafts to draft rows whose Kind
	// matches one of these values. Empty means publish every kind. The
	// featured publish cron passes [PostKindFeatured]; the rotation
	// publish cron passes every non-featured kind, keeping the two cron
	// slots independent so a 15:00 run never accidentally promotes a
	// featured draft the 11:00 slot missed.
	Kinds []PostKind
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
