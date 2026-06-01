// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import "context"

//go:generate go run go.uber.org/mock/mockgen -package=mocksocial -destination=../../mocks/social/Service.go . Service

// Service drives both legs of the daily featured pipeline and the
// Tue/Wed/Fri rotation.
//
// The featured pipeline is split in two so a human can review (and in
// future edit) drafts between them:
//
//  1. DraftFeatured runs at digest Build time (02:00). It picks the day's
//     featured item, reframes for each configured platform, and writes a
//     draft row per platform to social_posts.
//  2. PublishDrafts runs at the social cron (11:00). It loads today's
//     drafts, posts each to its platform, and updates the row to
//     published (or error).
//
// Rotation stays a single-shot generate+publish call because it depends
// on late-day data (Friday recap clicks) and same-day candidate
// eligibility.
type Service interface {
	// DraftFeatured generates and persists draft featured posts for the
	// issue dated opts.Date, one per configured platform. No platform send
	// happens here. Safe to re-run for the same date: existing drafts
	// for that issue are cleared first.
	DraftFeatured(ctx context.Context, opts PostOptions) ([]PostResult, error)

	// PublishDrafts loads today's draft featured posts and publishes
	// each to its platform, transitioning the rows to published (or
	// error on per-platform failures).
	PublishDrafts(ctx context.Context, opts PostOptions) ([]PostResult, error)

	// Rotate runs the day-aware rotation slot (recap, spotlight, cta,
	// self_release, community) for the wall clock in opts.Now.
	Rotate(ctx context.Context, opts RotateOptions) ([]PostResult, error)
}
