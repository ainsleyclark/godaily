// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import "time"

// IsWeekend reports whether t falls on a Saturday or Sunday.
func IsWeekend(t time.Time) bool {
	wd := t.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// IsRotationDay reports whether t falls on a social rotation day —
// Monday, Wednesday, or Friday.
//
// These mirror the rotation candidate rules in pkg/services/social
// (recap on Monday, community on Wednesday, new_source/spotlight/cta on
// Friday); every other day is a no-op. The rotation cron fires daily so
// BetterStack receives a heartbeat each day — this gate keeps the
// publish work to the days that actually have drafts.
func IsRotationDay(t time.Time) bool {
	switch t.Weekday() {
	case time.Monday, time.Wednesday, time.Friday:
		return true
	default:
		return false
	}
}
