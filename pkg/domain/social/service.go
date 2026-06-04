// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import "context"

//go:generate go run go.uber.org/mock/mockgen -package=mocksocial -destination=../../mocks/social/Service.go . Service

// Service drives the social media pipeline. Every post — featured and
// rotation — flows through the same generate-then-publish lifecycle so a
// human can review (and edit) drafts before they go live:
//
//  1. DraftAll runs at digest Build time (02:00). It picks the day's
//     featured item, reframes for each configured platform, and writes a
//     draft row per platform to social_posts. On rotation days
//     (Mon/Wed/Fri) it also generates the rotation post for that day as
//     a draft.
//  2. PublishDrafts runs at the publish cron (11:00). It walks every
//     draft row regardless of kind, posts each to its platform, and
//     updates the row to published (or error).
type Service interface {
	// DraftAll generates and persists every social draft owed for the
	// day: one featured draft per configured platform, plus the
	// rotation draft for the day's weekday (recap on Mon, community on
	// Wed, new_source/spotlight/cta on Fri). No platform send happens
	// here. Safe to re-run for the same date: existing drafts are
	// cleared first.
	DraftAll(ctx context.Context, opts PostOptions) ([]PostResult, error)

	// DraftFeatured drafts only the featured post for opts.Date — one row
	// per configured platform — without publishing. Used by the CLI to
	// regenerate a featured draft independently of DraftAll.
	DraftFeatured(ctx context.Context, opts PostOptions) ([]PostResult, error)

	// DraftRotation drafts only the eligible rotation post for opts.Date's
	// weekday, without publishing. Used by the CLI to regenerate a rotation
	// draft independently of DraftAll. Returns nil when no candidate is
	// eligible for the day.
	DraftRotation(ctx context.Context, opts PostOptions) ([]PostResult, error)

	// PublishDrafts loads every draft row (any kind) and publishes each
	// to its platform, transitioning the rows to published (or error on
	// per-platform failures). Cancelled rows are skipped.
	PublishDrafts(ctx context.Context, opts PostOptions) ([]PostResult, error)
}
