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

import (
	"hash/fnv"
	"time"
)

// CronSlotsPerHour is the number of 10-minute cron invocation windows in the
// 11:00 UTC hour. vercel.json declares the matching cron: `0,10,20,30,40,50 11 * * 1-5`.
// PickSlot hashes each date to exactly one of these slots so only one
// invocation per day does real work regardless of which endpoint fires.
const CronSlotsPerHour = 6

// PickSlot returns the 10-minute slot (0..SlotsPerHour-1) that should
// actually post for the given date. The result is stable across retries
// on the same day, so a cron invocation in the wrong slot can safely
// short-circuit while the right slot does the work.
//
// The slot is derived deterministically from the date so the time of
// posting jitters day-to-day without persisting state.
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
