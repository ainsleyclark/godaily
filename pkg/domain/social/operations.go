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
