// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import "context"

//go:generate go run go.uber.org/mock/mockgen -package=mocksocial -destination=../../mocks/social/Service.go . Service

// Service publishes social media posts for the daily featured slot and the
// Tue/Fri rotation slot. Implementations decide which platforms to target
// based on the credentials they were wired with at construction.
type Service interface {
	// Post runs the daily featured slot for the issue dated opts.Date.
	Post(ctx context.Context, opts PostOptions) ([]PostResult, error)

	// Rotate runs the day-aware rotation slot (recap, spotlight, cta,
	// self_release, community) for the wall clock in opts.Now.
	Rotate(ctx context.Context, opts RotateOptions) ([]PostResult, error)
}
