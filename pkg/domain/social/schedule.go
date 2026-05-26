// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"hash/fnv"
	"time"
)

// CronSlotsPerHour is the number of 10-minute cron invocation windows in the
// 11:00 UTC hour. vercel.json declares the matching cron: `0,10,20,30,40,50 11 * * 1-5`.
// PickSlot hashes each date to exactly one of these slots so only one
// invocation per day does real work regardless of which endpoint fires.
const CronSlotsPerHour = 6

// PickSlot returns the 10-minute slot (0..CronSlotsPerHour-1) that should
// actually post for the given date. The result is stable across retries
// on the same day, so a cron invocation in the wrong slot can safely
// short-circuit while the right slot does the work.
func PickSlot(date time.Time) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(date.UTC().Format("2006-01-02")))
	return int(h.Sum32() % CronSlotsPerHour)
}

// ShouldRun reports whether the current minute matches the slot picked for
// the given date. Returns false when now's minute falls outside any slot
// boundary, so an unexpected invocation is a safe no-op.
func ShouldRun(now, date time.Time) bool {
	return now.UTC().Minute()/10 == PickSlot(date)
}
